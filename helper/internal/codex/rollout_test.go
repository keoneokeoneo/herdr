package codex

import (
	"bufio"
	"os"
	"reflect"
	"testing"
	"time"

	"herdr-poke/internal/state"
)

func TestCtxPct(t *testing.T) {
	const w = 258400
	for _, tc := range []struct{ tokens, want int }{
		{0, 0},       // below the 12k baseline: nothing used
		{12000, 0},   // exactly the baseline
		{135200, 50}, // half of the effective window
		{258400, 100},
		{300000, 100}, // over the window clamps
	} {
		if got := CtxPct(w, tc.tokens); got != tc.want {
			t.Errorf("CtxPct(%d, %d) = %d, want %d", w, tc.tokens, got, tc.want)
		}
	}
	// A window no bigger than the baseline has no effective room.
	if got := CtxPct(12000, 5); got != 100 {
		t.Errorf("CtxPct(12000, 5) = %d, want 100", got)
	}
	if got := CtxPct(0, 5); got != 100 {
		t.Errorf("CtxPct(0, 5) = %d, want 100", got)
	}
}

func applyFile(t *testing.T, path string) Snapshot {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var s Snapshot
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		Apply(&s, sc.Bytes())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestApplyRealSample(t *testing.T) {
	s := applyFile(t, "testdata/rollout-gpt6-astra.jsonl")

	if s.Model != "gpt-6-astra" {
		t.Errorf("Model = %q, want gpt-6-astra", s.Model)
	}
	if s.Effort != "medium" {
		t.Errorf("Effort = %q, want medium", s.Effort)
	}
	// Last token_count of the sample: 71152 tokens in a 258400 window.
	if !s.HasCtx || s.CtxPct != 24 {
		t.Errorf("ctx = %d (has %v), want 24", s.CtxPct, s.HasCtx)
	}
	if !s.HasLimits {
		t.Fatal("HasLimits = false")
	}
	want := []state.Window{{Minutes: 10080, Used: 39.0, ResetsAt: 1793780927}}
	if !reflect.DeepEqual(s.Limits.Windows, want) {
		t.Errorf("Windows = %+v, want %+v (secondary is null in the sample)", s.Limits.Windows, want)
	}
	if s.Limits.Plan != "prolite" {
		t.Errorf("Plan = %q, want prolite", s.Limits.Plan)
	}
	// Updated is the event's own time, not the wall clock.
	if want := time.Date(2026, 11, 3, 5, 59, 40, 0, time.UTC).Unix(); s.Limits.Updated != want {
		t.Errorf("Updated = %d, want %d (last token_count's own time)", s.Limits.Updated, want)
	}
}

func TestApplyNullInfo(t *testing.T) {
	var s Snapshot
	Apply(&s, []byte(`{"timestamp":"2026-11-03T05:57:22.647Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"limit_id":"codex","primary":{"used_percent":12.5,"window_minutes":10080,"resets_at":1793780927},"secondary":null,"plan_type":"prolite"}}}`))
	if s.HasCtx || s.CtxPct != 0 {
		t.Errorf("null info set ctx: %d (has %v)", s.CtxPct, s.HasCtx)
	}
	// The limits in the same event still count.
	if !s.HasLimits || len(s.Limits.Windows) != 1 || s.Limits.Windows[0].Used != 12.5 {
		t.Errorf("limits = %+v (has %v)", s.Limits, s.HasLimits)
	}
}

func TestApplyNullRateLimitsKeepsPrevious(t *testing.T) {
	var s Snapshot
	Apply(&s, []byte(`{"timestamp":"2026-11-03T05:57:22.647Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":12.5,"window_minutes":10080,"resets_at":1793780927},"secondary":null,"plan_type":"prolite"}}}`))
	Apply(&s, []byte(`{"timestamp":"2026-11-03T05:58:00.000Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":null}}`))
	if !s.HasLimits || len(s.Limits.Windows) != 1 || s.Limits.Windows[0].Used != 12.5 {
		t.Errorf("limits lost after a null rate_limits: %+v (has %v)", s.Limits, s.HasLimits)
	}
}

func TestApplyBothWindows(t *testing.T) {
	var s Snapshot
	Apply(&s, []byte(`{"timestamp":"2026-11-03T05:57:22.647Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":7.0,"window_minutes":300,"resets_at":1793700000},"secondary":{"used_percent":41.0,"window_minutes":10080,"resets_at":1793780927},"plan_type":"pro"}}}`))
	want := []state.Window{
		{Minutes: 300, Used: 7.0, ResetsAt: 1793700000},
		{Minutes: 10080, Used: 41.0, ResetsAt: 1793780927},
	}
	if !reflect.DeepEqual(s.Limits.Windows, want) || s.Limits.Plan != "pro" {
		t.Errorf("limits = %+v", s.Limits)
	}
}

func TestApplyWindowFromTaskStarted(t *testing.T) {
	var s Snapshot
	Apply(&s, []byte(`{"type":"event_msg","payload":{"type":"task_started","model_context_window":258400}}`))
	// Older Codex builds leave model_context_window out of info.
	Apply(&s, []byte(`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":135200}}}}`))
	if !s.HasCtx || s.CtxPct != 50 {
		t.Errorf("ctx = %d (has %v), want 50", s.CtxPct, s.HasCtx)
	}
}

func TestApplyIgnoresGarbage(t *testing.T) {
	var s Snapshot
	for _, line := range []string{
		``, `not json`, `{}`, `[]`, `{"type":"event_msg"}`, `{"type":"event_msg","payload":7}`,
		`{"type":"turn_context","payload":"x"}`, `{"type":"event_msg","payload":{"type":"token_count","info":5}}`,
		`{"type":"response_item","payload":{"type":"message","text":"token_count turn_context"}}`,
	} {
		Apply(&s, []byte(line))
	}
	if !reflect.DeepEqual(s, Snapshot{}) {
		t.Errorf("garbage changed the snapshot: %+v", s)
	}
}

func TestSessionMetaFilters(t *testing.T) {
	const plain = `{"timestamp":"2026-11-03T05:57:10.900Z","type":"session_meta","payload":{"id":"aaaa","cwd":"/work/demo","originator":"codex-tui","timestamp":"2026-11-03T05:57:10.883Z","source":"cli"}}`
	m, ok := SessionMeta([]byte(plain))
	if !ok {
		t.Fatal("plain session_meta rejected")
	}
	wantAt := time.Date(2026, 11, 3, 5, 57, 10, 883_000_000, time.UTC)
	if m.ID != "aaaa" || m.Cwd != "/work/demo" || m.Originator != "codex-tui" || !m.Timestamp.Equal(wantAt) {
		t.Errorf("meta = %+v", m)
	}
	if m.ParentThreadID != "" || m.ForkedFromID != "" {
		t.Errorf("plain session has parent/fork: %+v", m)
	}

	fork, ok := SessionMeta([]byte(`{"type":"session_meta","payload":{"id":"bbbb","forked_from_id":"aaaa","cwd":"/work/demo","timestamp":"2026-11-03T05:57:10.883Z"}}`))
	if !ok || fork.ForkedFromID != "aaaa" {
		t.Errorf("fork meta = %+v ok=%v", fork, ok)
	}

	parent, ok := SessionMeta([]byte(`{"type":"session_meta","payload":{"id":"cccc","parent_thread_id":"aaaa","cwd":"/work/demo","timestamp":"2026-11-03T05:57:10.883Z"}}`))
	if !ok || parent.ParentThreadID != "aaaa" {
		t.Errorf("parent meta = %+v ok=%v", parent, ok)
	}

	// Sub-agents record their parent under source.subagent.thread_spawn.
	sub, ok := SessionMeta([]byte(`{"type":"session_meta","payload":{"id":"dddd","cwd":"/work/demo","timestamp":"2026-11-03T05:57:10.883Z","source":{"subagent":{"thread_spawn":{"parent_thread_id":"aaaa","depth":1}}}}}`))
	if !ok || sub.ParentThreadID != "aaaa" {
		t.Errorf("sub-agent meta = %+v ok=%v", sub, ok)
	}

	for _, line := range []string{
		``, `garbage`, `{"type":"turn_context","payload":{"model":"x"}}`,
		`{"type":"session_meta","payload":{"cwd":"/work/demo"}}`, // no id
		`{"type":"session_meta","payload":9}`,
	} {
		if m, ok := SessionMeta([]byte(line)); ok {
			t.Errorf("SessionMeta(%q) = %+v, want rejected", line, m)
		}
	}

	// A fixture file's first line is a valid meta.
	f, err := os.Open("testdata/rollout-gpt6-astra.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	sc.Scan()
	if m, ok := SessionMeta(sc.Bytes()); !ok || m.Cwd != "/work/demo" || m.Originator != "codex-tui" {
		t.Errorf("fixture meta = %+v ok=%v", m, ok)
	}
}
