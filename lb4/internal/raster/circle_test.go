package raster

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func connected(m *Mask) bool {
	start := -1
	for i, v := range m.Bits {
		if v {
			start = i
			break
		}
	}
	if start < 0 {
		return false
	}
	seen := map[int]bool{start: true}
	q := []int{start}
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		x, y := n%m.W, n/m.W
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				xx, yy := x+dx, y+dy
				k := yy*m.W + xx
				if m.At(xx, yy) && !seen[k] {
					seen[k] = true
					q = append(q, k)
				}
			}
		}
	}
	return len(seen) == m.Count()
}
func TestReuleauxGeometry(t *testing.T) {
	c := Default()
	v := c.Vertices()
	for i := 0; i < 3; i++ {
		a, b := v[i], v[(i+1)%3]
		if math.Abs(math.Hypot(a.X-b.X, a.Y-b.Y)-float64(c.Size)) > 1e-9 {
			t.Fatal("side length")
		}
	}
	// Independent support-width check over dense continuous arc samples.
	var points []Point
	for _, a := range c.Arcs() {
		for k := 0; k <= 4096; k++ {
			points = append(points, a.At(a.Start+a.Sweep*float64(k)/4096))
		}
	}
	for degree := 0; degree < 180; degree++ {
		ux, uy := math.Cos(float64(degree)*math.Pi/180), math.Sin(float64(degree)*math.Pi/180)
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, p := range points {
			s := p.X*ux + p.Y*uy
			lo = math.Min(lo, s)
			hi = math.Max(hi, s)
		}
		if math.Abs(hi-lo-float64(c.Size)) > .001 {
			t.Fatalf("width at %d: %.8f", degree, hi-lo)
		}
	}
}
func TestCircleAccuracySymmetryAndConnectivity(t *testing.T) {
	for _, radius := range []int{1, 2, 3, 7, 25, 100, 280} {
		c := Config{640, 640, 320, 320, 280, radius, true}
		ms, err := Custom(c)
		if err != nil {
			t.Fatal(err)
		}
		for method, m := range ms {
			if !connected(m) {
				t.Fatalf("disconnected radius=%d method=%d", radius, method)
			}
			for _, p := range []image.Point{{320 + radius, 320}, {320 - radius, 320}, {320, 320 + radius}, {320, 320 - radius}} {
				if !m.At(p.X, p.Y) {
					t.Fatal("missing cardinal")
				}
			}
			for n, on := range m.Bits {
				if !on {
					continue
				}
				x, y := n%m.W-320, n/m.W-320
				if math.Abs(math.Hypot(float64(x), float64(y))-float64(radius)) > 1 {
					t.Fatalf("radial error %d %d", radius, method)
				}
				// The angular stepping method is allowed different tie-breaking.
				if method != 1 && (!m.At(320-x, 320+y) || !m.At(320+y, 320+x)) {
					t.Fatal("broken eightfold symmetry")
				}
			}
		}
	}
}
func TestReuleauxContoursAndTranslation(t *testing.T) {
	for _, size := range []int{1, 2, 3, 7, 31, 80, 279, 280} {
		c := Default()
		c.Size = size
		ms, err := Custom(c)
		if err != nil {
			t.Fatal(err)
		}
		c2 := c
		c2.CX += 7
		c2.CY -= 5
		shift, _ := Custom(c2)
		for method, m := range ms {
			if !connected(m) {
				t.Fatalf("disconnected size=%d method=%d", size, method)
			}
			for _, p := range c.Vertices() {
				if !m.At(int(math.Round(p.X)), int(math.Round(p.Y))) {
					t.Fatal("missing shared vertex")
				}
			}
			for i, on := range m.Bits {
				if !on {
					continue
				}
				x, y := i%m.W, i/m.W
				if !shift[method].At(x+7, y-5) {
					t.Fatal("translation changed shape")
				}
				closest := math.Inf(1)
				for _, a := range c.Arcs() {
					e := math.Hypot(float64(x)-a.Center.X, float64(y)-a.Center.Y) - float64(size)
					closest = math.Min(closest, math.Abs(e))
					if e > 1.5 {
						t.Fatal("outside disk intersection", size, method)
					}
				}
				if closest > 1.2 {
					t.Fatal("not near boundary", size, method, closest)
				}
			}
		}
	}
}
func TestBresenhamDecisionAndClipping(t *testing.T) {
	for _, r := range []int{1, 2, 13, 280, 2048} {
		steps := Trace(r)
		for i, s := range steps {
			// d = 2*F(x+1,y-1/2)+1/2, evaluated using integer arithmetic.
			expected := 2*(s.X+1)*(s.X+1) + 2*s.Y*s.Y - 2*s.Y + 1 - 2*r*r
			if s.D != expected {
				t.Fatalf("decision invariant: %+v want %d", s, expected)
			}
			if i > 0 && (s.X != steps[i-1].X+1 || s.Y > steps[i-1].Y || s.Y < steps[i-1].Y-1) {
				t.Fatal("invalid step")
			}
		}
	}
	c := Default()
	c.CX = 0
	c.CY = 0
	ms, _ := Custom(c)
	for _, m := range ms {
		if m.Clipped == 0 || m.Count() == 0 {
			t.Fatal("clipping")
		}
	}
	c.CX = -4096
	c.CY = -4096
	ms, _ = Custom(c)
	for _, m := range ms {
		if m.Count() != 0 {
			t.Fatal("offscreen")
		}
	}
}
func TestResultsIOAndInvalidInputs(t *testing.T) {
	c := Default()
	c.Width = 80
	c.Height = 70
	c.CX = 40
	c.CY = 35
	c.Size = 40
	bg, _ := New(80, 70)
	bg.SetRGBA(0, 0, color.RGBA{10, 20, 30, 255})
	before := append([]byte(nil), bg.Pix...)
	m, _ := Custom(c)
	r, err := Compose(bg, c, [4]*Mask{m[0], m[1], m[2], m[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, bg.Pix) || r.Images[0].RGBAAt(0, 0) != bg.RGBAAt(0, 0) {
		t.Fatal("background changed")
	}
	if r.Metrics.PairwiseXOR[0][3] != 0 || r.Metrics.Common+r.Metrics.Different != r.Metrics.Union {
		t.Fatal("comparison")
	}
	dir := t.TempDir()
	if err = r.SaveAll(dir, ""); err != nil {
		t.Fatal(err)
	}
	img, err := Load(filepath.Join(dir, "equation.png"))
	if err != nil || !bytes.Equal(img.Pix, r.Images[0].Pix) {
		t.Fatal("PNG roundtrip", err)
	}
	if err = r.SaveAll(dir, filepath.Join(dir, "equation.png")); err == nil {
		t.Fatal("overwrote input")
	}
	if err = os.WriteFile(filepath.Join(dir, "bad.png"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(filepath.Join(dir, "bad.png")); err == nil {
		t.Fatal("bad input accepted")
	}
	for _, bad := range []Config{{0, 10, 0, 0, 10, 10, false}, {10, 10, 0, 0, 0, 10, false}, {10, 10, 0, 0, 10, 0, false}, {10, 10, 99999, 0, 10, 10, false}} {
		if _, err = Custom(bad); err == nil {
			t.Fatal("invalid accepted")
		}
	}
}
