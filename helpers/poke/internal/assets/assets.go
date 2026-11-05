// Package assets keeps the Pokemon files the poke UI shows in
// $HERDR_APP_HOME/poke-assets: sprites, a name table, and a copy of the
// active pokeTokenBar companion's sprite.
//
// The folder is deliberately outside mods/poke: Herdr reloads a mod on any file
// write inside its folder, so writing assets there would reload the UI in a loop.
// For the same reason nothing here rewrites a file whose content is unchanged.
package assets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	dirName = "poke-assets"

	defaultSpriteBase  = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon"
	defaultSpeciesBase = "https://pokeapi.co/api/v2/pokemon-species"

	maxAnimatedID = 649   // Black/White animated sprites stop at Genesect
	maxID         = 99999 // sanity bound for the command line
	maxBody       = 8 << 20
)

// Config says where files live and where they come from. ConfigFromEnv gives
// the real values; tests point the bases at an httptest server.
type Config struct {
	Home        string // Herdr app home; assets go in Home/poke-assets
	PTBDir      string // pokeTokenBar's Application Support folder (read only)
	SpriteBase  string // <base>/versions/generation-v/black-white/animated/<id>.gif and <base>/<id>.png
	SpeciesBase string // <base>/<id>
	PMDBase     string // <base>/<0025>/AnimData.xml and <Anim>-Anim.png (walking art)
	Client      *http.Client
}

// ConfigFromEnv reads HERDR_APP_HOME (default ~/.config/herdr-app) and, for
// tests, HERDR_POKE_SPRITE_BASE, HERDR_POKE_SPECIES_BASE, HERDR_POKE_PMD_BASE
// and HERDR_POKE_PTB_DIR.
func ConfigFromEnv() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	env := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}
	return Config{
		Home:        env("HERDR_APP_HOME", filepath.Join(home, ".config", "herdr-app")),
		PTBDir:      env("HERDR_POKE_PTB_DIR", filepath.Join(home, "Library", "Application Support", "PokeTokenBar")),
		SpriteBase:  env("HERDR_POKE_SPRITE_BASE", defaultSpriteBase),
		SpeciesBase: env("HERDR_POKE_SPECIES_BASE", defaultSpeciesBase),
		PMDBase:     env("HERDR_POKE_PMD_BASE", defaultPMDBase),
	}
}

// Dir is the asset folder.
func (c Config) Dir() string { return filepath.Join(c.Home, dirName) }

func (c Config) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// rel is path relative to the app home, with forward slashes (absolute if it
// lies outside the home).
func (c Config) rel(path string) string {
	if r, err := filepath.Rel(c.Home, path); err == nil && !strings.HasPrefix(r, "..") {
		return filepath.ToSlash(r)
	}
	return path
}

// Ensure makes sure the sprite and name of Pokemon id are on disk, and
// returns the sprite path relative to the app home. Files already there are
// neither fetched nor rewritten. If only the name fails, the sprite
// path is still returned, together with the error.
func Ensure(c Config, id int) (string, error) {
	if id < 1 || id > maxID {
		return "", fmt.Errorf("pokemon id %d out of range 1-%d", id, maxID)
	}
	if err := os.MkdirAll(c.Dir(), 0o755); err != nil {
		return "", err
	}
	var errs []error
	sprite, err := ensureSprite(c, id)
	if err != nil {
		errs = append(errs, err)
	}
	if err := ensureName(c, id); err != nil {
		errs = append(errs, err)
	}
	if sprite != "" {
		sprite = c.rel(filepath.Join(c.Dir(), sprite))
	}
	return sprite, errors.Join(errs...)
}

func ensureSprite(c Config, id int) (string, error) {
	gif, png := strconv.Itoa(id)+".gif", strconv.Itoa(id)+".png"
	if id <= maxAnimatedID {
		if has(c.Dir(), gif) {
			return gif, nil
		}
		if has(c.Dir(), png) { // an earlier run found no animated sprite
			return png, nil
		}
		url := fmt.Sprintf("%s/versions/generation-v/black-white/animated/%d.gif", c.SpriteBase, id)
		err := download(c, url, filepath.Join(c.Dir(), gif), isGIF)
		if err == nil {
			normalizeFile(filepath.Join(c.Dir(), gif))
			return gif, nil
		}
		if !errors.Is(err, errNotFound) {
			return "", fmt.Errorf("sprite #%d: %w", id, err)
		}
	} else if has(c.Dir(), png) {
		return png, nil
	}
	url := fmt.Sprintf("%s/%d.png", c.SpriteBase, id)
	if err := download(c, url, filepath.Join(c.Dir(), png), isPNG); err != nil {
		return "", fmt.Errorf("sprite #%d: %w", id, err)
	}
	return png, nil
}

type species struct {
	Name  string `json:"name"`
	Names []struct {
		Name     string `json:"name"`
		Language struct {
			Name string `json:"name"`
		} `json:"language"`
	} `json:"names"`
}

