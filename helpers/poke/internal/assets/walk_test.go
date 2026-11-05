package assets

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// rowColor tags each facing row of a fake sheet so a test can tell which row a GIF came from.
func rowColor(row int) color.NRGBA { return color.NRGBA{uint8(20 + row*25), 100, 200, 255} }

// sheet draws a PMD-style sheet: rows of frames, each with a small opaque body
// centred in its frame (PMD anchors frames at their centre).
func sheet(fw, fh, cols, rows int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, fw*cols, fh*rows))
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cx, cy := c*fw+fw/2, r*fh+fh/2
			for y := cy - 4; y < cy+4; y++ {
				for x := cx - 3; x < cx+3; x++ {
					img.Set(x, y, rowColor(r))
				}
			}
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func animXML(anims ...string) string {
	return "<AnimData><Anims>" + strings.Join(anims, "") + "</Anims></AnimData>"
}

func anim(name string, fw, fh int, durations ...int) string {
	var d strings.Builder
	for _, n := range durations {
		fmt.Fprintf(&d, "<Duration>%d</Duration>", n)
	}
	return fmt.Sprintf("<Anim><Name>%s</Name><FrameWidth>%d</FrameWidth><FrameHeight>%d</FrameHeight><Durations>%s</Durations></Anim>", name, fw, fh, d.String())
}

type pmdHost struct {
	srv   *httptest.Server
	mu    sync.Mutex
	got   []string
	files map[string][]byte
	fail  bool
}

func newPMDHost(t *testing.T, files map[string][]byte) *pmdHost {
	h := &pmdHost{files: files}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.got = append(h.got, r.URL.Path)
		fail := h.fail
		h.mu.Unlock()
		if fail {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		b, ok := h.files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(b)
	}))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *pmdHost) config(t *testing.T) Config {
	return Config{Home: t.TempDir(), PMDBase: h.srv.URL}
}

func (h *pmdHost) requests() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.got...)
}

func pikachuHost(t *testing.T) *pmdHost {
	return newPMDHost(t, map[string][]byte{
		"/0025/AnimData.xml":   []byte(animXML(anim("Walk", 32, 40, 8, 10, 8, 10), anim("Idle", 40, 56, 40, 2, 3), anim("Sleep", 32, 40, 30, 35))),
		"/0025/Walk-Anim.png":  sheet(32, 40, 4, 8),
		"/0025/Idle-Anim.png":  sheet(40, 56, 3, 8),
		"/0025/Sleep-Anim.png": sheet(32, 40, 2, 1),
	})
}

func readGIF(t *testing.T, path string) *gif.GIF {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return g
}

// opaque returns the colour of the first opaque pixel of frame i.
func opaque(g *gif.GIF, i int) color.NRGBA {
	f := g.Image[i]
	for y := f.Bounds().Min.Y; y < f.Bounds().Max.Y; y++ {
		for x := f.Bounds().Min.X; x < f.Bounds().Max.X; x++ {
			if c := color.NRGBAModel.Convert(f.At(x, y)).(color.NRGBA); c.A == 255 {
				return c
			}
		}
	}
	return color.NRGBA{}
}

func TestWalkCutsFourSquareGIFsFacingTheRightWay(t *testing.T) {
	h := pikachuHost(t)
	c := h.config(t)
	if err := EnsureWalk(c, 25); err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		frames int
		row    int
	}{"walk-25-r.gif": {4, 2}, "walk-25-l.gif": {4, 6}, "walk-25-idle.gif": {3, 0}, "walk-25-sleep.gif": {2, 0}}
	size := 0
	for name, w := range want {
		g := readGIF(t, filepath.Join(c.Dir(), name))
		if len(g.Image) != w.frames {
			t.Errorf("%s: %d frames, want %d", name, len(g.Image), w.frames)
		}
		if g.Config.Width != g.Config.Height {
			t.Errorf("%s: %dx%d is not square", name, g.Config.Width, g.Config.Height)
		}
		if size == 0 {
			size = g.Config.Width
		} else if g.Config.Width != size {
			t.Errorf("%s: size %d, others %d; one species shares one box", name, g.Config.Width, size)
		}
		if got := opaque(g, 0); got != rowColor(w.row) {
			t.Errorf("%s: colour %v, want row %d", name, got, w.row)
		}
		if g.LoopCount != 0 {
			t.Errorf("%s: loop count %d, want forever", name, g.LoopCount)
		}
	}
	// PMD durations are 1/60 s; GIF delays are 1/100 s.
	if d := readGIF(t, filepath.Join(c.Dir(), "walk-25-r.gif")).Delay; d[0] != 13 || d[1] != 17 {
		t.Errorf("walk delays %v, want 13, 17, …", d)
	}
	if d := readGIF(t, filepath.Join(c.Dir(), "walk-25-idle.gif")).Delay; d[1] != 3 {
		t.Errorf("idle delays %v: very short frames stay visible", d)
	}
}

