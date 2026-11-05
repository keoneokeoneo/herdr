package assets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	gifBytes = "GIF89a\x01\x00\x01\x00\x00\x00\x00;"
	pngBytes = "\x89PNG\r\n\x1a\n\x00\x00"
)

// fakeHost serves the three hosts under one httptest server and counts requests.
type fakeHost struct {
	srv *httptest.Server
	mu  sync.Mutex
	got []string
	// species maps id -> JSON body of /pokemon-species/<id>.
	species map[string]string
	// missing makes these paths answer 404.
	missing map[string]bool
	// override replaces the body for a path (status 200).
	override map[string]string
}

func newFakeHost(t *testing.T) *fakeHost {
	t.Helper()
	h := &fakeHost{
		species: map[string]string{
			"25":  `{"name":"pikachu","names":[{"name":"ピカチュウ","language":{"name":"ja"}},{"name":"Pikachu","language":{"name":"en"}}]}`,
			"122": `{"name":"mr-mime","names":[{"name":"Mr. Mime","language":{"name":"en"}}]}`,
			"700": `{"name":"sylveon","names":[{"name":"Sylveon","language":{"name":"en"}}]}`,
			"900": `{"name":"kleavor","names":[]}`, // no English name: derive one
		},
		missing:  map[string]bool{},
		override: map[string]string{},
	}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.got = append(h.got, r.URL.Path)
		missing, override := h.missing[r.URL.Path], h.override[r.URL.Path]
		h.mu.Unlock()
		switch {
		case missing:
			http.NotFound(w, r)
		case override != "":
			fmt.Fprint(w, override)
		case strings.HasPrefix(r.URL.Path, "/sprites/versions/generation-v/black-white/animated/") && strings.HasSuffix(r.URL.Path, ".gif"):
			fmt.Fprint(w, gifBytes)
		case strings.HasPrefix(r.URL.Path, "/sprites/") && strings.HasSuffix(r.URL.Path, ".png"):
			fmt.Fprint(w, pngBytes)
		case strings.HasPrefix(r.URL.Path, "/species/"):
			body, ok := h.species[strings.TrimPrefix(r.URL.Path, "/species/")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *fakeHost) requests() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.got...)
}

func (h *fakeHost) config(t *testing.T) Config {
	t.Helper()
	return Config{
		Home:        t.TempDir(),
		PTBDir:      t.TempDir(),
		SpriteBase:  h.srv.URL + "/sprites",
		SpeciesBase: h.srv.URL + "/species",
	}
}

func namesJSON(t *testing.T, c Config) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(c.Dir(), "names.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("names.json: %v\n%s", err, b)
	}
	return m
}

func TestAssetsGifUnder650(t *testing.T) {
	h := newFakeHost(t)
	c := h.config(t)

	rel, err := Ensure(c, 25)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "poke-assets/25.gif" {
		t.Errorf("path = %q, want poke-assets/25.gif (relative to the app home)", rel)
	}
	if b, _ := os.ReadFile(filepath.Join(c.Home, rel)); string(b) != gifBytes {
		t.Errorf("sprite content = %q", b)
	}
	if _, err := os.Stat(filepath.Join(c.Dir(), "25.mp3")); err == nil {
		t.Error("a cry was saved; Poke no longer has cries")
	}
	if got := namesJSON(t, c); !reflect.DeepEqual(got, map[string]string{"25": "Pikachu"}) {
		t.Errorf("names.json = %v", got)
	}
	want := []string{"/sprites/versions/generation-v/black-white/animated/25.gif", "/species/25"}
	got := h.requests()
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
}

func TestAssetsPngAbove649(t *testing.T) {
	h := newFakeHost(t)
	c := h.config(t)
	rel, err := Ensure(c, 700)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "poke-assets/700.png" {
		t.Errorf("path = %q, want poke-assets/700.png", rel)
	}
	if b, _ := os.ReadFile(filepath.Join(c.Home, rel)); string(b) != pngBytes {
		t.Errorf("sprite content = %q", b)
	}
	for _, r := range h.requests() {
		if strings.Contains(r, "animated") {
			t.Errorf("asked for an animated sprite of #700: %s", r)
		}
	}
	if _, err := os.Stat(filepath.Join(c.Dir(), "700.gif")); err == nil {
		t.Error("700.gif exists")
	}
}

