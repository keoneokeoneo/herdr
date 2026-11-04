package labels

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Telemetry is a complete snapshot. Missing values remove facts left by an
// earlier session on the same terminal. Owner is the foreground process PID,
// not a helper PID; the UI must match it before displaying these facts.
func Telemetry(owner int, provider, session string, values map[string]string) map[string]string {
	out := map[string]string{"poke.model": "", "poke.effort": "", "poke.ctx": "",
		"poke.owner": strconv.Itoa(owner), "poke.provider": provider, "poke.session": session}
	for k, v := range values {
		out[k] = v
	}
	return out
}

// ForegroundOwner is used by short-lived Claude statusline helpers. A bounded
// lookup prevents retained terminal labels from being attributed to a shell or
// another provider. Failure leaves the previous snapshot untouched and unverified.
func ForegroundOwner(terminal, provider string) int {
	if terminal == "" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "hc", "get", terminal, "--json")
	cmd.WaitDelay = 100 * time.Millisecond
	raw, err := cmd.Output()
	if err != nil {
		return 0
	}
	owner := parseOwner(raw, provider)
	if owner <= 0 {
		return 0
	}
	// A delayed statusline must not claim a replacement Claude on this terminal.
	if owner == os.Getpid() || owner == os.Getppid() {
		return owner
	}
	ps := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=")
	ps.WaitDelay = 100 * time.Millisecond
	tree, err := ps.Output()
	if err != nil {
		return 0
	}
	if ancestor(owner, os.Getpid(), tree) {
		return owner
	}
	return 0
}

func parseOwner(raw []byte, provider string) int {
	var d struct {
		Foreground *struct {
			Pid  int      `json:"pid"`
			Argv []string `json:"argv"`
		} `json:"foreground"`
	}
	if json.Unmarshal(raw, &d) != nil || d.Foreground == nil || d.Foreground.Pid <= 0 || len(d.Foreground.Argv) == 0 {
		return 0
	}
	name := filepath.Base(d.Foreground.Argv[0])
	if (provider == "claude" && (name == "claude" || name == "claude-code")) || (provider == "codex" && name == "codex") {
		return d.Foreground.Pid
	}
	return 0
}

// ancestor accepts only a parent chain containing the claimed process.
func ancestor(owner, child int, raw []byte) bool {
	parents := map[int]int{}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		pid, e1 := strconv.Atoi(f[0])
		parent, e2 := strconv.Atoi(f[1])
		if e1 == nil && e2 == nil {
			parents[pid] = parent
		}
	}
	seen := map[int]bool{}
	for child > 0 && !seen[child] {
		if child == owner {
			return true
		}
		seen[child] = true
		child = parents[child]
	}
	return false
}