// english is the English display name, or the API name made presentable.
func (s species) english() string {
	for _, n := range s.Names {
		if n.Language.Name == "en" && n.Name != "" {
			return n.Name
		}
	}
	words := strings.Split(s.Name, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func ensureName(c Config, id int) error {
	key := strconv.Itoa(id)
	if readNames(c)[key] != "" {
		return nil
	}
	sp, err := fetchSpecies(c, id)
	if err != nil {
		return fmt.Errorf("species lookup #%d (for the name): %w", id, err)
	}
	if err := setName(c, key, sp.english()); err != nil {
		return fmt.Errorf("name #%d: %w", id, err)
	}
	return nil
}

func fetchSpecies(c Config, id int) (species, error) {
	body, err := get(c, fmt.Sprintf("%s/%d", c.SpeciesBase, id))
	if err != nil {
		return species{}, err
	}
	var sp species
	if err := json.Unmarshal(body, &sp); err != nil {
		return species{}, fmt.Errorf("bad species JSON: %w", err)
	}
	if sp.Name == "" {
		return species{}, errors.New("species JSON has no name")
	}
	return sp, nil
}

// ---- names.json ----

func readNames(c Config) map[string]string {
	names := map[string]string{}
	if b, err := os.ReadFile(filepath.Join(c.Dir(), "names.json")); err == nil {
		if json.Unmarshal(b, &names) != nil || names == nil {
			names = map[string]string{}
		}
	}
	return names
}

// setName records names.json[id] = name, leaving the other entries alone.
func setName(c Config, id, name string) error {
	return withLock(c.Dir(), func() error {
		names := readNames(c) // fresh, under the lock
		if names[id] == name {
			return nil
		}
		names[id] = name
		b, err := json.MarshalIndent(names, "", "  ")
		if err != nil {
			return err
		}
		return writeFileAtomic(filepath.Join(c.Dir(), "names.json"), append(b, '\n'))
	})
}

// withLock serialises writers of names.json across processes.
func withLock(dir string, fn func() error) error {
	f, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

// ---- companion ----

type companionState struct {
	Active *struct {
		BaseID  int   `json:"baseID"`
		PathIDs []int `json:"pathIDs"`
		IsShiny bool  `json:"isShiny"`
	} `json:"active"`
}

// Companion copies the active pokeTokenBar companion's sprite to
// poke-assets/companion-<id>.gif|png and returns that path relative to the app
// home. The active id is the last of active.pathIDs (else active.baseID).
// pokeTokenBar's files are only read. An unchanged copy is not rewritten.
func Companion(c Config) (string, error) {
	b, err := os.ReadFile(filepath.Join(c.PTBDir, "companion-state.json"))
	if err != nil {
		return "", fmt.Errorf("companion: %w", err)
	}
	var st companionState
	if err := json.Unmarshal(b, &st); err != nil {
		return "", fmt.Errorf("companion: companion-state.json: %w", err)
	}
	if st.Active == nil {
		return "", errors.New("companion: no active companion")
	}
	id := st.Active.BaseID
	if n := len(st.Active.PathIDs); n > 0 {
		id = st.Active.PathIDs[n-1]
	}
	if id < 1 {
		return "", errors.New("companion: active companion has no id")
	}

	// Animated first, and a shiny prefers its own art at each step.
	n := strconv.Itoa(id)
	prefs := []string{n + "-a.gif", n + "-s.png"}
	if st.Active.IsShiny {
		prefs = []string{n + "-sha.gif", n + "-a.gif", n + "-shs.png", n + "-s.png"}
	}
	sprites := filepath.Join(c.PTBDir, "sprites")
	for _, name := range prefs {
		src, err := os.ReadFile(filepath.Join(sprites, name))
		if err != nil || len(src) == 0 {
			continue
		}
		if err := os.MkdirAll(c.Dir(), 0o755); err != nil {
			return "", err
		}
		if out, changed, err := normalizeGIF(src); err == nil && changed {
			src = out
		}
		dst := filepath.Join(c.Dir(), "companion-"+n+filepath.Ext(name))
		if _, err := writeIfChanged(dst, src); err != nil {
			return "", fmt.Errorf("companion: %w", err)
		}
		return c.rel(dst), nil
	}
	return "", fmt.Errorf("companion: no sprite for #%d in %s", id, sprites)
}

// ---- list ----

// List returns the names of the files in the asset folder, sorted. Hidden
// files (locks, temp files) and folders are left out. A missing folder is an
// empty list.
func List(c Config) ([]string, error) {
	names := []string{}
	entries, err := os.ReadDir(c.Dir())
	if errors.Is(err, os.ErrNotExist) {
		return names, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Type().IsRegular() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// ---- command line ----

const usageText = "usage: herdr-poke assets <pokemon id> | --walk <pokemon id> | --companion | --list | --normalize | --dex\n"

// Run is `herdr-poke assets ...`.
//
//	assets <id>        fetch sprite and name; print the sprite path
//	assets --walk <id> cut the Garden's walking art (or mark the species as having none)
//	assets --companion copy the active companion's sprite; print its path
//	assets --list      print the asset file names as a JSON array
//	assets --normalize square every GIF in the folder; print how many changed
func Run(args []string, c Config, stdout, stderr io.Writer) int {
	if len(args) == 2 && args[0] == "--walk" {
		id, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprint(stderr, usageText)
			return 2
		}
		if err := EnsureWalk(c, id); err != nil {
			fmt.Fprintf(stderr, "herdr-poke assets: %v\n", err)
			return 1
		}
		return 0
	}
	if len(args) != 1 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	switch arg := args[0]; {
	case arg == "--dex":
		if err := Dex(c); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case arg == "--list":
		names, err := List(c)
		if err != nil {
			fmt.Fprintf(stderr, "herdr-poke assets: %v\n", err)
			return 1
		}
		b, _ := json.Marshal(names)
		fmt.Fprintf(stdout, "%s\n", b)
		return 0
	case arg == "--normalize":
		n, err := NormalizeDir(c)
		if err != nil {
			fmt.Fprintf(stderr, "herdr-poke assets: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%d\n", n)
		return 0
	case arg == "--companion":
		return report(stdout, stderr)(Companion(c))
	case strings.HasPrefix(arg, "-"):
		fmt.Fprint(stderr, usageText)
		return 2
	default:
		id, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprint(stderr, usageText)
			return 2
		}
		return report(stdout, stderr)(Ensure(c, id))
	}
}

// report prints a path (if there is one) and an error (if there is one).
func report(stdout, stderr io.Writer) func(string, error) int {
	return func(path string, err error) int {
		if path != "" {
			fmt.Fprintln(stdout, path)
		}
		if err != nil {
			fmt.Fprintf(stderr, "herdr-poke assets: %v\n", err)
			return 1
		}
		return 0
	}
}

// ---- files and HTTP ----

func has(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

// writeFileAtomic writes data to a temp file beside path and renames it over,
// so a reader never sees half a file.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr, os.Chmod(name, 0o644)); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// writeIfChanged leaves path alone (not even its mtime) if it already holds data.
func writeIfChanged(path string, data []byte) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
		return false, nil
	}
	return true, writeFileAtomic(path, data)
}

var errNotFound = errors.New("not found (404)")

func get(c Config, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "herdr-poke")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s: %w", url, errNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	if len(body) > maxBody {
		return nil, fmt.Errorf("%s: larger than %d bytes", url, maxBody)
	}
	return body, nil
}

// download fetches url to dest if the body passes ok. Nothing is saved otherwise.
func download(c Config, url, dest string, ok func([]byte) bool) error {
	body, err := get(c, url)
	if err != nil {
		return err
	}
	if len(body) == 0 || !ok(body) {
		return fmt.Errorf("%s: not the expected kind of file", url)
	}
	return writeFileAtomic(dest, body)
}

func isGIF(b []byte) bool { return bytes.HasPrefix(b, []byte("GIF8")) }

func isPNG(b []byte) bool { return bytes.HasPrefix(b, []byte("\x89PNG")) }

// Dex copies each caught variant once, normalizing only newly copied GIFs.
// Missing source art is retried on the next sync; existing variants are immutable.
func Dex(c Config) error {
	b, err := os.ReadFile(filepath.Join(c.PTBDir, "companion-state.json"))
	if err != nil {
		return err
	}
	var st struct {
		Dex []struct {
			FinalID int   `json:"finalID"`
			BaseID  int   `json:"baseID"`
			Chain   []int `json:"chainOrder"`
			Shiny   bool  `json:"isShiny"`
		} `json:"dex"`
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	if err := os.MkdirAll(c.Dir(), 0755); err != nil {
		return err
	}
	for _, d := range st.Dex {
		id := d.FinalID
		if id == 0 && len(d.Chain) > 0 {
			id = d.Chain[len(d.Chain)-1]
		}
		if id == 0 {
			id = d.BaseID
		}
		if id <= 0 {
			continue
		}
		key := "dex-" + strconv.Itoa(id)
		prefs := []string{"a.gif", "s.png"}
		if d.Shiny {
			key += "-shiny"
			prefs = []string{"sha.gif", "shs.png"}
		}
		if has(c.Dir(), key+".gif") || has(c.Dir(), key+".png") {
			continue
		}
		for _, suffix := range prefs {
			src, err := os.ReadFile(filepath.Join(c.PTBDir, "sprites", strconv.Itoa(id)+"-"+suffix))
			if err != nil {
				continue
			}
			if strings.HasSuffix(suffix, ".gif") {
				out, _, err := normalizeGIF(src)
				if err != nil {
					continue
				}
				src = out
			}
			if err := writeFileAtomic(filepath.Join(c.Dir(), key+filepath.Ext(suffix)), src); err != nil {
				return err
			}
			break
		}
	}
	return nil
}