func TestAssetsBoundary649And650(t *testing.T) {
	h := newFakeHost(t)
	h.species["649"] = `{"name":"genesect","names":[{"name":"Genesect","language":{"name":"en"}}]}`
	h.species["650"] = `{"name":"chespin","names":[{"name":"Chespin","language":{"name":"en"}}]}`
	c := h.config(t)
	if rel, err := Ensure(c, 649); err != nil || rel != "poke-assets/649.gif" {
		t.Errorf("649: %q %v", rel, err)
	}
	if rel, err := Ensure(c, 650); err != nil || rel != "poke-assets/650.png" {
		t.Errorf("650: %q %v", rel, err)
	}
}

func TestAssetsIdempotent(t *testing.T) {
	h := newFakeHost(t)
	c := h.config(t)
	if _, err := Ensure(c, 25); err != nil {
		t.Fatal(err)
	}
	before := h.requests()
	mtimes := map[string]time.Time{}
	entries, _ := os.ReadDir(c.Dir())
	for _, e := range entries {
		info, _ := e.Info()
		mtimes[e.Name()] = info.ModTime()
	}
	time.Sleep(20 * time.Millisecond) // a rewrite would show in the mtime

	rel, err := Ensure(c, 25)
	if err != nil || rel != "poke-assets/25.gif" {
		t.Fatalf("second call: %q %v", rel, err)
	}
	if after := h.requests(); len(after) != len(before) {
		t.Errorf("second call made requests: %v", after[len(before):])
	}
	entries, _ = os.ReadDir(c.Dir())
	if len(entries) != len(mtimes) {
		t.Errorf("files changed: %d now, %d before", len(entries), len(mtimes))
	}
	for _, e := range entries {
		info, _ := e.Info()
		if !info.ModTime().Equal(mtimes[e.Name()]) {
			t.Errorf("%s was rewritten", e.Name())
		}
	}
}

func TestAssetsNameNeededOnlyAfterSpriteExists(t *testing.T) {
	// A half-finished earlier run left the sprite: only the missing name is fetched.
	h := newFakeHost(t)
	c := h.config(t)
	os.MkdirAll(c.Dir(), 0o755)
	os.WriteFile(filepath.Join(c.Dir(), "25.gif"), []byte(gifBytes), 0o644)
	if _, err := Ensure(c, 25); err != nil {
		t.Fatal(err)
	}
	if got := h.requests(); !reflect.DeepEqual(got, []string{"/species/25"}) {
		t.Errorf("requests = %v, want only the species lookup", got)
	}
}

func TestAssetsNameUsesEnglishSpeciesName(t *testing.T) {
	h := newFakeHost(t)
	c := h.config(t)
	if _, err := Ensure(c, 122); err != nil {
		t.Fatal(err)
	}
	if got := namesJSON(t, c)["122"]; got != "Mr. Mime" {
		t.Errorf("name = %q", got)
	}
}

func TestAssetsNameFallsBackToTitleCase(t *testing.T) {
	h := newFakeHost(t)
	c := h.config(t)
	if _, err := Ensure(c, 900); err != nil {
		t.Fatal(err)
	}
	if got := namesJSON(t, c)["900"]; got != "Kleavor" {
		t.Errorf("name = %q, want Kleavor", got)
	}
}

func TestAssetsNamesJSONKeepsOtherEntries(t *testing.T) {
	h := newFakeHost(t)
	c := h.config(t)
	os.MkdirAll(c.Dir(), 0o755)
	os.WriteFile(filepath.Join(c.Dir(), "names.json"), []byte(`{"1":"Bulbasaur"}`), 0o644)
	if _, err := Ensure(c, 25); err != nil {
		t.Fatal(err)
	}
	if got := namesJSON(t, c); !reflect.DeepEqual(got, map[string]string{"1": "Bulbasaur", "25": "Pikachu"}) {
		t.Errorf("names.json = %v", got)
	}
}

func TestAssetsGifMissingFallsBackToPng(t *testing.T) {
	h := newFakeHost(t)
	h.missing["/sprites/versions/generation-v/black-white/animated/25.gif"] = true
	c := h.config(t)
	rel, err := Ensure(c, 25)
	if err != nil || rel != "poke-assets/25.png" {
		t.Errorf("Ensure = %q, %v; want the png", rel, err)
	}
}

