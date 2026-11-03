package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
)

// normalizeGIF rewrites an animated sprite so every frame is the whole square
// canvas. Showdown/PokeAPI sprites store each frame as a smaller rectangle and
// many have non-square canvases; Herdr stretches each frame to the image box,
// so such sprites wobble, squash and spill out of their circle. The art is
// centred horizontally and sits on the bottom edge. Already-normal input comes
// back unchanged (changed=false); undecodable input is an error.
func normalizeGIF(src []byte) (out []byte, changed bool, err error) {
	g, err := gif.DecodeAll(bytes.NewReader(src))
	if err != nil {
		return nil, false, err
	}
	w, h := g.Config.Width, g.Config.Height
	if isNormal(g) {
		return src, false, nil
	}
	s := max(w, h)
	off := image.Pt((s-w)/2, s-h)

	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	res := &gif.GIF{Config: image.Config{Width: s, Height: s}, LoopCount: g.LoopCount, BackgroundIndex: 0}
	for i, f := range g.Image {
		disp := byte(gif.DisposalNone)
		if i < len(g.Disposal) {
			disp = g.Disposal[i]
		}
		var saved *image.RGBA
		if disp == gif.DisposalPrevious {
			saved = image.NewRGBA(canvas.Bounds())
			copy(saved.Pix, canvas.Pix)
		}
		draw.Draw(canvas, f.Bounds(), f, f.Bounds().Min, draw.Over)

		full := image.NewRGBA(image.Rect(0, 0, s, s))
		draw.Draw(full, canvas.Bounds().Add(off), canvas, image.Point{}, draw.Src)
		res.Image = append(res.Image, toPaletted(full, f.Palette))
		res.Disposal = append(res.Disposal, gif.DisposalBackground)
		delay := 0
		if i < len(g.Delay) {
			delay = g.Delay[i]
		}
		res.Delay = append(res.Delay, delay)

		switch disp {
		case gif.DisposalBackground:
			draw.Draw(canvas, f.Bounds(), image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvas = saved
		}
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, res); err != nil {
		return nil, false, err
	}
	return b.Bytes(), true, nil
}

func isNormal(g *gif.GIF) bool {
	w, h := g.Config.Width, g.Config.Height
	if w != h {
		return false
	}
	for i, f := range g.Image {
		if f.Bounds() != image.Rect(0, 0, w, h) || i >= len(g.Disposal) || g.Disposal[i] != gif.DisposalBackground {
			return false
		}
	}
	return true
}

// toPaletted keeps every colour exactly when the frame has at most 255 of
// them (sprites always do); otherwise it falls back to the source palette.
func toPaletted(img *image.RGBA, fallback color.Palette) *image.Paletted {
	p := color.Palette{color.RGBA{}}
	seen := map[color.RGBA]bool{}
	for i := 0; i < len(img.Pix); i += 4 {
		c := color.RGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
		if c.A == 0 || seen[c] {
			continue
		}
		seen[c] = true
		p = append(p, c)
	}
	if len(p) > 256 {
		p = append(color.Palette{color.RGBA{}}, fallback...)
		if len(p) > 256 {
			p = p[:256]
		}
	}
	out := image.NewPaletted(img.Bounds(), p)
	draw.Draw(out, img.Bounds(), img, image.Point{}, draw.Src)
	return out
}

// NormalizeDir normalizes every GIF in the asset folder in place and returns
// how many it rewrote. A GIF it cannot read is left as it is.
func NormalizeDir(c Config) (int, error) {
	entries, err := os.ReadDir(c.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".gif") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(c.Dir(), e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		out, changed, err := normalizeGIF(src)
		if err != nil || !changed {
			continue
		}
		if err := writeFileAtomic(path, out); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// normalizeFile normalizes one GIF in place; a file it cannot read stays as is.
func normalizeFile(path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if out, changed, err := normalizeGIF(src); err == nil && changed {
		_ = writeFileAtomic(path, out)
	}
}
