package main

import (
	"bytes"
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
