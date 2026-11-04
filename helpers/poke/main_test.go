package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnknownSubcommandPrintsUsageAndExits2(t *testing.T) {
	for _, args := range [][]string{nil, {"bogus"}, {""}} {
		var out, errOut bytes.Buffer
		code := run(args, strings.NewReader(""), &out, &errOut)
		if code != 2 {
			t.Errorf("run(%q) = %d, want 2", args, code)
		}
		if !strings.Contains(errOut.String(), "usage: herdr-poke") {
			t.Errorf("run(%q) stderr lacks usage: %q", args, errOut.String())
		}
		if out.Len() != 0 {
			t.Errorf("run(%q) wrote to stdout: %q", args, out.String())
		}
	}
}

func TestUsageListsEverySubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	run([]string{"nope"}, strings.NewReader(""), &out, &errOut)
	for _, cmd := range []string{"statusline", "codex", "usage", "assets"} {
		if !strings.Contains(errOut.String(), cmd) {
			t.Errorf("usage does not mention %q: %q", cmd, errOut.String())
		}
	}
}

func TestStatuslineSubcommandAlwaysExitsZero(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"statusline"}, strings.NewReader("garbage"), &out, &errOut)
	if code != 0 || out.String() != "\n" || errOut.Len() != 0 {
		t.Errorf("code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
}

func TestCodexSubcommandRunsTheCommandAfterDashes(t *testing.T) {
	t.Setenv("HC_TERMINAL_ID", "")
	var out, errOut bytes.Buffer
	if code := run([]string{"codex", "--", "sh", "-c", "exit 4"}, strings.NewReader(""), &out, &errOut); code != 4 {
		t.Errorf("exit code %d, want the child's 4", code)
	}
}

func TestCodexSubcommandNeedsACommand(t *testing.T) {
	for _, args := range [][]string{{"codex"}, {"codex", "--"}} {
		var out, errOut bytes.Buffer
		if code := run(args, strings.NewReader(""), &out, &errOut); code != 2 {
			t.Errorf("run(%q) = %d, want 2", args, code)
		}
	}
}

func TestUsageSubcommandWritesUsageJSON(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("HERDR_POKE_STATE", stateDir)
	t.Setenv("HOME", t.TempDir()) // no logs: 14 zero days
	var out, errOut bytes.Buffer
	if code := run([]string{"usage"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
	b, err := os.ReadFile(filepath.Join(stateDir, "usage", "daily.json"))
	if err != nil || !strings.Contains(string(b), `"days"`) {
		t.Errorf("daily.json: err=%v content=%q", err, b)
	}
	if out.Len() != 0 {
		t.Errorf("usage printed to stdout: %q", out.String())
	}
}

func TestAssetsSubcommandListsTheAssetFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HERDR_APP_HOME", home)
	var out, errOut bytes.Buffer
	if code := run([]string{"assets", "--list"}, strings.NewReader(""), &out, &errOut); code != 0 || out.String() != "[]\n" {
		t.Errorf("empty: code %d out %q err %q", code, out.String(), errOut.String())
	}
	dir := filepath.Join(home, "poke-assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "25.gif"), []byte("GIF89a"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run([]string{"assets", "--list"}, strings.NewReader(""), &out, &errOut); code != 0 || out.String() != "[\"25.gif\"]\n" {
		t.Errorf("one file: code %d out %q", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(home, "mods")); err == nil {
		t.Error("assets touched mods/")
	}
}

func TestAssetsSubcommandWithoutArgsIsUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"assets"}, strings.NewReader(""), &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage") {
		t.Errorf("code %d stderr %q", code, errOut.String())
	}
}