func TestAssetsRejectsNonImageBody(t *testing.T) {
	h := newFakeHost(t)
	h.override["/sprites/versions/generation-v/black-white/animated/25.gif"] = "<html>rate limited</html>"
	h.override["/sprites/25.png"] = "<html>rate limited</html>"
	c := h.config(t)
	if _, err := Ensure(c, 25); err == nil {
		t.Error("no error for an HTML body")
	}
	for _, name := range []string{"25.gif", "25.png"} {
		if _, err := os.Stat(filepath.Join(c.Dir(), name)); err == nil {
			t.Errorf("%s saved from an HTML body", name)
		}
	}
	entries, _ := os.ReadDir(c.Dir())
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestAssetsNameFailureKeepsSprite(t *testing.T) {
	h := newFakeHost(t)
	h.missing["/species/25"] = true
	c := h.config(t)
	rel, err := Ensure(c, 25)
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("err = %v, want one about the name", err)
	}
	if rel != "poke-assets/25.gif" {
		t.Errorf("path = %q: the sprite is there, so it must still be reported", rel)
	}
}

func TestAssetsBadID(t *testing.T) {
	c := newFakeHost(t).config(t)
	for _, id := range []int{0, -3, 100000} {
		if _, err := Ensure(c, id); err == nil {
			t.Errorf("Ensure(%d) succeeded", id)
		}
	}
}

