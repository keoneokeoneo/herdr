package codex

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strconv"
	"sync"
	"syscall"
	"time"

	"herdr-poke/internal/labels"
	"herdr-poke/internal/state"
)

const limitsProvider = "codex"

// Wrap is `herdr-poke codex -- <command> [args...]`. It runs the command (the
// real codex) as a child that shares this process's terminal, mirrors what
// Codex logs into terminal labels and limits.json while it runs, and returns
// the child's exit status (128+n if a signal n killed it).
//
// Codex draws its own screen and emits its own OSC 7501 state, so nothing is
// written to the terminal here. Whatever goes wrong on the herdr-poke side, the
// child still runs.
func Wrap(args []string) int {
	return wrap(args, FindOpts{})
}

func wrap(args []string, opts FindOpts) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: herdr-poke codex -- <command> [args...]")
		return 2
	}
	opts = opts.withDefaults()
	for _, a := range args[1:] {
		if a == "--no-daemon" {
			// Then lsof on the child is exact. Before the first prompt the
			// folder holds no file of this session, only other sessions' files,
			// so scanning could only attach to the wrong one.
			opts.NoFallback = true
		}
	}
	cwd, _ := os.Getwd()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	// Listen before the child exists, so no signal can kill only the wrapper.
	sigs := make(chan os.Signal, 8)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
	defer signal.Stop(sigs)

	started := time.Now()
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-poke codex: %v\n", err)
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return 127
		}
		return 126
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case s := <-sigs:
				_ = cmd.Process.Signal(s)
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		defer func() { _ = recover() }() // a bug in the mirror must never reach the child
		monitor(ctx, cmd.Process.Pid, cwd, started, opts)
	}()

	err := cmd.Wait()
	cancel()
	wg.Wait() // nothing outlives the child
	return exitCode(err)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return 1
	}
	if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return exit.ExitCode()
}

// monitor finds the child's rollout, then follows it until ctx is done,
// publishing whenever the log says something new. A last read after ctx is
// done picks up what Codex wrote on its way out.
func monitor(ctx context.Context, pid int, cwd string, started time.Time, opts FindOpts) {
	path, certain, err := FindRollout(ctx, pid, cwd, started, opts)
	if err != nil {
		return // the child ended before it logged anything
	}
	t := &tailer{path: path}
	pending := false // something changed and has not been published yet
	step := func() {
		if changed, _ := t.poll(); changed {
			pending = true
		}
		if pending && publish(t.snap, os.Getenv("HC_TERMINAL_ID"), time.Now()) == nil {
			pending = false
		}
	}

	ticker := time.NewTicker(opts.Poll)
	defer ticker.Stop()
	for {
		step()
		select {
		case <-ctx.Done():
			step()
			return
		case <-ticker.C:
		}
		if !certain {
			// The path came from a guess; as soon as lsof knows better, switch.
			if p := lsofRollout(pid, opts); p != "" {
				certain = true
				if p != t.path {
					t = &tailer{path: p}
				}
			}
		}
	}
}

// tailer follows a growing JSONL file by byte offset.
type tailer struct {
	path   string
	offset int64 // end of the last complete line applied
	snap   Snapshot
}

// poll applies the complete lines appended since the last call and reports
// whether the snapshot changed. A trailing partial line waits for its newline.
// A file that got shorter was replaced: start over.
func (t *tailer) poll() (changed bool, err error) {
	f, err := os.Open(t.path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	before := t.snap
	if fi.Size() < t.offset {
		t.offset, t.snap = 0, Snapshot{}
	}
	if fi.Size() > t.offset {
		if _, err := f.Seek(t.offset, io.SeekStart); err != nil {
			return false, err
		}
		r := bufio.NewReaderSize(io.LimitReader(f, fi.Size()-t.offset), 256<<10)
		for {
			line, err := r.ReadBytes('\n')
			if err != nil {
				break // EOF: a partial last line stays unread
			}
			t.offset += int64(len(line))
			Apply(&t.snap, line)
		}
	}
	return !reflect.DeepEqual(before, t.snap), nil
}

// publish mirrors a snapshot into the terminal's labels and limits.json. It
// returns an error if a write failed, so the caller can try again.
func publish(s Snapshot, terminal string, now time.Time) error {
	var errs []error
	kv := map[string]string{}
	if s.Model != "" {
		kv["poke.model"] = s.Model
		kv["poke.effort"] = s.Effort // "" removes a stale label
	}
	if s.HasCtx {
		kv["poke.ctx"] = strconv.Itoa(s.CtxPct)
	}
	if len(kv) > 0 {
		errs = append(errs, labels.Set(terminal, kv))
	}
	if s.HasLimits {
		errs = append(errs, publishLimits(s.Limits, now))
	}
	return errors.Join(errs...)
}

// publishLimits sets limits.json[codex]. Windows whose reset time has passed
// are left out (their percentage is stale), and nothing is written when no
// window is left or when limits.json already holds newer data: a resumed old
// session replays its last event, which must not hide what another session
// reported since.
func publishLimits(l state.ProviderLimits, now time.Time) error {
	var live []state.Window
	for _, w := range l.Windows {
		if w.ResetsAt > now.Unix() {
			live = append(live, w)
		}
	}
	if len(live) == 0 {
		return nil
	}
	l.Windows = live

	var all map[string]state.ProviderLimits
	if state.ReadJSON("limits.json", &all) == nil {
		if cur, ok := all[limitsProvider]; ok {
			if cur.Updated > l.Updated || (cur.Updated == l.Updated && reflect.DeepEqual(cur, l)) {
				return nil
			}
		}
	}
	return state.UpdateLimits(limitsProvider, l)
}
