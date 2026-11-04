// Package state owns the herdr-poke state directory: small JSON files that the
// helper writes (label caches, usage ledger and offsets, locks). The neutral usage data
// contract (claude.json, codex.json, daily.json) lives in UsageDir().
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Window is one rate-limit window of a provider.
type Window struct {
	Minutes  int     `json:"minutes"`
	Used     float64 `json:"used"`
	ResetsAt int64   `json:"resets_at"`
}

// ProviderLimits is what one provider (claude, codex) reports about its limits.
type ProviderLimits struct {
	Plan    string   `json:"plan"`
	Windows []Window `json:"windows"`
	Updated int64    `json:"updated"`
}

// lockWait bounds how long a writer waits for another one, so a stopped
// process holding a lock can never hang a statusline.
const lockWait = time.Second

// Dir returns the state directory: $HERDR_POKE_STATE, else
// ~/.local/state/herdr-poke. It is created (0700) if missing.
func Dir() string {
	dir := os.Getenv("HERDR_POKE_STATE")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		dir = filepath.Join(home, ".local", "state", "herdr-poke")
	}
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// UsageDir is the neutral usage data contract folder that the base mod reads:
// $HERDR_USAGE_STATE, else $HERDR_POKE_STATE/usage when that override is set (so tests
// and sandboxes that relocate the state dir never touch the real home), else
// ~/.local/state/herdr-app/usage. It is created (0700) if missing.
func UsageDir() string {
	dir := os.Getenv("HERDR_USAGE_STATE")
	if dir == "" {
		if poke := os.Getenv("HERDR_POKE_STATE"); poke != "" {
			dir = filepath.Join(poke, "usage")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				home = os.TempDir()
			}
			dir = filepath.Join(home, ".local", "state", "herdr-app", "usage")
		}
	}
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// WriteJSON atomically replaces Dir()/name with the JSON encoding of v: it
// writes a temp file in the same directory and renames it over the target, so
// readers see either the old or the new file, never a partial one.
func WriteJSON(name string, v any) error { return writeJSONIn(Dir(), name, v) }

// WriteUsageJSON is WriteJSON for the neutral usage data contract folder (UsageDir()).
func WriteUsageJSON(name string, v any) error { return writeJSONIn(UsageDir(), name, v) }

func writeJSONIn(dir, name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(dir, "."+name+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(dir, name)); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// ReadJSON decodes Dir()/name into v. A missing file is reported as an error
// satisfying errors.Is(err, os.ErrNotExist).
func ReadJSON(name string, v any) error {
	b, err := os.ReadFile(filepath.Join(Dir(), name))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ReadUsageJSON decodes UsageDir()/name into v.
func ReadUsageJSON(name string, v any) error {
	b, err := os.ReadFile(filepath.Join(UsageDir(), name))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// WithLock runs fn while holding an exclusive flock on Dir()/lockName. It
// gives up with an error if the lock is not free within a second.
func WithLock(lockName string, fn func() error) error { return withLockIn(Dir(), lockName, fn) }

func withLockIn(dir, lockName string, fn func() error) error {
	f, err := os.OpenFile(filepath.Join(dir, lockName), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	deadline := time.Now().Add(lockWait)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EINTR) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("state: timed out waiting for lock %s", lockName)
		}
		time.Sleep(2 * time.Millisecond)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func sameWindows(a, b []Window) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// UpdateLimitsLatest writes UsageDir()/<provider>.json (the provider's limits
// object itself) unless the file already holds newer or identical data. Compare and
// write run under one lock, including concurrent sessions.
func UpdateLimitsLatest(provider string, l ProviderLimits) error {
	if l.Windows == nil {
		l.Windows = []Window{}
	}
	name := provider + ".json"
	return withLockIn(UsageDir(), provider+".lock", func() error {
		var old ProviderLimits
		if ReadUsageJSON(name, &old) == nil {
			if old.Updated > l.Updated {
				return nil
			}
			if old.Updated == l.Updated && old.Plan == l.Plan && sameWindows(old.Windows, l.Windows) {
				return nil
			}
		}
		return WriteUsageJSON(name, l)
	})
}