func writePTB(t *testing.T, c Config, state string, sprites map[string]string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(c.PTBDir, "companion-state.json"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(c.PTBDir, "sprites")
	os.MkdirAll(dir, 0o755)
	for name, body := range sprites {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompanionCopyShiny(t *testing.T) {
	c := newFakeHost(t).config(t)
	writePTB(t, c, `{"active":{"baseID":52,"pathIDs":[52,53],"isShiny":true}}`, map[string]string{
		"53-sha.gif": "shiny-gif", "53-a.gif": "normal-gif", "53-s.png": "normal-png", "53-shs.png": "shiny-png",
		"52-sha.gif": "wrong-stage",
	})
	rel, err := Companion(c)
	if err != nil || rel != "poke-assets/companion-53.gif" {
		t.Fatalf("Companion = %q, %v", rel, err)
	}
	if b, _ := os.ReadFile(filepath.Join(c.Home, rel)); string(b) != "shiny-gif" {
		t.Errorf("copied %q, want the shiny gif of the last path id", b)
	}
}

func TestCompanionPicksSpriteByPreference(t *testing.T) {
	for _, tc := range []struct {
		name    string
		shiny   bool
		sprites map[string]string
		want    string // "" = error
		content string
	}{
		{"normal gif", false, map[string]string{"7-a.gif": "g", "7-s.png": "p", "7-sha.gif": "sg"}, "companion-7.gif", "g"},
		{"normal png fallback", false, map[string]string{"7-s.png": "p", "7-shs.png": "sp"}, "companion-7.png", "p"},
		{"shiny without shiny gif uses the gif", true, map[string]string{"7-a.gif": "g", "7-s.png": "p"}, "companion-7.gif", "g"},
		{"shiny png when no gif at all", true, map[string]string{"7-s.png": "p", "7-shs.png": "sp"}, "companion-7.png", "sp"},
		{"shiny falls back to normal png", true, map[string]string{"7-s.png": "p"}, "companion-7.png", "p"},
		{"nothing for this id", false, map[string]string{"8-a.gif": "g"}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeHost(t).config(t)
			shiny := "false"
			if tc.shiny {
				shiny = "true"
			}
			writePTB(t, c, `{"active":{"baseID":7,"pathIDs":[7],"isShiny":`+shiny+`}}`, tc.sprites)
			rel, err := Companion(c)
			if tc.want == "" {
				if err == nil {
					t.Errorf("Companion = %q, want an error", rel)
				}
				return
			}
			if err != nil || rel != "poke-assets/"+tc.want {
				t.Fatalf("Companion = %q, %v; want %s", rel, err, tc.want)
			}
			if b, _ := os.ReadFile(filepath.Join(c.Dir(), tc.want)); string(b) != tc.content {
				t.Errorf("copied %q, want %q", b, tc.content)
			}
		})
	}
}

func TestCompanionUsesBaseIDWithoutPath(t *testing.T) {
	c := newFakeHost(t).config(t)
	writePTB(t, c, `{"active":{"baseID":25,"isShiny":false}}`, map[string]string{"25-a.gif": "g"})
	if rel, err := Companion(c); err != nil || rel != "poke-assets/companion-25.gif" {
		t.Errorf("Companion = %q, %v", rel, err)
	}
	writePTB(t, c, `{"active":{"baseID":25,"pathIDs":[],"isShiny":false}}`, nil)
	if rel, err := Companion(c); err != nil || rel != "poke-assets/companion-25.gif" {
		t.Errorf("empty pathIDs: %q, %v", rel, err)
	}
}

func TestCompanionIdempotentAndReplacesChanged(t *testing.T) {
	c := newFakeHost(t).config(t)
	writePTB(t, c, `{"active":{"baseID":7,"pathIDs":[7],"isShiny":false}}`, map[string]string{"7-a.gif": "v1"})
	rel, err := Companion(c)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(c.Home, rel)
	info1, _ := os.Stat(dst)
	time.Sleep(20 * time.Millisecond)
	if _, err := Companion(c); err != nil {
		t.Fatal(err)
	}
	if info2, _ := os.Stat(dst); !info2.ModTime().Equal(info1.ModTime()) {
		t.Error("an unchanged companion sprite was rewritten")
	}
	// The source sprite changes (e.g. a new form art): the copy follows.
	os.WriteFile(filepath.Join(c.PTBDir, "sprites", "7-a.gif"), []byte("v2"), 0o644)
	if _, err := Companion(c); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "v2" {
		t.Errorf("copy = %q, want v2", b)
	}
	entries, _ := os.ReadDir(c.Dir())
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestCompanionErrors(t *testing.T) {
	c := newFakeHost(t).config(t)
	if _, err := Companion(c); err == nil {
		t.Error("no PokeTokenBar folder: want an error")
	}
	writePTB(t, c, `{"dex":[]}`, nil)
	if _, err := Companion(c); err == nil {
		t.Error("no active companion: want an error")
	}
	writePTB(t, c, `not json`, nil)
	if _, err := Companion(c); err == nil {
		t.Error("corrupt state: want an error")
	}
}

func TestCompanionNeverWritesPokeTokenBar(t *testing.T) {
	c := newFakeHost(t).config(t)
	writePTB(t, c, `{"active":{"baseID":7,"pathIDs":[7],"isShiny":false}}`, map[string]string{"7-a.gif": "v1"})
	before := snapshotDir(t, c.PTBDir)
	if _, err := Companion(c); err != nil {
		t.Fatal(err)
	}
	if after := snapshotDir(t, c.PTBDir); !reflect.DeepEqual(before, after) {
		t.Errorf("PokeTokenBar folder changed:\nbefore %v\nafter  %v", before, after)
	}
}

func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			b, _ := os.ReadFile(p)
			m[p] = fmt.Sprintf("%s|%v", b, info.ModTime())
		}
		return nil
	})
	return m
}

func TestListPrintsFilenames(t *testing.T) {
	c := Config{Home: t.TempDir()}
	if got, err := List(c); err != nil || !bytes.Equal(mustJSON(t, got), []byte(`[]`)) {
		t.Errorf("missing dir: %v %v, want []", got, err)
	}
	os.MkdirAll(c.Dir(), 0o755)
	for _, n := range []string{"25.gif", "700.png", "25.mp3", "names.json", "companion-53.gif", ".tmp-123"} {
		os.WriteFile(filepath.Join(c.Dir(), n), []byte("x"), 0o644)
	}
	os.MkdirAll(filepath.Join(c.Dir(), "subdir"), 0o755)
	got, err := List(c)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"25.gif", "25.mp3", "700.png", "companion-53.gif", "names.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List = %v, want %v", got, want)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("HERDR_APP_HOME", "/custom/home")
	t.Setenv("HERDR_POKE_SPRITE_BASE", "http://sprites.test")
	t.Setenv("HERDR_POKE_SPECIES_BASE", "http://species.test")
	t.Setenv("HERDR_POKE_PTB_DIR", "/custom/ptb")
	c := ConfigFromEnv()
	if c.Home != "/custom/home" || c.Dir() != "/custom/home/poke-assets" || c.PTBDir != "/custom/ptb" ||
		c.SpriteBase != "http://sprites.test" || c.SpeciesBase != "http://species.test" {
		t.Errorf("config = %+v", c)
	}

	t.Setenv("HERDR_APP_HOME", "")
	t.Setenv("HERDR_POKE_SPRITE_BASE", "")
	t.Setenv("HERDR_POKE_PTB_DIR", "")
	home, _ := os.UserHomeDir()
	c = ConfigFromEnv()
	if c.Home != filepath.Join(home, ".config", "herdr-app") ||
		c.SpriteBase != "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon" ||
		c.PTBDir != filepath.Join(home, "Library", "Application Support", "PokeTokenBar") {
		t.Errorf("default config = %+v", c)
	}
	if c.SpeciesBase != "http://species.test" { // only the ones that were emptied fall back
		t.Errorf("SpeciesBase = %q", c.SpeciesBase)
	}
}

