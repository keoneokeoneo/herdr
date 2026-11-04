package codex

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FindOpts configures FindRollout; the zero value is the real thing.
type FindOpts struct {
	SessionsDir string                          // default ~/.codex/sessions
	Lsof        func(pid int) ([]string, error) // names of files pid has open; default runs lsof
	Poll        time.Duration                   // time between attempts; default 1 s
	Rediscover  time.Duration                   // recheck the open rollout after an in-process session switch; default 5 s
	NoFallback  bool                            // trust lsof only: never guess from the sessions folder
}

// A rollout found by scanning (not by lsof) must have been created within
// this window around the child's spawn time. Codex stamps session_meta when
// the TUI has booted, so it always trails the spawn by a little.
const (
	scanBefore = 500 * time.Millisecond // clock rounding
	scanAfter  = 10 * time.Second       // slow TUI boot
	maxMetaLen = 4 << 20                // session_meta carries the base instructions (~20 KB)
	lsofLimit  = 750 * time.Millisecond
)

func (o FindOpts) withDefaults() FindOpts {
	if o.SessionsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		o.SessionsDir = filepath.Join(home, ".codex", "sessions")
	}
	if o.Lsof == nil {
		o.Lsof = lsofNames
	}
	if o.Rediscover <= 0 {
		o.Rediscover = 5 * time.Second
	}
	if o.Poll <= 0 {
		o.Poll = time.Second
	}
	return o
}

// FindRollout waits for the rollout file of the Codex session that child pid
// is running, polling every opts.Poll until it appears or ctx is done (then
// err is ctx.Err()).
//
// The primary way is lsof on pid: the file is held open by the process that
// writes it, so a hit is certain. That works only for `codex --no-daemon`,
// and only once the first prompt has created the file. As a fallback it picks
// the rollout in the sessions folder whose cwd is cwd, which is not a fork or
// sub-agent, and whose creation time is closest after started; that is a
// guess, so certain is false.
func FindRollout(ctx context.Context, pid int, cwd string, started time.Time, opts FindOpts) (path string, certain bool, err error) {
	opts = opts.withDefaults()
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		if p, c := findOnce(pid, cwd, started, opts); p != "" {
			return p, c, nil
		}
		timer.Reset(opts.Poll)
		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		case <-timer.C:
		}
	}
}

// findOnce is one attempt: lsof first, then (unless disabled) the scan.
func findOnce(pid int, cwd string, started time.Time, opts FindOpts) (string, bool) {
	if p := lsofRollout(pid, opts); p != "" {
		return p, true
	}
	if opts.NoFallback {
		return "", false
	}
	return scanRollout(opts.SessionsDir, cwd, started), false
}

func lsofRollout(pid int, opts FindOpts) string {
	names, err := opts.Lsof(pid)
	if err != nil {
		return ""
	}
	var found []string
	for _, n := range names {
		base := filepath.Base(n)
		if strings.HasPrefix(base, "rollout-") && strings.HasSuffix(base, ".jsonl") {
			found = append(found, n)
		}
	}
	if len(found) == 0 {
		return ""
	}
	sort.Strings(found)
	return found[len(found)-1] // one session per process; if ever more, the newest
}

// lsofNames is the real Lsof: `lsof -a -p <pid> -Fn`, the `n` (name) fields.
// "Nothing open" and "no such process" both exit 1 with no output: not an error.
func lsofNames(pid int) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lsofLimit)
	defer cancel()
	cmd := exec.CommandContext(ctx, "lsof", "-a", "-p", strconv.Itoa(pid), "-Fn")
	var out bytes.Buffer
	cmd.Stdout = &out                      // stderr stays nil (/dev/null)
	cmd.WaitDelay = 100 * time.Millisecond // a grandchild holding the pipe must not outlive the timeout
	err := cmd.Run()
	var names []string
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "n") {
			names = append(names, line[1:])
		}
	}
	var exit *exec.ExitError
	if len(names) > 0 || errors.As(err, &exit) {
		return names, nil
	}
	return nil, err
}

// scanRollout is the fallback: the best rollout in the day folders around
// started, or "".
func scanRollout(sessions, cwd string, started time.Time) string {
	var best string
	var bestGap time.Duration
	for _, dir := range dayDirs(sessions, started, started.Add(scanAfter)) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
				continue
			}
			if !nameInWindow(name, started) {
				continue // cheap: the file name carries the session start
			}
			path := filepath.Join(dir, name)
			m, ok := readMeta(path)
			if !ok || m.ForkedFromID != "" || m.ParentThreadID != "" || !sameDir(m.Cwd, cwd) {
				continue
			}
			lag := m.Timestamp.Sub(started)
			if lag < -scanBefore || lag > scanAfter {
				continue
			}
			if lag < 0 {
				lag = -lag
			}
			if best == "" || lag < bestGap || (lag == bestGap && path < best) {
				best, bestGap = path, lag
			}
		}
	}
	return best
}

// dayDirs lists the sessions/YYYY/MM/DD folders (local dates) of the given
// instants, without repeats.
func dayDirs(sessions string, at ...time.Time) []string {
	var dirs []string
	seen := map[string]bool{}
	for _, t := range at {
		d := filepath.Join(sessions, t.Local().Format("2006/01/02"))
		if !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// nameInWindow checks the local timestamp in rollout-YYYY-MM-DDTHH-MM-SS-<id>.jsonl
// (second resolution) against the scan window. Names it cannot read pass, so
// the meta line decides.
func nameInWindow(name string, started time.Time) bool {
	const stampLen = len("2006-01-02T15-04-05")
	rest := strings.TrimPrefix(name, "rollout-")
	if len(rest) < stampLen {
		return true
	}
	t, err := time.ParseInLocation("2006-01-02T15-04-05", rest[:stampLen], time.Local)
	if err != nil {
		return true
	}
	lag := t.Sub(started)
	// The name is truncated to whole seconds, so it can trail the real time by up to 1 s.
	return lag >= -scanBefore-time.Second && lag <= scanAfter
}

// readMeta parses the first line of a rollout.
func readMeta(path string) (Meta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Meta{}, false
	}
	defer f.Close()
	line, err := bufio.NewReaderSize(io.LimitReader(f, maxMetaLen), 64<<10).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return Meta{}, false
	}
	return SessionMeta(line)
}

// sameDir reports whether two paths name the same folder, looking through
// symlinks (Codex records the cwd it was given, a shell may report the real one).
func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}
