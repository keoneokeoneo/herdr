package labels

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeHC puts a stub `hc` first on PATH. Every call appends one line to the
// returned log, with each argv element in [brackets]. If the file "fail"
// exists next to the stub, it exits 1 after logging.
func fakeHC(t *testing.T, extra string) (logPath, failPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "calls.log")
	failPath = filepath.Join(dir, "fail")
	script := `#!/bin/sh
[ "$1" = warmup ] && exit 0
for a in "$@"; do printf '[%s]' "$a" >> '` + logPath + `'; done
printf '\n' >> '` + logPath + `'
` + extra + `
if [ -e '` + failPath + `' ]; then exit 1; fi
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "hc"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	// macOS scans a freshly written executable on its first run (~200 ms),
	// which would trip Timeout. Run it once now, outside the code under test.
	if err := exec.Command(filepath.Join(dir, "hc"), "warmup").Run(); err != nil {
		t.Fatalf("warm up fake hc: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HERDR_POKE_STATE", filepath.Join(t.TempDir(), "state"))
	return logPath, failPath
}

func calls(t *testing.T, logPath string) []string {
	t.Helper()
	b, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func TestLabelsSkipsUnchanged(t *testing.T) {
	logPath, _ := fakeHC(t, "")

	if err := Set("t7", map[string]string{"poke.model": "Opus 4.7", "poke.ctx": "12"}); err != nil {
		t.Fatal(err)
	}
	got := calls(t, logPath)
	if len(got) != 1 || got[0] != "[label][t7][poke.ctx=12][poke.model=Opus 4.7]" {
		t.Fatalf("first Set calls = %q", got)
	}

	// Identical values: no call at all.
	if err := Set("t7", map[string]string{"poke.model": "Opus 4.7", "poke.ctx": "12"}); err != nil {
		t.Fatal(err)
	}
	if got := calls(t, logPath); len(got) != 1 {
		t.Fatalf("identical Set made a call: %q", got)
	}

	// One value changed: exactly one call carrying only that key.
	if err := Set("t7", map[string]string{"poke.model": "Opus 4.7", "poke.ctx": "13"}); err != nil {
		t.Fatal(err)
	}
	got = calls(t, logPath)
	if len(got) != 2 || got[1] != "[label][t7][poke.ctx=13]" {
		t.Fatalf("changed Set calls = %q", got)
	}
}

func TestLabelsCachePerTerminal(t *testing.T) {
	logPath, _ := fakeHC(t, "")
	if err := Set("t7", map[string]string{"poke.ctx": "5"}); err != nil {
		t.Fatal(err)
	}
	// Another terminal has its own cache, so the same value is sent again.
	if err := Set("t8", map[string]string{"poke.ctx": "5"}); err != nil {
		t.Fatal(err)
	}
	got := calls(t, logPath)
	if len(got) != 2 || got[1] != "[label][t8][poke.ctx=5]" {
		t.Fatalf("calls = %q", got)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HERDR_POKE_STATE"), "labels-t7.json")); err != nil {
		t.Fatalf("cache file labels-t7.json missing: %v", err)
	}
}

func TestLabelsEmptyTerminalIsNoop(t *testing.T) {
	logPath, _ := fakeHC(t, "")
	if err := Set("", map[string]string{"poke.model": "x"}); err != nil {
		t.Fatalf("Set with empty terminal: %v", err)
	}
	if got := calls(t, logPath); got != nil {
		t.Fatalf("hc was called: %q", got)
	}
}

func TestLabelsEmptyValueRemovesOnce(t *testing.T) {
	logPath, _ := fakeHC(t, "")
	for i := 0; i < 2; i++ {
		if err := Set("t7", map[string]string{"poke.effort": ""}); err != nil {
			t.Fatal(err)
		}
	}
	got := calls(t, logPath)
	if len(got) != 1 || got[0] != "[label][t7][poke.effort=]" {
		t.Fatalf("calls = %q", got)
	}
	// Setting it again after removal is a change.
	if err := Set("t7", map[string]string{"poke.effort": "high"}); err != nil {
		t.Fatal(err)
	}
	if got := calls(t, logPath); len(got) != 2 || got[1] != "[label][t7][poke.effort=high]" {
		t.Fatalf("calls = %q", got)
	}
}

func TestLabelsFailureIsRetried(t *testing.T) {
	logPath, failPath := fakeHC(t, "")
	if err := os.WriteFile(failPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Set("t7", map[string]string{"poke.ctx": "9"}); err == nil {
		t.Fatal("want error when hc exits non-zero")
	}
	if err := os.Remove(failPath); err != nil {
		t.Fatal(err)
	}
	if err := Set("t7", map[string]string{"poke.ctx": "9"}); err != nil {
		t.Fatal(err)
	}
	if got := calls(t, logPath); len(got) != 2 {
		t.Fatalf("failed write must not be cached; calls = %q", got)
	}
}

func TestLabelsHungHCDoesNotBlock(t *testing.T) {
	_, _ = fakeHC(t, "exec sleep 5")
	old := Timeout
	Timeout = 100 * time.Millisecond
	t.Cleanup(func() { Timeout = old })

	start := time.Now()
	err := Set("t7", map[string]string{"poke.ctx": "1"})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("want error from a hung hc")
	}
	if elapsed > time.Second {
		t.Fatalf("Set blocked for %v", elapsed)
	}
}

func TestLabelsMissingHCReturnsError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HERDR_POKE_STATE", filepath.Join(t.TempDir(), "state"))
	if err := Set("t7", map[string]string{"poke.ctx": "1"}); err == nil {
		t.Fatal("want error when hc is not on PATH")
	}
}
