package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"herdr-poke/internal/state"
)

var kst = time.FixedZone("KST", 9*3600)

// noon is "now" for most tests: 2026-11-03 12:00 KST.
var noon = time.Date(2026, 11, 3, 12, 0, 0, 0, kst)

type env struct {
	t      *testing.T
	claude string
	codex  string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	return &env{t: t, claude: t.TempDir(), codex: t.TempDir()}
}

func (e *env) roots() Roots { return Roots{ClaudeProjects: e.claude, CodexSessions: e.codex} }

func (e *env) run(now time.Time) {
	e.t.Helper()
	if err := Run(now, e.roots()); err != nil {
		e.t.Fatalf("Run: %v", err)
	}
}

// ts renders an instant the way the logs do: UTC, millisecond precision.
func ts(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func claudeLine(id string, at time.Time, in, out, cacheWrite, cacheRead int) string {
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"requestId":"req_%s","message":{"id":%q,"role":"assistant","model":"m","content":[{"type":"text","text":""}],"usage":{"input_tokens":%d,"output_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d}}}`+"\n",
		ts(at), id, id, in, out, cacheWrite, cacheRead)
}

func codexLine(responseID string, at time.Time, total int) string {
	return fmt.Sprintf(`{"timestamp":%q,"type":"token_usage_record","payload":{"response_id":%q,"usage":{"input_tokens":%d,"output_tokens":0,"total_tokens":%d}}}`+"\n",
		ts(at), responseID, total, total)
}

func (e *env) write(root, rel string, lines ...string) string {
	e.t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "")), 0o644); err != nil {
		e.t.Fatal(err)
	}
	return path
}

func appendTo(t *testing.T, path string, lines ...string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(strings.Join(lines, "")); err != nil {
		t.Fatal(err)
	}
}

type out struct {
	Days []struct {
		Date   string `json:"date"`
		Claude int    `json:"claude"`
		Codex  int    `json:"codex"`
	} `json:"days"`
	Updated int64 `json:"updated"`
}

func readUsage(t *testing.T) out {
	t.Helper()
	var o out
	if err := state.ReadUsageJSON("daily.json", &o); err != nil {
		t.Fatal(err)
	}
	return o
}

// day returns the totals of one date, failing if it is not in the output.
func (o out) day(t *testing.T, date string) (claude, codex int) {
	t.Helper()
	for _, d := range o.Days {
		if d.Date == date {
			return d.Claude, d.Codex
		}
	}
	t.Fatalf("date %s not in %+v", date, o.Days)
	return
}

func TestClaudeDedupe(t *testing.T) {
	e := newEnv(t)
	at := noon.Add(-time.Hour)
	// One message logged three times while streaming: the output count grows.
	e.write(e.claude, "proj/s1.jsonl",
		claudeLine("msg_1", at, 2, 10, 100, 1000),
		claudeLine("msg_1", at.Add(time.Second), 2, 10, 100, 1000),
		claudeLine("msg_1", at.Add(2*time.Second), 2, 25, 100, 1000),
		claudeLine("msg_2", at, 1, 1, 1, 1),
	)
	// A resumed session copies the message into another file.
	e.write(e.claude, "proj/s2.jsonl", claudeLine("msg_1", at, 2, 25, 100, 1000))

	e.run(noon)
	c, _ := readUsage(t).day(t, "2026-11-03")
	if want := (2 + 25 + 100 + 1000) + 4; c != want {
		t.Errorf("claude today = %d, want %d (msg_1 once at its largest, plus msg_2)", c, want)
	}
}

