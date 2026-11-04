// Package state owns the herdr-poke state directory: small JSON files that the
// helper writes and the Herdr mods read (limits.json, usage.json, label caches).
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

const limitsFile = "limits.json"

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

// WriteJSON atomically replaces Dir()/name with the JSON encoding of v: it
// writes a temp file in the same directory and renames it over the target, so
// readers see either the old or the new file, never a partial one.
func WriteJSON(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	dir := Dir()
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

// WithLock runs fn while holding an exclusive flock on Dir()/lockName. It
// gives up with an error if the lock is not free within a second.
func WithLock(lockName string, fn func() error) error {
	f, err := os.OpenFile(filepath.Join(Dir(), lockName), os.O_CREATE|os.O_RDWR, 0o600)
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

// UpdateLimits sets limits.json[provider] = l, leaving every other provider's
// entry untouched. The read-modify-write runs under a flock, and the file is
// replaced atomically. A missing or corrupt limits.json starts from empty.
func UpdateLimits(provider string, l ProviderLimits) error {
	if l.Windows == nil {
		l.Windows = []Window{} // serialise as [] rather than null
	}
	entry, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return WithLock("limits.lock", func() error {
		all := map[string]json.RawMessage{}
		if b, err := os.ReadFile(filepath.Join(Dir(), limitsFile)); err == nil {
			if json.Unmarshal(b, &all) != nil || all == nil {
				all = map[string]json.RawMessage{}
			}
		}
		all[provider] = entry
		return WriteJSON(limitsFile, all)
	})
}

// MergeLimits is UpdateLimits for a provider reported by several processes at
// once (one Claude statusline per session). For a window with the same reset
// time it keeps the larger used %, so an older session can't move it back,
// and it skips the write entirely when no window changed: every rewrite wakes
// the mod's folder watch. wrote reports whether limits.json was written.
func MergeLimits(provider string, l ProviderLimits) (wrote bool, err error) {
	err = WithLock("limits.lock", func() error {
		all := map[string]json.RawMessage{}
		if b, e := os.ReadFile(filepath.Join(Dir(), limitsFile)); e == nil {
			if json.Unmarshal(b, &all) != nil || all == nil {
				all = map[string]json.RawMessage{}
			}
		}
		var old ProviderLimits
		if raw, ok := all[provider]; ok {
			_ = json.Unmarshal(raw, &old)
		}
		merged := make([]Window, 0, len(l.Windows))
		for _, w := range l.Windows {
			for _, o := range old.Windows {
				if o.Minutes == w.Minutes && o.ResetsAt == w.ResetsAt && o.Used > w.Used {
					w.Used = o.Used
				}
			}
			merged = append(merged, w)
		}
		if sameWindows(old.Windows, merged) && old.Plan == l.Plan {
			return nil
		}
		l.Windows = merged
		entry, e := json.Marshal(l)
		if e != nil {
			return e
		}
		all[provider] = entry
		wrote = true
		return WriteJSON(limitsFile, all)
	})
	return wrote, err
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

// UpdateLimitsLatest compares and writes under one lock, including concurrent sessions.
func UpdateLimitsLatest(provider string, l ProviderLimits) error {
	return WithLock("limits.lock", func() error {
		all := map[string]ProviderLimits{}
		_ = ReadJSON(limitsFile, &all)
		if all == nil {
			all = map[string]ProviderLimits{}
		}
		if old, ok := all[provider]; ok {
			if old.Updated > l.Updated {
				return nil
			}
			if old.Updated == l.Updated && old.Plan == l.Plan && sameWindows(old.Windows, l.Windows) {
				return nil
			}
		}
		all[provider] = l
		return WriteJSON(limitsFile, all)
	})
}