func TestRunCLI(t *testing.T) {
	h := newFakeHost(t)
	c := h.config(t)
	var out, errOut bytes.Buffer

	if code := Run([]string{"25"}, c, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != "poke-assets/25.gif" {
		t.Errorf("assets 25: code %d out %q err %q", code, out.String(), errOut.String())
	}

	out.Reset()
	if code := Run([]string{"--list"}, c, &out, &errOut); code != 0 {
		t.Errorf("--list code %d: %s", code, errOut.String())
	}
	var names []string
	if err := json.Unmarshal(out.Bytes(), &names); err != nil || !reflect.DeepEqual(names, []string{"25.gif", "names.json"}) {
		t.Errorf("--list out %q (%v)", out.String(), err)
	}

	writePTB(t, c, `{"active":{"baseID":25,"pathIDs":[25],"isShiny":false}}`, map[string]string{"25-a.gif": "g"})
	out.Reset()
	if code := Run([]string{"--companion"}, c, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != "poke-assets/companion-25.gif" {
		t.Errorf("--companion: code %d out %q err %q", code, out.String(), errOut.String())
	}

	for _, args := range [][]string{nil, {"abc"}, {"25", "26"}, {"--bogus"}, {"--list", "x"}} {
		out.Reset()
		errOut.Reset()
		if code := Run(args, c, &out, &errOut); code != 2 || out.Len() != 0 || !strings.Contains(errOut.String(), "usage") {
			t.Errorf("Run(%q) = %d, out %q err %q; want 2 with usage on stderr", args, code, out.String(), errOut.String())
		}
	}
}

func TestRunCLIFailureExitsNonZeroButPrintsSprite(t *testing.T) {
	h := newFakeHost(t)
	h.missing["/species/25"] = true
	var out, errOut bytes.Buffer
	if code := Run([]string{"25"}, h.config(t), &out, &errOut); code != 1 {
		t.Errorf("code %d, want 1", code)
	}
	if strings.TrimSpace(out.String()) != "poke-assets/25.gif" || !strings.Contains(errOut.String(), "name") {
		t.Errorf("out %q err %q", out.String(), errOut.String())
	}
}

func TestDexKeepsShinyArtSeparate(t *testing.T) {
	c := Config{Home: t.TempDir(), PTBDir: t.TempDir()}
	os.MkdirAll(filepath.Join(c.PTBDir, "sprites"), 0755)
	os.WriteFile(filepath.Join(c.PTBDir, "sprites", "6-s.png"), []byte("ordinary"), 0600)
	os.WriteFile(filepath.Join(c.PTBDir, "sprites", "6-shs.png"), []byte("shiny"), 0600)
	os.WriteFile(filepath.Join(c.PTBDir, "companion-state.json"), []byte(`{"dex":[{"finalID":6},{"finalID":6,"isShiny":true}]}`), 0600)
	if err := Dex(c); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"dex-6.png": "ordinary", "dex-6-shiny.png": "shiny"} {
		b, err := os.ReadFile(filepath.Join(c.Dir(), name))
		if err != nil || string(b) != want {
			t.Fatalf("%s: %q %v", name, b, err)
		}
	}
}
