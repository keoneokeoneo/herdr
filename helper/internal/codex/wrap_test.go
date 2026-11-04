package codex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"herdr-poke/internal/labels"
	"herdr-poke/internal/state"
)

func TestWrapPassesExitCode(t *testing.T) {
	if got := Wrap([]string{"sh", "-c", "exit 3"}); got != 3 {
		t.Errorf("Wrap(exit 3) = %d, want 3", got)
	}
	if got := Wrap([]string{"sh", "-c", "exit 0"}); got != 0 {
		t.Errorf("Wrap(exit 0) = %d, want 0", got)
	}
	// Killed by SIGTERM: the shell convention, 128 + signal.
	if got := Wrap([]string{"sh", "-c", "kill -TERM $$"}); got != 143 {
		t.Errorf("Wrap(killed by TERM) = %d, want 143", got)
	}
}

func TestWrapMissingCommand(t *testing.T) {
	if got := Wrap([]string{"herdr-poke-no-such-command"}); got != 127 {
		t.Errorf("Wrap(missing command) = %d, want 127", got)
	}
	if got := Wrap(nil); got != 2 {
		t.Errorf("Wrap(no command) = %d, want 2", got)
	}
}

func TestWrapInheritsStdio(t *testing.T) {
	// Stdout of the child is Wrap's own stdout (no copying goroutine, no PTY).
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	code := Wrap([]string{"sh", "-c", "printf hello"})
	os.Stdout = saved
	w.Close()
	out := make([]byte, 16)
	n, _ := r.Read(out)
	r.Close()
	if code != 0 || string(out[:n]) != "hello" {
		t.Errorf("code=%d stdout=%q, want 0 and hello", code, out[:n])
	}
}

// settled reports the goroutine count once it has stopped changing.
func settled() int {
	n := runtime.NumGoroutine()
	for i := 0; i < 50; i++ {
		time.Sleep(10 * time.Millisecond)
		m := runtime.NumGoroutine()
		if m == n {
			return n
		}
		n = m
	}
	return n
}

func TestWrapNoRolloutNoLeftovers(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	lsof := func(int) ([]string, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return nil, nil
	}
	o := FindOpts{SessionsDir: t.TempDir(), Lsof: lsof, Poll: 50 * time.Millisecond}

	wrap([]string{"true"}, o) // warm-up: os/signal starts its own long-lived goroutine once
	before := settled()

	start := time.Now()
	if got := wrap([]string{"true"}, o); got != 0 {
		t.Errorf("wrap(true) = %d, want 0", got)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("wrap with an instantly exiting child took %v, want under 1 s", d)
	}
	if after := settled(); after > before {
		buf := make([]byte, 1<<16)
		t.Errorf("goroutines: %d before, %d after\n%s", before, after, buf[:runtime.Stack(buf, true)])
	}
}

func TestWrapJoinsMonitorBeforeReturning(t *testing.T) {
	// lsof is slow on its first call; the child exits meanwhile. wrap must not
	// return while the monitor goroutine is still inside it.
	var running atomic.Int32
	lsof := func(int) ([]string, error) {
		running.Add(1)
		defer running.Add(-1)
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}
	o := FindOpts{SessionsDir: t.TempDir(), Lsof: lsof, Poll: 20 * time.Millisecond}
	start := time.Now()
	wrap([]string{"true"}, o)
	if n := running.Load(); n != 0 {
		t.Errorf("wrap returned while %d lsof call(s) were still running", n)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("took %v, want under 1 s", d)
	}
}

