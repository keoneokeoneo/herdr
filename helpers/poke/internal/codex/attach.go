package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"herdr-poke/internal/state"
)

// foreground is the part of `hc get <t>` that says what runs in the terminal.
type foreground struct {
	Argv []string `json:"argv"`
	Cwd  string   `json:"cwd"`
	Pid  int      `json:"pid"`
}

// parseForeground accepts `hc get` output whose foreground program is Codex
// itself. A Codex under the herdr-poke wrapper is refused: the wrapper already
// publishes its labels.
func parseForeground(b []byte) (foreground, bool) {
	var d struct {
		Foreground *foreground `json:"foreground"`
	}
	if json.Unmarshal(b, &d) != nil || d.Foreground == nil || d.Foreground.Pid <= 0 || len(d.Foreground.Argv) == 0 {
		return foreground{}, false
	}
	if filepath.Base(d.Foreground.Argv[0]) != "codex" {
		return foreground{}, false
	}
	return *d.Foreground, true
}

// tryTerminalLock takes Dir()/codex-<t>.lock without waiting. The holder is
// the one process mirroring that terminal's Codex; it releases on exit.
func tryTerminalLock(terminal string) (release func(), ok bool) {
	f, err := os.OpenFile(filepath.Join(state.Dir(), "codex-"+terminal+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false
	}
	if syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		f.Close()
		return nil, false
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, true
}

// Attach is `herdr-poke codex --attach <t>`: mirror a Codex that was started
// without the wrapper (by `hc run`, a script, another tool) into terminal t's
// labels, until that Codex exits. It exits 0 at once when t runs no bare Codex
// or another herdr-poke already mirrors it.
func Attach(terminal string) int {
	if terminal == "" || strings.HasPrefix(terminal, "-") {
		fmt.Fprintln(os.Stderr, "usage: herdr-poke codex --attach <terminal>")
		return 2
	}

	// Take the lock before querying hc: periodic health probes of a live mirror are cheap.
	release, ok := tryTerminalLock(terminal)
	if !ok {
		return 0
	}
	defer release()
	fg, ok := getForeground(terminal)
	if !ok {
		return 0
	}
	run, stop := context.WithCancel(context.Background())
	defer stop()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-run.Done():
				return
			case <-ticker.C:
				current, ok := getForeground(terminal)
				if !ok || current.Pid != fg.Pid || current.Cwd != fg.Cwd {
					stop()
					return
				}
			}
		}
	}()
	defer func() { _ = recover() }()
	// lsof only: started is unknown, so scanning the sessions folder could pick another session.
	monitor(run, fg.Pid, fg.Cwd, time.Now(), terminal, FindOpts{NoFallback: true}.withDefaults(), fg.Pid)
	return 0
}

func getForeground(terminal string) (foreground, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "hc", "get", terminal, "--json")
	cmd.WaitDelay = 100 * time.Millisecond
	out, err := cmd.Output()
	if err != nil {
		return foreground{}, false
	}
	return parseForeground(out)
}
