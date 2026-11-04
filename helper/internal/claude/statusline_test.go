package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"herdr-poke/internal/state"
)

// testNow must equal FAKE now in testdata/gen-golden.sh (now_ms=1793700000000).
var testNow = time.Unix(1793700000, 0)

const testHome = "/home/fixture"

// stubGit stands in for the real repos that gen-golden.sh creates: the
// directory name decides the answer.
func stubGit(dir string) (string, bool, bool) {
	switch filepath.Base(dir) {
	case "clean":
		return "main", false, true
	case "dirty":
		return "feat/x", true, true
	}
	return "", false, false
}

// loadFixture reads testdata/in-<name>.json (with __HOME__ replaced) and its
// optional in-<name>.env sidecar, mirroring gen-golden.sh.
func loadFixture(t *testing.T, name string) (Input, Env) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "in-"+name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.ReplaceAll(raw, []byte("__HOME__"), []byte(testHome))
	var in Input
	if err := json.Unmarshal(raw, &in); err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	env := Env{
		Home:           testHome,
		Cwd:            testHome + "/repos/nogit", // gen-golden.sh runs node here
		Now:            testNow,
		ClaudeJSONPath: filepath.Join("testdata", "claude.json"),
		Git:            stubGit,
	}
	side, err := os.ReadFile(filepath.Join("testdata", "in-"+name+".env"))
	if err == nil {
		for _, line := range strings.Split(string(side), "\n") {
			k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok {
				continue
			}
			switch k {
			case "CLAUDE_CODE_AUTO_COMPACT_WINDOW":
				env.AutoCompactWindow = parseAutoCompactWindow(v)
			case "ACCOUNT_FILE":
				if v == "-" {
					env.ClaudeJSONPath = filepath.Join(t.TempDir(), "no-such.json")
				} else {
					env.ClaudeJSONPath = filepath.Join("testdata", v)
				}
			default:
				t.Fatalf("fixture %s: unknown sidecar key %q", name, k)
			}
		}
	}
	return in, env
}

func TestRenderMatchesGolden(t *testing.T) {
	ins, err := filepath.Glob(filepath.Join("testdata", "in-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ins) < 25 {
		t.Fatalf("only %d fixtures found; run testdata/gen-golden.sh?", len(ins))
	}
	for _, path := range ins {
		name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "in-"), ".json")
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", "golden-"+name+".txt"))
			if err != nil {
				t.Fatalf("missing golden (run testdata/gen-golden.sh): %v", err)
			}
			in, env := loadFixture(t, name)
			got := Render(in, env)
			if got != string(want) {
				t.Errorf("output differs from Node golden\n got: %q\nwant: %q", got, want)
			}
			if strings.HasSuffix(got, "\n") {
				t.Errorf("output must not end with a newline: %q", got)
			}
		})
	}
}