func TestWrapForwardsSIGTERM(t *testing.T) {
	// Signals sent to the wrapper reach the child; the child's fate is the exit code.
	done := make(chan int, 1)
	go func() {
		done <- wrap([]string{"sh", "-c", "trap 'exit 7' TERM; while :; do sleep 0.05; done"}, FindOpts{Lsof: noLsof, Poll: 20 * time.Millisecond, SessionsDir: t.TempDir()})
	}()
	time.Sleep(300 * time.Millisecond)
	p, _ := os.FindProcess(os.Getpid())
	if err := p.Signal(syscall.SIGTERM); err != nil { // the wrapper catches it and passes it on
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 7 {
			t.Errorf("exit code %d, want 7 (child trapped the forwarded TERM)", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("wrap did not return after SIGTERM")
	}
}

// rollout lines for the end-to-end test. resets is in the future.
func tokenCountLine(at time.Time, lastTotal int, used float64, resets int64) string {
	return fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":%d},"model_context_window":258400},"rate_limits":{"limit_id":"codex","primary":{"used_percent":%v,"window_minutes":10080,"resets_at":%d},"secondary":null,"plan_type":"prolite"}}}`+"\n",
		at.UTC().Format(time.RFC3339Nano), lastTotal, used, resets)
}

const turnContextLine = `{"type":"turn_context","payload":{"model":"gpt-6-astra","effort":"high"}}` + "\n"

// fakeHC puts an `hc` first on PATH that logs its arguments, one call per line.
func fakeHC(t *testing.T) (logPath string) {
	t.Helper()
	bin := t.TempDir()
	logPath = filepath.Join(t.TempDir(), "hc.log")
	script := "#!/bin/sh\necho \"$@\" >> " + logPath + "\n"
	if err := os.WriteFile(filepath.Join(bin, "hc"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	// A loaded machine can take longer than the production 300 ms to start a shell.
	saved := labels.Timeout
	labels.Timeout = 10 * time.Second
	t.Cleanup(func() { labels.Timeout = saved })
	return logPath
}

func readLog(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

func TestWrapTailsRolloutIntoLabelsAndLimits(t *testing.T) {
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	t.Setenv("HC_TERMINAL_ID", "t42")
	hcLog := fakeHC(t)

	now := time.Now()
	resets := now.Add(24 * time.Hour).Unix()
	rollout := filepath.Join(t.TempDir(), "rollout-2026-11-03T12-00-00-aaaa.jsonl")
	// The file has history already when the wrapper attaches (a resumed session).
	first := turnContextLine + tokenCountLine(now, 135200, 38.0, resets)
	if err := os.WriteFile(rollout, []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	lsof := func(int) ([]string, error) { return []string{rollout}, nil }
	o := FindOpts{SessionsDir: t.TempDir(), Lsof: lsof, Poll: 20 * time.Millisecond}

	// While Codex runs, a new turn appends lines, the last one cut mid-line.
	go func() {
		time.Sleep(400 * time.Millisecond)
		f, err := os.OpenFile(rollout, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		defer f.Close()
		line := tokenCountLine(time.Now(), 12000, 39.0, resets)
		f.WriteString(line[:40]) // partial: must not be applied yet
		time.Sleep(100 * time.Millisecond)
		f.WriteString(line[40:])
	}()

	if got := wrap([]string{"sh", "-c", "sleep 1"}, o); got != 0 {
		t.Fatalf("wrap = %d", got)
	}

	log := readLog(hcLog)
	if !strings.Contains(log, "label t42 poke.ctx=50 poke.effort=high poke.model=gpt-6-astra") {
		t.Errorf("first labels missing from hc log:\n%s", log)
	}
	if !strings.Contains(log, "label t42 poke.ctx=0") {
		t.Errorf("updated ctx missing from hc log:\n%s", log)
	}
	if strings.Contains(log, "poke.model") && strings.Count(log, "poke.model") != 1 {
		t.Errorf("unchanged model was written again:\n%s", log)
	}

	var all map[string]state.ProviderLimits
	if err := state.ReadJSON("limits.json", &all); err != nil {
		t.Fatal(err)
	}
	l, ok := all["codex"]
	if !ok || l.Plan != "prolite" || len(l.Windows) != 1 || l.Windows[0].Minutes != 10080 || l.Windows[0].Used != 39.0 || l.Windows[0].ResetsAt != resets {
		b, _ := json.Marshal(all)
		t.Errorf("limits.json = %s, want codex weekly 39%%", b)
	}
}

func TestWrapSwitchesToLsofFileAfterGuess(t *testing.T) {
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	t.Setenv("HC_TERMINAL_ID", "t43")
	hcLog := fakeHC(t)

	now := time.Now()
	resets := now.Add(24 * time.Hour).Unix()
	sessions := t.TempDir()
	cwd, _ := os.Getwd()

	own := filepath.Join(t.TempDir(), "rollout-2026-11-03T12-00-05-cccc.jsonl")
	var mu sync.Mutex
	owned := false
	lsof := func(int) ([]string, error) {
		mu.Lock()
		defer mu.Unlock()
		if owned {
			return []string{own}, nil
		}
		return nil, nil
	}
	go func() {
		// A sibling session in the same folder appears just after the spawn: the
		// scan picks it first, because lsof sees nothing yet.
		time.Sleep(100 * time.Millisecond)
		sibling := writeRollout(t, sessions, "bbbb", cwd, time.Now(), "")
		appendTo(t, sibling, turnContextLine+tokenCountLine(now, 12000, 10.0, resets))
		// Later this wrapper's own session writes its file, and lsof sees it.
		time.Sleep(600 * time.Millisecond)
		appendTo(t, own, `{"type":"turn_context","payload":{"model":"gpt-6-own","effort":"low"}}`+"\n"+tokenCountLine(time.Now(), 135200, 20.0, resets))
		mu.Lock()
		owned = true
		mu.Unlock()
	}()

	o := FindOpts{SessionsDir: sessions, Lsof: lsof, Poll: 20 * time.Millisecond}
	wrap([]string{"sh", "-c", "sleep 1.5"}, o)

	log := readLog(hcLog)
	if !strings.Contains(log, "poke.model=gpt-6-astra") || !strings.Contains(log, "poke.model=gpt-6-own") {
		t.Errorf("expected the guess first, then the real file's labels:\n%s", log)
	}
	if strings.LastIndex(log, "gpt-6-own") < strings.LastIndex(log, "gpt-6-astra") {
		t.Errorf("real file's labels must come last:\n%s", log)
	}
}

func appendTo(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(s); err != nil {
		t.Error(err)
	}
}

func TestTailerHandlesTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-x.jsonl")
	appendTo(t, path, turnContextLine+tokenCountLine(time.Now(), 135200, 38.0, 1))
	tl := &tailer{path: path}
	if changed, err := tl.poll(); err != nil || !changed || tl.snap.CtxPct != 50 {
		t.Fatalf("first poll: changed=%v err=%v ctx=%d", changed, err, tl.snap.CtxPct)
	}
	if changed, _ := tl.poll(); changed {
		t.Error("second poll with no new data reported a change")
	}
	// The file is replaced by a shorter one: start over, forget the old snapshot.
	if err := os.WriteFile(path, []byte(`{"type":"turn_context","payload":{"model":"m2","effort":"low"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, err := tl.poll(); err != nil || !changed || tl.snap.Model != "m2" || tl.snap.HasCtx {
		t.Errorf("after truncation: changed=%v err=%v snap=%+v", changed, err, tl.snap)
	}
}

func TestPublishLimitsSkipsStaleAndOlder(t *testing.T) {
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	now := time.Unix(1_800_000_000, 0)

	newer := state.ProviderLimits{Plan: "p", Updated: now.Unix() - 10, Windows: []state.Window{{Minutes: 10080, Used: 50, ResetsAt: now.Unix() + 3600}}}
	publishLimits(newer, now)
	read := func() state.ProviderLimits {
		var all map[string]state.ProviderLimits
		if err := state.ReadJSON("limits.json", &all); err != nil {
			t.Fatal(err)
		}
		return all["codex"]
	}
	if got := read(); got.Windows[0].Used != 50 {
		t.Fatalf("first publish not written: %+v", got)
	}

	// Older data (a resumed session's last event) must not replace newer data.
	older := state.ProviderLimits{Plan: "p", Updated: now.Unix() - 1000, Windows: []state.Window{{Minutes: 10080, Used: 10, ResetsAt: now.Unix() + 3600}}}
	publishLimits(older, now)
	if got := read(); got.Windows[0].Used != 50 {
		t.Errorf("older data overwrote newer: %+v", got)
	}

	// A window whose reset has passed is dropped; with none left nothing is written.
	stale := state.ProviderLimits{Plan: "p", Updated: now.Unix() + 5, Windows: []state.Window{{Minutes: 10080, Used: 99, ResetsAt: now.Unix() - 1}}}
	publishLimits(stale, now)
	if got := read(); got.Windows[0].Used != 50 {
		t.Errorf("stale window was written: %+v", got)
	}

	// Newer data with a mix keeps only the live window.
	mixed := state.ProviderLimits{Plan: "p", Updated: now.Unix() + 6, Windows: []state.Window{
		{Minutes: 300, Used: 1, ResetsAt: now.Unix() - 1}, {Minutes: 10080, Used: 60, ResetsAt: now.Unix() + 3600}}}
	publishLimits(mixed, now)
	if got := read(); len(got.Windows) != 1 || got.Windows[0].Used != 60 {
		t.Errorf("mixed publish = %+v, want only the weekly window", got)
	}
}

func TestWrapSkipsMirrorWhenTerminalLockHeld(t *testing.T) {
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	t.Setenv("HC_TERMINAL_ID", "locked")
	release, ok := tryTerminalLock("locked")
	if !ok {
		t.Fatal("lock")
	}
	defer release()
	called := false
	opts := FindOpts{Lsof: func(int) ([]string, error) { called = true; return nil, nil }}
	if code := wrap([]string{"true"}, opts); code != 0 {
		t.Fatalf("child failed: %d", code)
	}
	if called {
		t.Fatal("mirror ran without owning terminal lock")
	}
}
