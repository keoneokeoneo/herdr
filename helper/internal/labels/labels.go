// Package labels writes per-terminal facts as herdr-core labels (`hc label`).
package labels

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"time"

	"herdr-poke/internal/state"
)

// Timeout caps one `hc label` call, so a hung hc can never block a statusline.
var Timeout = 300 * time.Millisecond

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func cacheName(terminal string) string {
	return "labels-" + unsafeName.ReplaceAllString(terminal, "_") + ".json"
}

// Set runs `hc label <terminal> k=v...` for the keys whose value differs from
// the last value that was successfully written (cached in
// state.Dir()/labels-<terminal>.json). An empty value removes the label in hc,
// and is cached like any other value. It does nothing when terminal is "".
//
// All changed keys go out in one hc call. If hc fails or times out, nothing is
// cached, so the next Set retries.
func Set(terminal string, kv map[string]string) error {
	if terminal == "" || len(kv) == 0 {
		return nil
	}
	cache := map[string]string{}
	_ = state.ReadJSON(cacheName(terminal), &cache) // missing or corrupt: nothing cached

	var args []string
	var changed []string
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if old, ok := cache[k]; ok && old == kv[k] {
			continue
		}
		changed = append(changed, k)
		args = append(args, k+"="+kv[k])
	}
	if len(changed) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	// Stdout/Stderr stay nil (/dev/null): no pipes to wait on after a kill.
	cmd := exec.CommandContext(ctx, "hc", append([]string{"label", terminal}, args...)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hc label %s: %w", terminal, err)
	}

	// Merge into a fresh read under the lock: a concurrent Set for the same
	// terminal may have cached other keys meanwhile.
	return state.WithLock(cacheName(terminal)+".lock", func() error {
		fresh := map[string]string{}
		_ = state.ReadJSON(cacheName(terminal), &fresh)
		for _, k := range changed {
			fresh[k] = kv[k]
		}
		return state.WriteJSON(cacheName(terminal), fresh)
	})
}