func TestWalkFeetShareOneGroundLine(t *testing.T) {
	h := pikachuHost(t)
	c := h.config(t)
	if err := EnsureWalk(c, 25); err != nil {
		t.Fatal(err)
	}
	bottom := func(name string) int {
		g := readGIF(t, filepath.Join(c.Dir(), name))
		f := g.Image[0]
		for y := f.Bounds().Max.Y - 1; y >= 0; y-- {
			for x := 0; x < f.Bounds().Max.X; x++ {
				if _, _, _, a := f.At(x, y).RGBA(); a != 0 {
					return y
				}
			}
		}
		return -1
	}
	// Frames of different sizes are anchored at their centres, so the body sits at the same line.
	if r, i := bottom("walk-25-r.gif"), bottom("walk-25-idle.gif"); r != i || r < 0 {
		t.Errorf("walk bottom %d, idle bottom %d", r, i)
	}
}

func TestWalkFollowsCopyOf(t *testing.T) {
	h := newPMDHost(t, map[string][]byte{
		"/0007/AnimData.xml":  []byte(animXML(anim("Walk", 24, 32, 6, 6), "<Anim><Name>Idle</Name><CopyOf>Walk</CopyOf></Anim>")),
		"/0007/Walk-Anim.png": sheet(24, 32, 2, 8),
	})
	c := h.config(t)
	if err := EnsureWalk(c, 7); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"walk-7-idle.gif", "walk-7-sleep.gif"} {
		if g := readGIF(t, filepath.Join(c.Dir(), name)); len(g.Image) != 2 {
			t.Errorf("%s: %d frames, want the walk's 2", name, len(g.Image))
		}
	}
	for _, r := range h.requests() {
		if strings.Contains(r, "Idle") || strings.Contains(r, "Sleep") {
			t.Errorf("fetched %s; Idle copies Walk and Sleep falls back to Idle", r)
		}
	}
}

func TestWalkMissingSpeciesIsRememberedOnce(t *testing.T) {
	h := newPMDHost(t, map[string][]byte{})
	c := h.config(t)
	if err := EnsureWalk(c, 999); err != nil {
		t.Fatalf("a species PMD lacks is not an error: %v", err)
	}
	if !has(c.Dir(), "walk-999.none") {
		t.Fatal("no walk-999.none marker")
	}
	n := len(h.requests())
	if err := EnsureWalk(c, 999); err != nil || len(h.requests()) != n {
		t.Errorf("asked again after the marker: %v, %v", err, h.requests())
	}
}

func TestWalkNetworkTroubleIsRetriedLater(t *testing.T) {
	h := pikachuHost(t)
	h.fail = true
	c := h.config(t)
	if err := EnsureWalk(c, 25); err == nil {
		t.Fatal("no error for a 503")
	}
	if has(c.Dir(), "walk-25.none") || has(c.Dir(), "walk-25-r.gif") {
		t.Error("a passing failure left a marker or a partial set")
	}
	h.fail = false
	if err := EnsureWalk(c, 25); err != nil || !has(c.Dir(), "walk-25-r.gif") {
		t.Errorf("retry: %v", err)
	}
}

func TestWalkDoneSetIsNotFetchedAgain(t *testing.T) {
	h := pikachuHost(t)
	c := h.config(t)
	if err := EnsureWalk(c, 25); err != nil {
		t.Fatal(err)
	}
	n := len(h.requests())
	if err := EnsureWalk(c, 25); err != nil || len(h.requests()) != n {
		t.Errorf("fetched again: %v", h.requests()[n:])
	}
}

func TestRunWalkCLI(t *testing.T) {
	h := pikachuHost(t)
	var out, errOut bytes.Buffer
	c := h.config(t)
	if code := Run([]string{"--walk", "25"}, c, &out, &errOut); code != 0 {
		t.Fatalf("code %d: %s", code, errOut.String())
	}
	if code := Run([]string{"--walk", "x"}, c, &out, &errOut); code != 2 {
		t.Errorf("bad id: code %d", code)
	}
}
