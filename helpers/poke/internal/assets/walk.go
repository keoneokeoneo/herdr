package assets

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Walking art for the Poké Garden, cut from PMDCollab's SpriteCollab sheets.
//
// A species gets four looping GIFs in the asset folder: walk-<id>-r.gif and
// walk-<id>-l.gif (walking right / left), walk-<id>-idle.gif (standing, facing
// the viewer) and walk-<id>-sleep.gif. All four share one square box with the
// feet on one ground line, so the UI can swap them without the sprite jumping.
// walk-<id>-r.gif is written last and marks the set complete. A species PMD
// has no art for gets walk-<id>.none and is not asked for again.

const (
	defaultPMDBase = "https://raw.githubusercontent.com/PMDCollab/SpriteCollab/master/sprite"
	walkTarget     = 96 // px the box is scaled up towards, by a whole factor
)

// PMD sheets have one row per facing: 0 down, 1 down-right, 2 right, 3 up-right,
// 4 up, 5 up-left, 6 left, 7 down-left.
var walkCuts = []struct {
	file string
	anim string
	row  int
}{
	{"idle", "Idle", 0},
	{"sleep", "Sleep", 0},
	{"l", "Walk", 6},
	{"r", "Walk", 2}, // last: the completion mark
}

type pmdAnim struct {
	name      string
	copyOf    string
	fw, fh    int
	durations []int
}

var (
	reAnim     = regexp.MustCompile(`(?s)<Anim>(.*?)</Anim>`)
	reDuration = regexp.MustCompile(`<Duration>(\d+)</Duration>`)
)