func TestClaudeCountsSubagentsAndIgnoresOtherLines(t *testing.T) {
	e := newEnv(t)
	at := noon.Add(-time.Hour)
	e.write(e.claude, "proj/s1/subagents/agent-a.jsonl", claudeLine("msg_sub", at, 5, 5, 5, 5))
	e.write(e.claude, "proj/s1.jsonl",
		`{"type":"user","timestamp":"`+ts(at)+`","message":{"id":"msg_u","usage":{"input_tokens":999}}}`+"\n",
		"not json at all\n",
		`{"type":"assistant","timestamp":"`+ts(at)+`","message":{"role":"assistant"}}`+"\n", // no usage
		`{"type":"assistant","timestamp":"garbage","message":{"id":"msg_x","usage":{"input_tokens":7}}}`+"\n",
		"\n",
	)
	e.write(e.claude, "proj/notes.txt", claudeLine("msg_txt", at, 1000, 0, 0, 0))

	e.run(noon)
	if c, _ := readUsage(t).day(t, "2026-11-03"); c != 20 {
		t.Errorf("claude today = %d, want 20 (only the subagent message)", c)
	}
}

func TestCodexDedupe(t *testing.T) {
	e := newEnv(t)
	at := noon.Add(-time.Hour)
	e.write(e.codex, "2026/11/03/rollout-a.jsonl",
		codexLine("resp_1", at, 1000),
		codexLine("resp_1", at.Add(time.Second), 1000),
		codexLine("resp_2", at, 500),
		`{"timestamp":"`+ts(at)+`","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":99999}}}}`+"\n",
	)
	// A fork replays the parent's history, with later timestamps.
	e.write(e.codex, "2026/11/03/rollout-b.jsonl", codexLine("resp_1", at.Add(time.Minute), 1000), codexLine("resp_3", at, 7))

	e.run(noon)
	_, x := readUsage(t).day(t, "2026-11-03")
	if x != 1507 {
		t.Errorf("codex today = %d, want 1507", x)
	}
}

func TestFourteenDaysZeroFilled(t *testing.T) {
	e := newEnv(t)
	e.write(e.claude, "p/a.jsonl",
		claudeLine("m_today", noon.Add(-time.Hour), 1, 0, 0, 0),
		claudeLine("m_oldest", noon.AddDate(0, 0, -13), 3, 0, 0, 0),
		claudeLine("m_outside", noon.AddDate(0, 0, -14), 1000, 0, 0, 0),
		claudeLine("m_future", noon.AddDate(0, 0, 1), 1000, 0, 0, 0),
	)
	e.write(e.codex, "2026/10/25/rollout-a.jsonl", codexLine("r1", noon.AddDate(0, 0, -9), 40))

	e.run(noon)
	o := readUsage(t)
	if len(o.Days) != 14 {
		t.Fatalf("%d days, want 14", len(o.Days))
	}
	if o.Days[0].Date != "2026-10-21" || o.Days[13].Date != "2026-11-03" {
		t.Errorf("range %s..%s, want 2026-10-21..2026-11-03 (oldest first)", o.Days[0].Date, o.Days[13].Date)
	}
	for i, d := range o.Days {
		wantC, wantX := 0, 0
		switch d.Date {
		case "2026-11-03":
			wantC = 1
		case "2026-10-21":
			wantC = 3
		case "2026-10-25":
			wantX = 40
		}
		if d.Claude != wantC || d.Codex != wantX {
			t.Errorf("day %d %s = {claude %d, codex %d}, want {%d, %d}", i, d.Date, d.Claude, d.Codex, wantC, wantX)
		}
	}
	if o.Updated != noon.Unix() {
		t.Errorf("updated = %d, want %d", o.Updated, noon.Unix())
	}
}

func TestMissingRootsStillWriteZeroDays(t *testing.T) {
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	if err := Run(noon, Roots{ClaudeProjects: "/nonexistent/claude", CodexSessions: "/nonexistent/codex"}); err != nil {
		t.Fatal(err)
	}
	o := readUsage(t)
	if len(o.Days) != 14 {
		t.Fatalf("%d days", len(o.Days))
	}
	for _, d := range o.Days {
		if d.Claude != 0 || d.Codex != 0 {
			t.Errorf("non-zero day %+v", d)
		}
	}
}

