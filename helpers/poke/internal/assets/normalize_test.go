package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

var pal = color.Palette{color.Transparent, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}

func frame(x, y, w, h int, idx uint8) *image.Paletted {
	p := image.NewPaletted(image.Rect(x, y, x+w, y+h), pal)
	for i := range p.Pix {
		p.Pix[i] = idx
	}
	return p
}

// A tall 4x6 sprite whose two frames cover different rectangles.
func tallGIF(t *testing.T) []byte {
	t.Helper()
	g := &gif.GIF{
		Image:    []*image.Paletted{frame(0, 0, 4, 6, 1), frame(0, 0, 2, 3, 2)},
		Delay:    []int{5, 7},
		Disposal: []byte{gif.DisposalNone, gif.DisposalNone},
		Config:   image.Config{Width: 4, Height: 6, ColorModel: pal},
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestNormalizeSquaresEveryFrame(t *testing.T) {
	out, changed, err := normalizeGIF(tallGIF(t))
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if g.Config.Width != 6 || g.Config.Height != 6 {
		t.Fatalf("canvas %dx%d, want 6x6", g.Config.Width, g.Config.Height)
	}
	if len(g.Image) != 2 || g.Delay[1] != 7 {
		t.Fatalf("frames %d delays %v", len(g.Image), g.Delay)
	}
	for i, f := range g.Image {
		if f.Bounds() != image.Rect(0, 0, 6, 6) {
			t.Fatalf("frame %d bounds %v, want full canvas", i, f.Bounds())
		}
	}
	// Centred horizontally: one empty column left, one right.
	if _, _, _, a := g.Image[0].At(0, 3).RGBA(); a != 0 {
		t.Fatal("left padding not transparent")
	}
	if r, _, _, _ := g.Image[0].At(1, 3).RGBA(); r == 0 {
		t.Fatal("sprite not at x=1")
	}
	// Frame 2 drew over frame 1 (DisposalNone): blue top-left, red kept below it.
	if _, _, b, _ := g.Image[1].At(1, 0).RGBA(); b == 0 {
		t.Fatal("frame 2 pixel missing")
	}
	if r, _, _, _ := g.Image[1].At(1, 5).RGBA(); r == 0 {
		t.Fatal("frame 1 pixel not kept under frame 2")
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	once, _, _ := normalizeGIF(tallGIF(t))
	_, changed, err := normalizeGIF(once)
	if err != nil || changed {
		t.Fatalf("second pass changed=%v err=%v", changed, err)
	}
}

func TestNormalizeLeavesUndecodableAlone(t *testing.T) {
	_, changed, err := normalizeGIF([]byte(gifBytes))
	if changed || err == nil {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
}

func TestNormalizeDirRewritesOnlyGIFs(t *testing.T) {
	c := Config{Home: t.TempDir()}
	os.MkdirAll(c.Dir(), 0o755)
	os.WriteFile(filepath.Join(c.Dir(), "152.gif"), tallGIF(t), 0o644)
	os.WriteFile(filepath.Join(c.Dir(), "700.png"), []byte(pngBytes), 0o644)
	n, err := NormalizeDir(c)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if n, _ := NormalizeDir(c); n != 0 {
		t.Fatalf("second run rewrote %d", n)
	}
}
