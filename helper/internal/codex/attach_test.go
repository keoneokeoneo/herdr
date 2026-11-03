package codex

import (
	"testing"
)

func TestParseForegroundCodex(t *testing.T) {
	j := `{"address":"t1g","foreground":{"argv":["codex","--no-daemon","-m","gpt-6-astra"],"cwd":"/w","pid":4242}}`
	fg, ok := parseForeground([]byte(j))
	if !ok || fg.Pid != 4242 || fg.Cwd != "/w" {
		t.Fatalf("got %+v ok=%v", fg, ok)
	}
}

func TestParseForegroundFullPathCodex(t *testing.T) {
	j := `{"foreground":{"argv":["/opt/homebrew/bin/codex","--yolo"],"cwd":"/w","pid":7}}`
	if _, ok := parseForeground([]byte(j)); !ok {
		t.Fatal("absolute codex path not recognised")
	}
}

func TestParseForegroundSkipsWrapperAndOthers(t *testing.T) {
	for _, j := range []string{
		// Already wrapped: the wrapper publishes, attaching would double it.
		`{"foreground":{"argv":["/Users/k/.local/bin/herdr-poke","codex","--","codex"],"cwd":"/w","pid":7}}`,
		`{"foreground":{"argv":["claude","--model","sonnet"],"cwd":"/w","pid":7}}`,
		`{"foreground":null}`,
		`{"foreground":{"argv":["codex"],"cwd":"/w","pid":0}}`,
		`not json`,
	} {
		if fg, ok := parseForeground([]byte(j)); ok {
			t.Errorf("%s: attached to %+v", j, fg)
		}
	}
}

func TestAttachLockIsExclusive(t *testing.T) {
	t.Setenv("HERDR_POKE_STATE", t.TempDir())
	release, ok := tryTerminalLock("t9")
	if !ok {
		t.Fatal("first lock refused")
	}
	if _, ok := tryTerminalLock("t9"); ok {
		t.Fatal("second lock granted while first is held")
	}
	release()
	r2, ok := tryTerminalLock("t9")
	if !ok {
		t.Fatal("lock not free after release")
	}
	r2()
}

func TestAttachRejectsBadTerminal(t *testing.T) {
	if got := Attach(""); got != 2 {
		t.Errorf("Attach(\"\") = %d, want 2", got)
	}
}