func tag(body, name string) string {
	m := regexp.MustCompile(`<` + name + `>([^<]*)</` + name + `>`).FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func parseAnimData(xml []byte) map[string]pmdAnim {
	anims := map[string]pmdAnim{}
	for _, m := range reAnim.FindAllSubmatch(xml, -1) {
		body := string(m[1])
		a := pmdAnim{name: tag(body, "Name"), copyOf: tag(body, "CopyOf")}
		a.fw, _ = strconv.Atoi(tag(body, "FrameWidth"))
		a.fh, _ = strconv.Atoi(tag(body, "FrameHeight"))
		for _, d := range reDuration.FindAllStringSubmatch(body, -1) {
			n, _ := strconv.Atoi(d[1])
			a.durations = append(a.durations, n)
		}
		if a.name != "" {
			anims[a.name] = a
		}
	}
	return anims
}

// resolve follows CopyOf ("Idle is the same as Walk") to the animation that owns a sheet.
func resolve(anims map[string]pmdAnim, name string) (pmdAnim, bool) {
	a, ok := anims[name]
	for hops := 0; ok && a.copyOf != "" && hops < 4; hops++ {
		a, ok = anims[a.copyOf]
	}
	return a, ok && a.fw > 0 && a.fh > 0 && len(a.durations) > 0
}

// EnsureWalk makes sure the walking set (or the "none" marker) for id exists.
func EnsureWalk(c Config, id int) error {
	if id < 1 || id > maxID {
		return fmt.Errorf("pokemon id %d out of range 1-%d", id, maxID)
	}
	n := strconv.Itoa(id)
	if has(c.Dir(), "walk-"+n+"-r.gif") || has(c.Dir(), "walk-"+n+".none") {
		return nil
	}
	if err := os.MkdirAll(c.Dir(), 0o755); err != nil {
		return err
	}
	none := func() error { return writeFileAtomic(filepath.Join(c.Dir(), "walk-"+n+".none"), []byte("none\n")) }
	base := fmt.Sprintf("%s/%04d", c.pmdBase(), id)
	xml, err := get(c, base+"/AnimData.xml")
	if errors.Is(err, errNotFound) {
		return none()
	}
	if err != nil {
		return fmt.Errorf("walk #%d: %w", id, err)
	}
	anims := parseAnimData(xml)
	pick := map[string]pmdAnim{}
	walk, ok := resolve(anims, "Walk")
	if !ok {
		return none()
	}
	pick["Walk"] = walk
	if pick["Idle"], ok = resolve(anims, "Idle"); !ok {
		pick["Idle"] = walk
	}
	if pick["Sleep"], ok = resolve(anims, "Sleep"); !ok {
		pick["Sleep"] = pick["Idle"]
	}
	sheets := map[string]image.Image{}
	for _, a := range pick {
		if sheets[a.name] != nil {
			continue
		}
		b, err := get(c, base+"/"+a.name+"-Anim.png")
		if errors.Is(err, errNotFound) {
			return none()
		}
		if err != nil {
			return fmt.Errorf("walk #%d: %w", id, err)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("walk #%d: %s sheet: %w", id, a.name, err)
		}
		sheets[a.name] = img
	}

	// Every frame of every cut, in centre-anchored coordinates.
	type frame struct {
		img    image.Image
		origin image.Point // sheet position of the frame's top-left
		a      pmdAnim
	}
	cuts := make([][]frame, len(walkCuts))
	box := image.Rectangle{}
	for i, cut := range walkCuts {
		a := pick[cut.anim]
		sh := sheets[a.name]
		cols, rows := sh.Bounds().Dx()/a.fw, sh.Bounds().Dy()/a.fh
		if cols < 1 || rows < 1 {
			return none()
		}
		row := min(cut.row, rows-1)
		for col := 0; col < min(cols, len(a.durations)); col++ {
			f := frame{sh, image.Pt(col*a.fw, row*a.fh), a}
			cuts[i] = append(cuts[i], f)
			if r := opaqueBounds(sh, image.Rect(f.origin.X, f.origin.Y, f.origin.X+a.fw, f.origin.Y+a.fh)); !r.Empty() {
				box = box.Union(r.Sub(f.origin).Sub(image.Pt(a.fw/2, a.fh/2)))
			}
		}
	}
	if box.Empty() {
		return none()
	}
	side := max(box.Dx(), box.Dy())
	scale := max(1, walkTarget/side)
	// Centred horizontally, feet on the bottom edge.
	shift := image.Pt((side-box.Dx())/2-box.Min.X, side-box.Dy()-box.Min.Y)

	for i, cut := range walkCuts {
		g := &gif.GIF{Config: image.Config{Width: side * scale, Height: side * scale}, LoopCount: 0}
		for j, f := range cuts[i] {
			canvas := image.NewNRGBA(image.Rect(0, 0, side, side))
			centre := image.Pt(f.a.fw/2, f.a.fh/2)
			for y := 0; y < f.a.fh; y++ {
				for x := 0; x < f.a.fw; x++ {
					c := color.NRGBAModel.Convert(f.img.At(f.origin.X+x, f.origin.Y+y)).(color.NRGBA)
					if c.A == 0 {
						continue
					}
					p := image.Pt(x, y).Sub(centre).Add(shift)
					canvas.SetNRGBA(p.X, p.Y, c)
				}
			}
			g.Image = append(g.Image, paletted(canvas, scale))
			g.Delay = append(g.Delay, max(2, int(math.Round(float64(f.a.durations[j])*100/60))))
			g.Disposal = append(g.Disposal, gif.DisposalBackground)
		}
		var out bytes.Buffer
		if err := gif.EncodeAll(&out, g); err != nil {
			return fmt.Errorf("walk #%d: %w", id, err)
		}
		if err := writeFileAtomic(filepath.Join(c.Dir(), "walk-"+n+"-"+cut.file+".gif"), out.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func opaqueBounds(img image.Image, r image.Rectangle) image.Rectangle {
	out := image.Rectangle{}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				out = out.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return out
}

// paletted scales src up by a whole factor (nearest neighbour) into a frame
// whose index 0 is transparent. Pixel art has few colours; past 255 the rest
// map to their nearest.
func paletted(src *image.NRGBA, scale int) *image.Paletted {
	pal := color.Palette{color.NRGBA{}}
	seen := map[color.NRGBA]bool{}
	for i := 0; i < len(src.Pix); i += 4 {
		c := color.NRGBA{src.Pix[i], src.Pix[i+1], src.Pix[i+2], 255}
		if src.Pix[i+3] == 0 || seen[c] || len(pal) == 256 {
			continue
		}
		seen[c] = true
		pal = append(pal, c)
	}
	b := src.Bounds()
	dst := image.NewPaletted(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale), pal)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.NRGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			idx := uint8(pal.Index(color.NRGBA{c.R, c.G, c.B, 255}))
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					dst.SetColorIndex(x*scale+dx, y*scale+dy, idx)
				}
			}
		}
	}
	return dst
}

func (c Config) pmdBase() string {
	if c.PMDBase != "" {
		return c.PMDBase
	}
	return defaultPMDBase
}
