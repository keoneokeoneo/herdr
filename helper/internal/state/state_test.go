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

func readLimitsFile(t *testing.T, dir string) map[string]ProviderLimits {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "limits.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]ProviderLimits
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("limits.json invalid: %v\n%s", err, b)
	}
	return m
}

func TestUpdateLimitsKeepsOtherProvider(t *testing.T) {
	dir := useTempState(t)

	claude := ProviderLimits{
		Windows: []Window{{Minutes: 300, Used: 42.5, ResetsAt: 1793705100}},
		Updated: 1793700000,
	}
	codex := ProviderLimits{
		Plan:    "pro",
		Windows: []Window{{Minutes: 10080, Used: 39, ResetsAt: 1793780927}},
		Updated: 1793700001,
	}
	if err := UpdateLimits("claude", claude); err != nil {
		t.Fatal(err)
	}
	if err := UpdateLimits("codex", codex); err != nil {
		t.Fatal(err)
	}

	got := readLimitsFile(t, dir)
	if len(got) != 2 {
		t.Fatalf("want claude+codex, got %v", got)
	}
	if g := got["claude"]; len(g.Windows) != 1 || g.Windows[0] != claude.Windows[0] || g.Updated != claude.Updated {
		t.Errorf("claude entry changed: %+v", g)
	}
	if g := got["codex"]; g.Plan != "pro" || len(g.Windows) != 1 || g.Windows[0] != codex.Windows[0] {
		t.Errorf("codex entry wrong: %+v", g)
	}

	// Replacing one provider leaves the other alone.
	claude2 := ProviderLimits{Windows: []Window{{Minutes: 10080, Used: 7, ResetsAt: 1794000000}}, Updated: 1793700100}
	if err := UpdateLimits("claude", claude2); err != nil {
		t.Fatal(err)
	}
	got = readLimitsFile(t, dir)
	if g := got["claude"]; len(g.Windows) != 1 || g.Windows[0].Minutes != 10080 {
		t.Errorf("claude not replaced: %+v", g)
	}
	if g := got["codex"]; g.Plan != "pro" {
		t.Errorf("codex lost after claude update: %+v", g)
	}
}

func TestUpdateLimitsJSONShape(t *testing.T) {
	dir := useTempState(t)
	if err := UpdateLimits("claude", ProviderLimits{Windows: []Window{{Minutes: 300, Used: 1.5, ResetsAt: 9}}, Updated: 5}); err != nil {
		t.Fatal(err)
	}
	if err := UpdateLimits("codex", ProviderLimits{Updated: 6}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "limits.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"plan", "windows", "updated"} {
		if _, ok := raw["claude"][k]; !ok {
			t.Errorf("claude entry missing key %q: %s", k, b)
		}
	}
	var win []map[string]json.RawMessage
	if err := json.Unmarshal(raw["claude"]["windows"], &win); err != nil || len(win) != 1 {
		t.Fatalf("windows not an array of 1: %s", raw["claude"]["windows"])
	}
	for _, k := range []string{"minutes", "used", "resets_at"} {
		if _, ok := win[0][k]; !ok {
			t.Errorf("window missing key %q: %s", k, raw["claude"]["windows"])
		}
	}
	// A provider with no windows must serialise "windows" as [] (never null).
	if got := strings.TrimSpace(string(raw["codex"]["windows"])); got != "[]" {
		t.Errorf("empty windows = %s, want []", got)
	}
}

func TestUpdateLimitsConcurrentNoLostUpdate(t *testing.T) {
	dir := useTempState(t)
	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- UpdateLimits(fmt.Sprintf("p%d", i), ProviderLimits{Updated: int64(i)})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := readLimitsFile(t, dir); len(got) != n {
		t.Fatalf("lost updates: %d of %d providers present", len(got), n)
	}
}

func TestUpdateLimitsRecoversFromCorruptFile(t *testing.T) {
	dir := useTempState(t)
	Dir() // create it
	if err := os.WriteFile(filepath.Join(dir, "limits.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpdateLimits("claude", ProviderLimits{Updated: 1}); err != nil {
		t.Fatal(err)
	}
	if got := readLimitsFile(t, dir); got["claude"].Updated != 1 {
		t.Fatalf("not rewritten: %+v", got)
	}
}

func TestMergeLimitsSkipsUnchanged(t *testing.T) {
	dir := useTempState(t)
	l := ProviderLimits{Windows: []Window{{Minutes: 300, Used: 42, ResetsAt: 1793705100}}, Updated: 1}
	if wrote, err := MergeLimits("claude", l); err != nil || !wrote {
		t.Fatalf("first merge: wrote=%v err=%v", wrote, err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "limits.json"))
	l.Updated = 2 // only the timestamp moved
	if wrote, err := MergeLimits("claude", l); err != nil || wrote {
		t.Fatalf("unchanged windows must not rewrite: wrote=%v err=%v", wrote, err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "limits.json"))
	if string(before) != string(after) {
		t.Fatal("limits.json changed although windows did not")
	}
}

func TestMergeLimitsKeepsLargerUsedForSameReset(t *testing.T) {
	dir := useTempState(t)
	newer := ProviderLimits{Windows: []Window{{Minutes: 300, Used: 50, ResetsAt: 1793705100}}, Updated: 2}
	older := ProviderLimits{Windows: []Window{{Minutes: 300, Used: 40, ResetsAt: 1793705100}}, Updated: 3}
	if _, err := MergeLimits("claude", newer); err != nil {
		t.Fatal(err)
	}
	if _, err := MergeLimits("claude", older); err != nil {
		t.Fatal(err)
	}
	if g := readLimitsFile(t, dir)["claude"]; g.Windows[0].Used != 50 {
		t.Fatalf("stale session lowered used: %+v", g)
	}
	next := ProviderLimits{Windows: []Window{{Minutes: 300, Used: 3, ResetsAt: 1793723100}}, Updated: 4}
	if _, err := MergeLimits("claude", next); err != nil {
		t.Fatal(err)
	}
	if g := readLimitsFile(t, dir)["claude"]; g.Windows[0].Used != 3 || g.Windows[0].ResetsAt != 1793723100 {
		t.Fatalf("new window not taken: %+v", g)
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
	var all map[string]ProviderLimits
	if err := ReadJSON("limits.json", &all); err != nil {
		t.Fatal(err)
	}
	if all["codex"].Updated != 30 {
		t.Fatalf("latest lost: %+v", all)
	}
}