func TestParseAutoCompactWindow(t *testing.T) {
	for in, want := range map[string]float64{
		"": 0, "abc": 0, "0": 0, "-5": 0, "NaN": 0, "Infinity": 0,
		"250000": 250000, " 250000 ": 250000, "1e5": 100000, "400000.5": 400000.5,
	} {
		if got := parseAutoCompactWindow(in); got != want {
			t.Errorf("parseAutoCompactWindow(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestJSRound(t *testing.T) {
	for in, want := range map[float64]float64{
		0.5: 1, 1.5: 2, 2.5: 3, -0.5: 0, -1.5: -1, 49.4999: 49, 79.5: 80, 0.49999999999999994: 0,
	} {
		if got := jsRound(in); got != want {
			t.Errorf("jsRound(%v) = %v, want %v (Math.round semantics)", in, got, want)
		}
	}
}

func TestLimitsFrom(t *testing.T) {
	pct := func(v float64) *float64 { return &v }
	now := testNow.Unix()

	t.Run("both windows", func(t *testing.T) {
		in := Input{RateLimits: &RateLimits{
			FiveHour: &RateLimit{UsedPercentage: pct(42.4), ResetsAt: float64(now + 5100)},
			SevenDay: &RateLimit{UsedPercentage: pct(81.6), ResetsAt: float64(now + 183840)},
		}}
		got, ok := limitsFrom(in, testNow)
		if !ok {
			t.Fatal("want limits")
		}
		want := []state.Window{{Minutes: 300, Used: 42.4, ResetsAt: now + 5100}, {Minutes: 10080, Used: 81.6, ResetsAt: now + 183840}}
		if fmt.Sprint(got.Windows) != fmt.Sprint(want) || got.Updated != now {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("expired and null windows are omitted", func(t *testing.T) {
		in := Input{RateLimits: &RateLimits{
			FiveHour: &RateLimit{UsedPercentage: pct(10), ResetsAt: float64(now)}, // resets_at <= now
			SevenDay: &RateLimit{UsedPercentage: pct(20), ResetsAt: float64(now + 60)},
		}}
		got, ok := limitsFrom(in, testNow)
		if !ok || len(got.Windows) != 1 || got.Windows[0].Minutes != 10080 {
			t.Errorf("got %+v ok=%v", got, ok)
		}
		in = Input{RateLimits: &RateLimits{FiveHour: &RateLimit{ResetsAt: float64(now + 60)}}} // used null
		if _, ok := limitsFrom(in, testNow); ok {
			t.Error("null used_percentage must not produce limits")
		}
	})
	t.Run("absent", func(t *testing.T) {
		if _, ok := limitsFrom(Input{}, testNow); ok {
			t.Error("no rate_limits must not produce limits")
		}
		if _, ok := limitsFrom(Input{RateLimits: &RateLimits{}}, testNow); ok {
			t.Error("empty rate_limits must not produce limits")
		}
	})
}

func TestLabelsFrom(t *testing.T) {
	in, env := loadFixture(t, "full")
	got := labelsFrom(in, env)
	want := map[string]string{"poke.model": "Opus 4.7", "poke.effort": "high", "poke.ctx": "45"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v want %v", got, want)
	}

	// A model without an effort param clears a stale effort label.
	in, env = loadFixture(t, "style-other")
	got = labelsFrom(in, env)
	if v, ok := got["poke.effort"]; !ok || v != "" {
		t.Errorf("effort must be present and empty, got %v", got)
	}
	if _, ok := got["poke.ctx"]; ok {
		t.Errorf("no usage data: ctx label must be absent, got %v", got)
	}

	// No model at all: leave labels alone.
	in, env = loadFixture(t, "empty")
	if got := labelsFrom(in, env); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

// ---- Run ----

func fakeHC(t *testing.T, extra string) (logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "calls.log")
	script := `#!/bin/sh
[ "$1" = warmup ] && exit 0
for a in "$@"; do printf '[%s]' "$a" >> '` + logPath + `'; done
printf '\n' >> '` + logPath + `'
` + extra + `
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "hc"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	// macOS scans a fresh executable on first run (~200 ms); do that outside the code under test.
	if err := exec.Command(filepath.Join(dir, "hc"), "warmup").Run(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func hcCalls(t *testing.T, logPath string) []string {
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

// runEnv isolates everything Run reads from the process environment.
func runEnv(t *testing.T) (stateDir string) {
	t.Helper()
	stateDir = filepath.Join(t.TempDir(), "state")
	t.Setenv("HERDR_POKE_STATE", stateDir)
	t.Setenv("HOME", t.TempDir()) // no .claude.json: no account segment
	t.Setenv("HC_TERMINAL_ID", "t7")
	t.Setenv("CLAUDE_CODE_AUTO_COMPACT_WINDOW", "")
	return stateDir
}

func liveInput(t *testing.T, resetPast bool) string {
	t.Helper()
	now := time.Now().Unix()
	// Resets sit mid-minute so the countdown text ("1h 25m") stays the same for 30 s:
	// two Runs a second apart must render identically even on a loaded machine.
	five := now + 5100 + 30
	if resetPast {
		five = now - 60
	}
	return fmt.Sprintf(`{"model":{"id":"claude-opus-4-7","display_name":"Opus 4.7"},"effort":{"level":"high"},
"cwd":%q,
"context_window":{"context_window_size":200000,"current_usage":{"input_tokens":10,"cache_creation_input_tokens":20000,"cache_read_input_tokens":70000}},
"rate_limits":{"five_hour":{"used_percentage":42.4,"resets_at":%d},"seven_day":{"used_percentage":81.6,"resets_at":%d}}}`,
		t.TempDir(), five, now+183840+30)
}

func TestRunWritesLabelsAndLimits(t *testing.T) {
	stateDir := runEnv(t)
	logPath := fakeHC(t, "")
	input := liveInput(t, false)

	var out bytes.Buffer
	if code := Run(strings.NewReader(input), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"Opus 4.7", ":high", "ctx 45%", "5h ", "7d "} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q: %q", want, out.String())
		}
	}
	if strings.HasSuffix(out.String(), "\n") {
		t.Errorf("trailing newline: %q", out.String())
	}

	got := hcCalls(t, logPath)
	if len(got) != 1 || got[0] != "[label][t7][poke.ctx=45][poke.effort=high][poke.model=Opus 4.7]" {
		t.Fatalf("hc calls = %q", got)
	}

	b, err := os.ReadFile(filepath.Join(stateDir, "limits.json"))
	if err != nil {
		t.Fatalf("limits.json: %v", err)
	}
	var lim map[string]state.ProviderLimits
	if err := json.Unmarshal(b, &lim); err != nil {
		t.Fatal(err)
	}
	c := lim["claude"]
	if len(c.Windows) != 2 || c.Windows[0].Minutes != 300 || c.Windows[0].Used != 42.4 ||
		c.Windows[1].Minutes != 10080 || c.Windows[1].Used != 81.6 {
		t.Errorf("claude limits = %+v", c)
	}
	if d := time.Now().Unix() - c.Updated; d < 0 || d > 5 {
		t.Errorf("updated = %d, not ~now", c.Updated)
	}

	// Same input again: output is stable and no label call is made.
	var out2 bytes.Buffer
	Run(strings.NewReader(input), &out2)
	if out2.String() != out.String() {
		t.Errorf("second run differs: %q vs %q", out2.String(), out.String())
	}
	if got := hcCalls(t, logPath); len(got) != 1 {
		t.Errorf("unchanged labels still called hc: %q", got)
	}
}

func TestRunOmitsExpiredWindow(t *testing.T) {
	stateDir := runEnv(t)
	fakeHC(t, "")
	var out bytes.Buffer
	Run(strings.NewReader(liveInput(t, true)), &out)
	b, err := os.ReadFile(filepath.Join(stateDir, "limits.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lim map[string]state.ProviderLimits
	if err := json.Unmarshal(b, &lim); err != nil {
		t.Fatal(err)
	}
	if w := lim["claude"].Windows; len(w) != 1 || w[0].Minutes != 10080 {
		t.Errorf("windows = %+v, want only 7d", w)
	}
}

func TestRunWithoutTerminalOrRateLimits(t *testing.T) {
	stateDir := runEnv(t)
	t.Setenv("HC_TERMINAL_ID", "")
	logPath := fakeHC(t, "")
	var out bytes.Buffer
	Run(strings.NewReader(`{"model":{"id":"m","display_name":"M"},"cwd":"/nonexistent"}`), &out)
	if got := hcCalls(t, logPath); got != nil {
		t.Errorf("no HC_TERMINAL_ID but hc called: %q", got)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "limits.json")); !os.IsNotExist(err) {
		t.Errorf("no rate_limits must not create limits.json (err=%v)", err)
	}
	if !strings.Contains(out.String(), "M") {
		t.Errorf("output = %q", out.String())
	}
}

func TestRunBadJSONStillPrints(t *testing.T) {
	stateDir := runEnv(t)
	logPath := fakeHC(t, "")
	for _, garbage := range []string{"not json at all", `{"model":`, "  ", "\x00\x01"} {
		var out bytes.Buffer
		if code := Run(strings.NewReader(garbage), &out); code != 0 {
			t.Errorf("Run(%q) exit %d, want 0", garbage, code)
		}
		if out.String() != "\n" {
			t.Errorf("Run(%q) printed %q, want an empty line", garbage, out.String())
		}
	}
	if got := hcCalls(t, logPath); got != nil {
		t.Errorf("bad input must not touch labels: %q", got)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "limits.json")); !os.IsNotExist(err) {
		t.Errorf("bad input must not write limits (err=%v)", err)
	}
}

func TestRunEmptyStdinBehavesLikeEmptyObject(t *testing.T) {
	runEnv(t)
	fakeHC(t, "")
	var out bytes.Buffer
	if code := Run(strings.NewReader(""), &out); code != 0 {
		t.Fatal(code)
	}
	cwd, _ := os.Getwd()
	if !strings.Contains(out.String(), filepath.Base(cwd)) {
		t.Errorf("empty stdin should print the cwd segment, got %q", out.String())
	}
}

func TestRunTypeMismatchStillPrints(t *testing.T) {
	runEnv(t)
	fakeHC(t, "")
	var out bytes.Buffer
	// effort.level has the wrong type; the rest of the line must survive.
	Run(strings.NewReader(`{"model":{"display_name":"Opus"},"effort":{"level":5},"cwd":"/nonexistent"}`), &out)
	if !strings.Contains(out.String(), "Opus") {
		t.Errorf("output = %q", out.String())
	}
}

type firstWriteTimer struct {
	start time.Time
	at    time.Duration
	buf   bytes.Buffer
}

func (w *firstWriteTimer) Write(p []byte) (int, error) {
	if w.at == 0 {
		w.at = time.Since(w.start)
	}
	return w.buf.Write(p)
}

func TestRunHungHCDoesNotDelayOutput(t *testing.T) {
	runEnv(t)
	fakeHC(t, "exec sleep 5")
	w := &firstWriteTimer{start: time.Now()}
	Run(strings.NewReader(liveInput(t, false)), w)
	total := time.Since(w.start)

	if w.at > 100*time.Millisecond {
		t.Errorf("output written after %v; labels must not precede it", w.at)
	}
	if total > 2*time.Second {
		t.Errorf("Run took %v with a hung hc", total)
	}
	if !strings.Contains(w.buf.String(), "Opus 4.7") {
		t.Errorf("output = %q", w.buf.String())
	}
}

// ---- real git ----

func TestGitInfoRealRepos(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	env := []string{
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_CEILING_DIRECTORIES=" + root,
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
	}
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		t.Setenv(k, v)
	}
	git := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	mk := func(name, branch string) string {
		d := filepath.Join(root, name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		git(d, "init", "-q", "-b", branch)
		if err := os.WriteFile(filepath.Join(d, "f"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		git(d, "add", "f")
		git(d, "commit", "-q", "-m", "init")
		return d
	}

	clean := mk("clean", "main")
	if b, dirty, ok := gitInfo(clean); !ok || b != "main" || dirty {
		t.Errorf("clean: %q %v %v", b, dirty, ok)
	}

	dirtyRepo := mk("dirty", "feat/x")
	if err := os.WriteFile(filepath.Join(dirtyRepo, "new"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, dirty, ok := gitInfo(dirtyRepo); !ok || b != "feat/x" || !dirty {
		t.Errorf("dirty: %q %v %v", b, dirty, ok)
	}

	det := mk("detached", "main")
	sha := git(det, "rev-parse", "--short", "HEAD")
	git(det, "checkout", "-q", "--detach")
	if b, dirty, ok := gitInfo(det); !ok || b != sha || dirty {
		t.Errorf("detached: %q (want %q) %v %v", b, sha, dirty, ok)
	}

	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := gitInfo(plain); ok {
		t.Error("plain dir reported as repo")
	}
	if _, _, ok := gitInfo(filepath.Join(root, "missing")); ok {
		t.Error("missing dir reported as repo")
	}
}
