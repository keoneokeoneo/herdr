package labels

import (
	"encoding/json"
	"herdr-poke/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTelemetryClearsPreviousContextAndIdentifiesOwner(t *testing.T) {
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	// Inspect the generated complete snapshot independently of hc.
	got := Telemetry(42, "codex", "new", map[string]string{"poke.model": "new-model"})
	if got["poke.ctx"] != "" || got["poke.effort"] != "" || got["poke.owner"] != "42" || got["poke.provider"] != "codex" || got["poke.session"] != "new" {
		t.Fatalf("snapshot: %v", got)
	}
	if _, ok := got["poke.ctx"]; !ok {
		t.Fatal("context not explicitly cleared")
	}
}

func TestNewOwnerRepublishesUnchangedValues(t *testing.T) {
	oldTimeout := Timeout
	Timeout = 5 * time.Second
	t.Cleanup(func() { Timeout = oldTimeout })
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	os.WriteFile(filepath.Join(bin, "hc"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$LOG\"\n"), 0755)
	t.Setenv("LOG", log)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := Telemetry(1, "codex", "s", map[string]string{"poke.model": "same"})
	b, _ := json.Marshal(old)
	os.WriteFile(filepath.Join(state.Dir(), cacheName("t1")), b, 0600)
	if err := Set("t1", Telemetry(2, "codex", "s", map[string]string{"poke.model": "same"})); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(log)
	if !strings.Contains(string(raw), "poke.model=same") {
		t.Fatalf("new owner needs complete publication: %s", raw)
	}
}

func TestOwnerRejectsDifferentForeground(t *testing.T) {
	for _, raw := range []string{`{"foreground":null}`, `{"foreground":{"pid":42,"argv":["zsh"]}}`, `{"foreground":{"pid":42,"argv":["codex"]}}`, `broken`} {
		if got := parseOwner([]byte(raw), "claude"); got != 0 {
			t.Errorf("accepted wrong owner %d: %s", got, raw)
		}
	}
	if got := parseOwner([]byte(`{"foreground":{"pid":42,"argv":["/bin/claude"]}}`), "claude"); got != 42 {
		t.Errorf("owner=%d", got)
	}
}

func TestAncestorRejectsReplacementProcess(t *testing.T) {
	tree := []byte("100 1\n200 100\n300 200\n400 1\n")
	if !ancestor(100, 300, tree) || ancestor(400, 300, tree) || ancestor(100, 500, tree) {
		t.Fatal("wrong invocation owner")
	}
}