func TestIncrementalOnlyReadsNew(t *testing.T) {
	e := newEnv(t)
	at := noon.Add(-2 * time.Hour)
	first := claudeLine("msg_1", at, 10, 0, 0, 0)
	path := e.write(e.claude, "p/s.jsonl", first)
	codexPath := e.write(e.codex, "2026/11/03/rollout-a.jsonl", codexLine("r1", at, 100))
	e.run(noon)
	if c, x := readUsage(t).day(t, "2026-11-03"); c != 10 || x != 100 {
		t.Fatalf("after first run: claude %d codex %d", c, x)
	}

	var offsets map[string]struct{ Size, Offset int64 }
	if err := state.ReadJSON("usage-offsets.json", &offsets); err != nil {
		t.Fatal(err)
	}
	if got := offsets[path]; got.Size != int64(len(first)) || got.Offset != int64(len(first)) {
		t.Errorf("offsets[%s] = %+v, want size and offset %d", path, got, len(first))
	}

	// Rewrite the first line in place (same length) with a bigger count: only a
	// re-read could notice. Then append a new message.
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteAt([]byte(strings.Replace(first, `"input_tokens":10`, `"input_tokens":99`, 1)), 0)
	f.Close()
	appendTo(t, path, claudeLine("msg_2", at, 5, 0, 0, 0))
	appendTo(t, codexPath, codexLine("r2", at, 7))

	e.run(noon.Add(time.Minute))
	if c, x := readUsage(t).day(t, "2026-11-03"); c != 15 || x != 107 {
		t.Errorf("after second run: claude %d (want 15: 10 + 5), codex %d (want 107)", c, x)
	}
}

func TestPartialLineWaitsForItsNewline(t *testing.T) {
	e := newEnv(t)
	at := noon.Add(-time.Hour)
	done := claudeLine("msg_1", at, 1, 0, 0, 0)
	half := claudeLine("msg_2", at, 20, 0, 0, 0)
	path := e.write(e.claude, "p/s.jsonl", done, half[:50])
	e.run(noon)
	if c, _ := readUsage(t).day(t, "2026-11-03"); c != 1 {
		t.Fatalf("claude = %d, want 1 while msg_2 is half written", c)
	}
	appendTo(t, path, half[50:])
	e.run(noon)
	if c, _ := readUsage(t).day(t, "2026-11-03"); c != 21 {
		t.Errorf("claude = %d, want 21 once the line is complete", c)
	}
	e.run(noon)
	if c, _ := readUsage(t).day(t, "2026-11-03"); c != 21 {
		t.Errorf("claude = %d after a no-op run, want 21 (nothing counted twice)", c)
	}
}

func TestLastLineWithoutNewlineCountsOnce(t *testing.T) {
	e := newEnv(t)
	at := noon.Add(-time.Hour)
	line := claudeLine("msg_1", at, 8, 0, 0, 0)
	path := e.write(e.claude, "p/s.jsonl", strings.TrimSuffix(line, "\n")) // complete record, newline not written yet
	e.run(noon)
	e.run(noon)
	if c, _ := readUsage(t).day(t, "2026-11-03"); c != 8 {
		t.Errorf("claude = %d, want 8", c)
	}
	appendTo(t, path, "\n", claudeLine("msg_2", at, 2, 0, 0, 0))
	e.run(noon)
	if c, _ := readUsage(t).day(t, "2026-11-03"); c != 10 {
		t.Errorf("claude = %d, want 10", c)
	}
}

func TestShrunkFileIsReadAgainWithoutDoubleCounting(t *testing.T) {
	e := newEnv(t)
	at := noon.Add(-time.Hour)
	path := e.write(e.claude, "p/s.jsonl", claudeLine("msg_1", at, 10, 0, 0, 0), claudeLine("msg_2", at, 20, 0, 0, 0))
	e.run(noon)
	// The file is replaced by a shorter one that repeats msg_2 and adds msg_3.
	if err := os.WriteFile(path, []byte(claudeLine("msg_3", at, 4, 0, 0, 0)), 0o644); err != nil {
		t.Fatal(err)
	}
	e.run(noon)
	if c, _ := readUsage(t).day(t, "2026-11-03"); c != 34 {
		t.Errorf("claude = %d, want 34 (10 + 20 kept, 4 added)", c)
	}
	var offsets map[string]struct{ Size, Offset int64 }
	_ = state.ReadJSON("usage-offsets.json", &offsets)
	if got := offsets[path]; got.Offset != got.Size || got.Size == 0 {
		t.Errorf("offsets after shrink = %+v", got)
	}
}

