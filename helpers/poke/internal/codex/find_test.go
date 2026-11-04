package codex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// started is a fixed local time so the day folder never depends on the clock.
var started = time.Date(2026, 11, 3, 12, 0, 0, 0, time.Local)

// writeRollout creates sessions/YYYY/MM/DD/rollout-<local ts>-<id>.jsonl for a
// session created at `at`, with a one-line session_meta, and returns its path.
// extra is spliced into the meta payload (e.g. `"forked_from_id":"x",`).
func writeRollout(t *testing.T, sessions, id, cwd string, at time.Time, extra string) string {
	t.Helper()
	dir := filepath.Join(sessions, at.Local().Format("2006/01/02"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rollout-"+at.Local().Format("2006-01-02T15-04-05")+"-"+id+".jsonl")
	line := fmt.Sprintf(`{"timestamp":%q,"type":"session_meta","payload":{"id":%q,%s"cwd":%q,"originator":"codex-tui","timestamp":%q,"source":"cli"}}`+"\n",
		at.UTC().Add(8*time.Second).Format(time.RFC3339Nano), id, extra, cwd, at.UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func noLsof(int) ([]string, error) { return nil, nil }

func opts(sessions string, lsof func(int) ([]string, error)) FindOpts {
	return FindOpts{SessionsDir: sessions, Lsof: lsof, Poll: 5 * time.Millisecond}
}

func find(t *testing.T, o FindOpts, pid int, cwd string, at time.Time) (string, bool, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	return FindRollout(ctx, pid, cwd, at, o)
}

func TestFindViaLsof(t *testing.T) {
	sessions := t.TempDir()
	want := writeRollout(t, sessions, "aaaa", "/work/demo", started.Add(300*time.Millisecond), "")
	var gotPID int
	lsof := func(pid int) ([]string, error) {
		gotPID = pid
		return []string{"/Users/x/project", "/dev/ttys003", want, "/usr/lib/dyld"}, nil
	}
	// A decoy that the fallback would also accept must not matter.
	writeRollout(t, sessions, "bbbb", "/work/demo", started.Add(100*time.Millisecond), "")

	path, certain, err := find(t, opts(sessions, lsof), 4242, "/work/demo", started)
	if err != nil || path != want || !certain {
		t.Errorf("FindRollout = %q certain=%v err=%v, want %q certain", path, certain, err, want)
	}
	if gotPID != 4242 {
		t.Errorf("lsof asked about pid %d, want 4242", gotPID)
	}
}

func TestFindPollsUntilLsofSeesTheFile(t *testing.T) {
	want := "/home/x/.codex/sessions/2026/11/03/rollout-2026-11-03T12-00-01-cccc.jsonl"
	calls := 0
	lsof := func(int) ([]string, error) {
		calls++
		if calls < 3 {
			return nil, nil // rollout is created lazily, on the first prompt
		}
		return []string{want}, nil
	}
	path, certain, err := find(t, opts(t.TempDir(), lsof), 1, "/work/demo", started)
	if err != nil || path != want || !certain || calls != 3 {
		t.Errorf("FindRollout = %q certain=%v err=%v after %d calls", path, certain, err, calls)
	}
}

func TestFindFallbackPicksMatchingCwd(t *testing.T) {
	sessions := t.TempDir()
	other := writeRollout(t, sessions, "aaaa", "/work/other", started.Add(300*time.Millisecond), "")
	mine := writeRollout(t, sessions, "bbbb", "/work/demo", started.Add(300*time.Millisecond), "")

	path, certain, err := find(t, opts(sessions, noLsof), 1, "/work/demo", started)
	if err != nil || path != mine || certain {
		t.Errorf("FindRollout = %q certain=%v err=%v, want %q uncertain", path, certain, err, mine)
	}
	path, _, err = find(t, opts(sessions, noLsof), 1, "/work/other", started)
	if err != nil || path != other {
		t.Errorf("FindRollout(other cwd) = %q err=%v, want %q", path, err, other)
	}
}

func TestFindFallbackSameCwdOrder(t *testing.T) {
	sessions := t.TempDir()
	first := writeRollout(t, sessions, "aaaa", "/work/demo", started.Add(400*time.Millisecond), "")
	second := writeRollout(t, sessions, "bbbb", "/work/demo", started.Add(1400*time.Millisecond), "")

	// Spawned 1 s apart, so each wrapper owns the file created nearest after it.
	if path, _, err := find(t, opts(sessions, noLsof), 1, "/work/demo", started); err != nil || path != first {
		t.Errorf("first wrapper got %q err=%v, want %q", path, err, first)
	}
	if path, _, err := find(t, opts(sessions, noLsof), 2, "/work/demo", started.Add(time.Second)); err != nil || path != second {
		t.Errorf("second wrapper got %q err=%v, want %q", path, err, second)
	}
}

func TestFindIgnoresForks(t *testing.T) {
	sessions := t.TempDir()
	// Both are nearer to the spawn time than the real session.
	writeRollout(t, sessions, "aaaa", "/work/demo", started.Add(100*time.Millisecond), `"forked_from_id":"zzzz",`)
	writeRollout(t, sessions, "bbbb", "/work/demo", started.Add(200*time.Millisecond), `"parent_thread_id":"zzzz",`)
	real := writeRollout(t, sessions, "cccc", "/work/demo", started.Add(900*time.Millisecond), "")

	path, certain, err := find(t, opts(sessions, noLsof), 1, "/work/demo", started)
	if err != nil || path != real || certain {
		t.Errorf("FindRollout = %q certain=%v err=%v, want %q", path, certain, err, real)
	}
}

func TestFindForkOnlyFindsNothing(t *testing.T) {
	sessions := t.TempDir()
	writeRollout(t, sessions, "aaaa", "/work/demo", started.Add(100*time.Millisecond), `"forked_from_id":"zzzz",`)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	path, _, err := FindRollout(ctx, 1, "/work/demo", started, opts(sessions, noLsof))
	if path != "" || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("FindRollout = %q err=%v, want nothing and a deadline error", path, err)
	}
}

func TestFindFallbackIgnoresOtherTimes(t *testing.T) {
	sessions := t.TempDir()
	writeRollout(t, sessions, "aaaa", "/work/demo", started.Add(-time.Minute), "")  // older session
	writeRollout(t, sessions, "bbbb", "/work/demo", started.Add(time.Minute), "")   // much later session
	writeRollout(t, sessions, "cccc", "/work/demo", started.Add(-24*time.Hour), "") // yesterday's folder

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	path, _, err := FindRollout(ctx, 1, "/work/demo", started, opts(sessions, noLsof))
	if path != "" || err == nil {
		t.Errorf("FindRollout = %q err=%v, want nothing", path, err)
	}
}

func TestFindFallbackAcrossMidnight(t *testing.T) {
	sessions := t.TempDir()
	at := time.Date(2026, 11, 3, 23, 59, 59, 0, time.Local)
	want := writeRollout(t, sessions, "aaaa", "/work/demo", at.Add(1500*time.Millisecond), "") // lands in the 11/04 folder
	path, _, err := find(t, opts(sessions, noLsof), 1, "/work/demo", at)
	if err != nil || path != want {
		t.Errorf("FindRollout = %q err=%v, want %q", path, err, want)
	}
}

func TestFindLsofErrorFallsBack(t *testing.T) {
	sessions := t.TempDir()
	want := writeRollout(t, sessions, "aaaa", "/work/demo", started.Add(300*time.Millisecond), "")
	lsof := func(int) ([]string, error) { return nil, errors.New("lsof: not found") }
	path, certain, err := find(t, opts(sessions, lsof), 1, "/work/demo", started)
	if err != nil || path != want || certain {
		t.Errorf("FindRollout = %q certain=%v err=%v, want %q uncertain", path, certain, err, want)
	}
}

func TestFindNoFallback(t *testing.T) {
	sessions := t.TempDir()
	writeRollout(t, sessions, "aaaa", "/work/demo", started.Add(300*time.Millisecond), "")
	o := opts(sessions, noLsof)
	o.NoFallback = true

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	if path, _, err := FindRollout(ctx, 1, "/work/demo", started, o); path != "" || err == nil {
		t.Errorf("FindRollout = %q err=%v, want nothing with NoFallback", path, err)
	}
}

func TestFindStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, err := FindRollout(ctx, 1, "/work/demo", started, opts(t.TempDir(), noLsof))
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("FindRollout did not return after cancel")
	}
}
