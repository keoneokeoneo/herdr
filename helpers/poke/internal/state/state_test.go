package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func useTempState(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "state")
	t.Setenv("HERDR_POKE_STATE", dir)
	return dir
}

func TestDirEnvOverrideIsCreatedPrivate(t *testing.T) {
	want := useTempState(t)
	got := Dir()
	if got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("state dir not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("state dir perm = %o, want 700", perm)
	}
}

func TestDirDefaultsUnderHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERDR_POKE_STATE", "")
	t.Setenv("HOME", home)
	want := filepath.Join(home, ".local", "state", "herdr-poke")
	if got := Dir(); got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("default state dir not created: %v", err)
	}
}

func TestWriteJSONAtomic(t *testing.T) {
	dir := useTempState(t)
	path := filepath.Join(dir, "data.json")

	type payload struct {
		Writer int      `json:"writer"`
		Seq    int      `json:"seq"`
		Pad    []string `json:"pad"`
	}
	mk := func(w, s int) payload {
		pad := make([]string, 200)
		for i := range pad {
			pad[i] = fmt.Sprintf("w%d-s%d-%d", w, s, i)
		}
		return payload{Writer: w, Seq: s, Pad: pad}
	}

	const writers, rounds = 12, 40
	done := make(chan struct{})
	var readerErr error
	var readerWG sync.WaitGroup
	readerWG.Add(1)
	go func() {
		defer readerWG.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			b, err := os.ReadFile(path)
			if err != nil {
				continue // not created yet
			}
			var p payload
			if err := json.Unmarshal(b, &p); err != nil {
				readerErr = fmt.Errorf("reader saw torn file (%d bytes): %w", len(b), err)
				return
			}
		}
	}()

	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for s := 0; s < rounds; s++ {
				if err := WriteJSON("data.json", mk(w, s)); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(done)
	readerWG.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("WriteJSON: %v", err)
	}
	if readerErr != nil {
		t.Fatal(readerErr)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var final payload
	if err := json.Unmarshal(b, &final); err != nil {
		t.Fatalf("final file is not valid JSON: %v", err)
	}
	if len(final.Pad) != 200 {
		t.Fatalf("final payload truncated: %d pad entries", len(final.Pad))
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "tmp") {
			t.Errorf("leftover temp file %q", e.Name())
		}
	}
}

func TestUpdateLimitsLatestConcurrent(t *testing.T) {
	useTempState(t)
	var wg sync.WaitGroup
	for i := 1; i <= 30; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := UpdateLimitsLatest("codex", ProviderLimits{Updated: int64(n), Windows: []Window{{Minutes: 300, Used: float64(n)}}}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	var got ProviderLimits
	if err := ReadUsageJSON("codex.json", &got); err != nil {
		t.Fatal(err)
	}
	if got.Updated != 30 {
		t.Fatalf("latest lost: %+v", got)
	}
}

func TestUsageDirEnvOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "usage")
	t.Setenv("HERDR_USAGE_STATE", want)
	t.Setenv("HERDR_POKE_STATE", filepath.Join(t.TempDir(), "other"))
	if got := UsageDir(); got != want {
		t.Fatalf("UsageDir() = %q, want %q", got, want)
	}
	if info, err := os.Stat(want); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("usage dir not created private: %v", err)
	}
}

func TestUsageDirFollowsPokeStateOverrideThenHome(t *testing.T) {
	dir := useTempState(t)
	t.Setenv("HERDR_USAGE_STATE", "")
	if got, want := UsageDir(), filepath.Join(dir, "usage"); got != want {
		t.Fatalf("UsageDir() = %q, want %q", got, want)
	}
	home := t.TempDir()
	t.Setenv("HERDR_POKE_STATE", "")
	t.Setenv("HOME", home)
	if got, want := UsageDir(), filepath.Join(home, ".local", "state", "herdr-app", "usage"); got != want {
		t.Fatalf("UsageDir() = %q, want %q", got, want)
	}
}

func TestUpdateLimitsLatestWritesProviderFile(t *testing.T) {
	dir := useTempState(t)
	t.Setenv("HERDR_USAGE_STATE", "")
	l := ProviderLimits{Plan: "p", Updated: 10, Windows: []Window{{Minutes: 10080, Used: 5, ResetsAt: 99}}}
	if err := UpdateLimitsLatest("codex", l); err != nil {
		t.Fatal(err)
	}
	var got ProviderLimits
	if err := ReadUsageJSON("codex.json", &got); err != nil || got.Plan != "p" || len(got.Windows) != 1 {
		t.Fatalf("codex.json = %+v err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "limits.json")); !os.IsNotExist(err) {
		t.Fatalf("limits.json must not be written (err=%v)", err)
	}
}

func TestOlderSnapshotDoesNotReplaceNewer(t *testing.T) {
	useTempState(t)
	t.Setenv("HERDR_USAGE_STATE", "")
	newer := ProviderLimits{Updated: 20, Windows: []Window{{Minutes: 300, Used: 50, ResetsAt: 99}}}
	older := ProviderLimits{Updated: 10, Windows: []Window{{Minutes: 300, Used: 1, ResetsAt: 99}}}
	_ = UpdateLimitsLatest("codex", newer)
	_ = UpdateLimitsLatest("codex", older)
	var got ProviderLimits
	if err := ReadUsageJSON("codex.json", &got); err != nil || got.Windows[0].Used != 50 {
		t.Fatalf("older replaced newer: %+v err=%v", got, err)
	}
}
