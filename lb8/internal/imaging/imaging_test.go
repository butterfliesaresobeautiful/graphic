package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func closeTo(t *testing.T, a, b float64) {
	t.Helper()
	if math.Abs(a-b) > 1e-8 {
		t.Fatalf("got %.12g want %.12g", a, b)
	}
}
func pattern(w, h int) *image.RGBA {
	m, _ := Canvas(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetRGBA(x, y, color.RGBA{uint8(x*7 + y), uint8(x + y*11), uint8(x ^ y), 255})
		}
	}
	return m
}
func TestLagrangeCardinalityAndPolynomialReproduction(t *testing.T) {
	for _, v := range []float64{-1, 0, .1, .5, .99, 1, 2} {
		weights := Weights(v)
		for degree := 0; degree <= 3; degree++ {
			sum := 0.
			for i, w := range weights {
				sum += w * math.Pow(float64(i-1), float64(degree))
			}
			closeTo(t, sum, math.Pow(v, float64(degree)))
		}
	}
	for node := -1; node <= 2; node++ {
		for i, w := range Weights(float64(node)) {
			want := 0.
			if i-1 == node {
				want = 1
			}
			closeTo(t, w, want)
		}
	}
}
func TestSamplesAgainstAnalyticSurfaces(t *testing.T) {
	m, _ := Canvas(6, 6)
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			m.SetRGBA(x, y, color.RGBA{uint8(20 + x*x*x + 2*y*y), uint8(10 + 5*x + 3*y), 100, 255})
		}
	}
	x, y := 2.3, 1.4
	nearest := Sample(m, x, y, Nearest)
	closeTo(t, nearest[0], 30)
	closeTo(t, nearest[1], 23)
	linear := Sample(m, x, y, Bilinear)
	closeTo(t, linear[0], 38.1)
	closeTo(t, linear[1], 10+5*x+3*y)
	cubic := Sample(m, x, y, Bicubic)
	closeTo(t, cubic[0], 20+x*x*x+2*y*y)
	closeTo(t, cubic[1], 10+5*x+3*y)
	closeTo(t, cubic[2], 100)
	for method := Nearest; method <= Bicubic; method++ {
		for yy := 0; yy < 6; yy++ {
			for xx := 0; xx < 6; xx++ {
				p := Sample(m, float64(xx), float64(yy), method)
				v := m.RGBAAt(xx, yy)
				for c, w := range []uint8{v.R, v.G, v.B} {
					closeTo(t, p[c], float64(w))
				}
			}
		}
	}
}
func TestIdentityQuarterTurnAndNonzeroBounds(t *testing.T) {
	backing := pattern(9, 8)
	src := backing.SubImage(image.Rect(2, 3, 7, 6)).(*image.RGBA)
	before := append([]byte(nil), backing.Pix...)
	r, e := Transform(src, Config{1, 1, 0, 0, 0})
	if e != nil {
		t.Fatal(e)
	}
	for _, m := range r.Images {
		for y := 0; y < 3; y++ {
			for x := 0; x < 5; x++ {
				if m.RGBAAt(x, y) != src.RGBAAt(x+2, y+3) {
					t.Fatal("identity", x, y)
				}
			}
		}
	}
	if r.Metrics.CoveredPixels != 15 {
		t.Fatal("identity coverage")
	}
	for _, d := range r.Metrics.Differences {
		if d.MAE != 0 || d.Max != 0 {
			t.Fatal("identity metrics")
		}
	}
	r, e = Transform(src, Config{1, 1, 90, 0, 0})
	if e != nil {
		t.Fatal(e)
	}
	if r.Images[0].Rect != image.Rect(0, 0, 3, 5) {
		t.Fatal("rotation extent")
	}
	for _, m := range r.Images {
		for y := 0; y < 5; y++ {
			for x := 0; x < 3; x++ {
				if m.RGBAAt(x, y) != src.RGBAAt(2+y, 3+2-x) {
					t.Fatal("clockwise quarter turn", x, y)
				}
			}
		}
	}
	if !bytes.Equal(before, backing.Pix) {
		t.Fatal("source mutated")
	}
}
func TestAffineGeometryAndInverse(t *testing.T) {
	for _, p := range Presets() {
		g, e := Plan(480, 360, p.Config)
		if e != nil {
			t.Fatal(e)
		}
		for _, xy := range [][2]float64{{0, 0}, {480, 0}, {0, 360}, {480, 360}, {117.25, 219.75}} {
			// Independent step-by-step scale, shear and rotation.
			u, v := (xy[0]-240)*p.Config.ScaleX, (xy[1]-180)*p.Config.ScaleY
			u, v = u+p.Config.ShearX*v, p.Config.ShearY*u+v
			s, c := math.Sincos(p.Config.Angle * math.Pi / 180)
			wx, wy := c*u-s*v, s*u+c*v
			x, y := g.Forward.Apply(xy[0]-240, xy[1]-180)
			closeTo(t, x, wx)
			closeTo(t, y, wy)
			if x < g.OriginX-1e-8 || x > g.OriginX+float64(g.Width)+1e-8 || y < g.OriginY-1e-8 || y > g.OriginY+float64(g.Height)+1e-8 {
				t.Fatal("corner clipped", p.Name)
			}
			x, y = g.Inverse.Apply(x, y)
			closeTo(t, x, xy[0]-240)
			closeTo(t, y, xy[1]-180)
		}
	}
}
func TestEdgeReplicationOvershootAndBackground(t *testing.T) {
	m, _ := Canvas(8, 8)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			v := uint8(0)
			if x >= 4 {
				v = 255
			}
			m.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	for method := Nearest; method <= Bicubic; method++ {
		for _, xy := range [][2]float64{{-.49, -.49}, {7.49, 7.49}} {
			got := Sample(m, xy[0], xy[1], method)
			want := 0.
			if xy[0] > 7 {
				want = 255
			}
			for _, v := range got {
				closeTo(t, v, want)
			}
		}
	}
	if Sample(m, 2.5, 3, Bicubic)[0] >= 0 {
		t.Fatal("expected negative cubic lobe")
	}
	r, e := Transform(m, Config{2, 2, 28, 0, 0})
	if e != nil {
		t.Fatal(e)
	}
	if r.Metrics.ClippedChannels[2] == 0 || r.Metrics.ClippedChannels[0] != 0 || r.Metrics.ClippedChannels[1] != 0 {
		t.Fatal("clipping counts", r.Metrics.ClippedChannels)
	}
	outside := 0
	for i, in := range r.Mask {
		if !in {
			outside++
			for _, im := range r.Images {
				for c := 0; c < 4; c++ {
					if im.Pix[4*i+c] != 255 {
						t.Fatal("nonwhite background")
					}
				}
			}
		}
	}
	if outside == 0 {
		t.Fatal("rotation has no background")
	}
	one, _ := Canvas(1, 1)
	one.SetRGBA(0, 0, color.RGBA{11, 22, 33, 255})
	for method := Nearest; method <= Bicubic; method++ {
		p := Sample(one, .4, -.4, method)
		closeTo(t, p[0], 11)
		closeTo(t, p[1], 22)
		closeTo(t, p[2], 33)
	}
}
func TestDifferenceMetricsExcludeBackground(t *testing.T) {
	a, _ := Canvas(2, 1)
	b, _ := Canvas(2, 1)
	a.SetRGBA(0, 0, color.RGBA{13, 14, 10, 255})
	b.SetRGBA(0, 0, color.RGBA{10, 10, 10, 255})
	b.SetRGBA(1, 0, color.RGBA{0, 0, 0, 255})
	r := &Result{Images: [3]*image.RGBA{a, b, nil}, Mask: []bool{true, false}, Metrics: Metrics{CoveredPixels: 1}}
	d := compare(r, 0, 1)
	closeTo(t, d.MAE, 7./3)
	closeTo(t, d.RMSE, math.Sqrt(25./3))
	if d.Max != 4 || d.DifferentPixels != 1 {
		t.Fatal(d)
	}
}
func TestIOAlphaAndSourceProtection(t *testing.T) {
	n := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	n.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 128})
	n.SetNRGBA(1, 0, color.NRGBA{10, 20, 30, 255})
	var buf bytes.Buffer
	if e := png.Encode(&buf, n); e != nil {
		t.Fatal(e)
	}
	src, e := Decode(&buf)
	if e != nil {
		t.Fatal(e)
	}
	if src.RGBAAt(0, 0) != (color.RGBA{255, 127, 127, 255}) {
		t.Fatal("alpha")
	}
	r, e := Transform(src, Config{1, 1, 0, 0, 0})
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	if e = r.Save(dir, ""); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(dir, "source.png")
	before, _ := os.ReadFile(p)
	if e = r.Save(dir, p); e == nil {
		t.Fatal("overwrote source")
	}
	for _, link := range []string{"hard.png", "sym.png"} {
		q := filepath.Join(dir, link)
		if link == "hard.png" {
			e = os.Link(p, q)
		} else {
			e = os.Symlink(p, q)
		}
		if e != nil {
			t.Fatal(e)
		}
		if Protect(q, p) == nil {
			t.Fatal("alias not protected")
		}
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("changed source")
	}
	got, e := Load(filepath.Join(dir, "bicubic.png"))
	if e != nil || !bytes.Equal(got.Pix, src.Pix) {
		t.Fatal("PNG roundtrip", e)
	}
	if _, e = Decode(bytes.NewBufferString("bad")); e == nil {
		t.Fatal("invalid file")
	}
}
func TestInvalidParametersAndOutputLimit(t *testing.T) {
	for _, c := range []Config{{0, 1, 0, 0, 0}, {1, 1, 361, 0, 0}, {1, 1, 0, 1, 1}, {math.NaN(), 1, 0, 0, 0}, {1, math.Inf(1), 0, 0, 0}, {1, 1, 0, 4, 0}} {
		if c.Validate() == nil {
			t.Fatal("invalid accepted", c)
		}
	}
	if _, e := Plan(4096, 4096, Config{1, 1, 0, 0, 0}); e == nil {
		t.Fatal("source pixel cap")
	}
	if _, e := Plan(1000, 1000, Config{8, 8, 0, 0, 0}); e == nil {
		t.Fatal("oversized output")
	}
	if _, e := Transform(nil, Default()); e == nil {
		t.Fatal("nil source")
	}
	one, _ := Canvas(1, 1)
	if _, e := Transform(one, Config{.05, .05, 0, 0, 0}); e == nil {
		t.Fatal("empty coverage")
	}
}