func TestLocalDateBoundary(t *testing.T) {
	for _, loc := range []*time.Location{kst, time.FixedZone("PST", -8*3600), time.UTC} {
		t.Run(loc.String(), func(t *testing.T) {
			e := newEnv(t)
			now := time.Date(2026, 11, 4, 12, 0, 0, 0, loc)
			before := time.Date(2026, 11, 3, 23, 59, 0, 0, loc)
			after := time.Date(2026, 11, 4, 0, 1, 0, 0, loc)
			e.write(e.claude, "p/s.jsonl", claudeLine("a", before, 1, 0, 0, 0), claudeLine("b", after, 10, 0, 0, 0))
			e.write(e.codex, "2026/11/03/rollout-a.jsonl", codexLine("r1", before, 100), codexLine("r2", after, 1000))

			e.run(now)
			o := readUsage(t)
			if c, x := o.day(t, "2026-11-03"); c != 1 || x != 100 {
				t.Errorf("11-03 = claude %d codex %d, want 1 and 100", c, x)
			}
			if c, x := o.day(t, "2026-11-04"); c != 10 || x != 1000 {
				t.Errorf("11-04 = claude %d codex %d, want 10 and 1000", c, x)
			}
		})
	}
}

func TestOldFilesAreNotRead(t *testing.T) {
	e := newEnv(t)
	old := e.write(e.claude, "p/old.jsonl", claudeLine("msg_old", noon.Add(-time.Hour), 500, 0, 0, 0))
	when := noon.AddDate(0, 0, -30)
	if err := os.Chtimes(old, when, when); err != nil {
		t.Fatal(err)
	}
	e.write(e.claude, "p/new.jsonl", claudeLine("msg_new", noon.Add(-time.Hour), 3, 0, 0, 0))

	// The files' mtimes are real, so use a "now" that makes old.jsonl 30 days stale.
	now := time.Now().In(kst)
	e.run(now)
	var offsets map[string]struct{ Size, Offset int64 }
	_ = state.ReadJSON("usage-offsets.json", &offsets)
	if _, ok := offsets[old]; ok {
		t.Errorf("a file untouched for 30 days was read: %+v", offsets)
	}
	if len(offsets) != 1 {
		t.Errorf("offsets = %+v, want only new.jsonl", offsets)
	}
}

func TestStateFilesAreValidJSON(t *testing.T) {
	e := newEnv(t)
	e.write(e.claude, "p/s.jsonl", claudeLine("a", noon.Add(-time.Hour), 1, 0, 0, 0))
	e.run(noon)
	for _, p := range []string{filepath.Join(state.UsageDir(), "daily.json"), filepath.Join(state.Dir(), "usage-offsets.json")} {
		name := filepath.Base(p)
		b, err := os.ReadFile(p)
		if err != nil || !json.Valid(b) {
			t.Errorf("%s: err=%v valid=%v", name, err, json.Valid(b))
		}
	}
}

// The files in testdata are real log lines with every text field emptied:
// four Claude assistant lines (the first two are one message logged twice)
// and a Codex rollout with its five token_usage_records.
func TestRealShapedSamples(t *testing.T) {
	e := newEnv(t)
	for src, dst := range map[string]string{
		"testdata/claude-assistant.jsonl": filepath.Join(e.claude, "proj", "s.jsonl"),
		"testdata/codex-rollout.jsonl":    filepath.Join(e.codex, "2026", "11", "03", "rollout-x.jsonl"),
	} {
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e.run(noon)
	o := readUsage(t)
	// 2026-10-29T01:54Z is 10:54 on the 29th in KST.
	if c, _ := o.day(t, "2026-10-29"); c != 50516+52577+53861 {
		t.Errorf("claude 10-29 = %d, want %d", c, 50516+52577+53861)
	}
	if _, x := o.day(t, "2026-11-03"); x != 17557+18392+67694+71020+71152 {
		t.Errorf("codex 11-03 = %d, want %d", x, 17557+18392+67694+71020+71152)
	}
}
