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

func pattern(w, h int) *image.RGBA {
	m, _ := Canvas(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetRGBA(x, y, color.RGBA{uint8(x*7 + y), uint8(x + y*11), uint8(x ^ y), 255})
		}
	}
	return m
}
func TestExactRGBTransferAndVariant(t *testing.T) {
	for _, sz := range [][2]int{{80, 60}, {800, 600}, {123, 99}} {
		src := pattern(sz[0], sz[1])
		before := append([]byte(nil), src.Pix...)
		c := Default()
		c.Radius = min(70, (min(sz[0]/2, sz[1]/2)-1)/2)
		c, e := c.Resolve(src.Rect)
		if e != nil {
			t.Fatal(e)
		}
		dst, _ := Canvas(c.Width, c.Height)
		st, e := Transfer(src, dst, c)
		if e != nil {
			t.Fatal(e)
		}
		count := 0
		for y := 0; y < c.Height; y++ {
			for x := 0; x < c.Width; x++ {
				ox, oy := x-c.DX, y-c.DY
				got := dst.RGBAAt(x, y)
				if ox*ox+oy*oy <= c.Radius*c.Radius {
					count++
					want := src.RGBAAt(c.SX+ox, c.SY+oy)
					if got != want {
						t.Fatalf("RGB mismatch %d %d", x, y)
					}
				} else if got != (color.RGBA{255, 255, 255, 255}) {
					t.Fatal("outside disk changed")
				}
			}
		}
		if st.Copied != count || st.Clipped != 0 || !bytes.Equal(before, src.Pix) {
			t.Fatal("count or source mutation")
		}
		if c.Radius == 70 && st.Copied != 15373 {
			t.Fatal("default disk count", st)
		}
		if c.SX != c.Radius || c.SY != sz[1]-1-c.Radius || c.DX != c.Width/2 || c.DY != c.Radius {
			t.Fatal("variant placement")
		}
	}
}
func TestClippingAndInclusiveBoundary(t *testing.T) {
	src := pattern(5, 5)
	dst, _ := Canvas(5, 5)
	c := Config{Width: 5, Height: 5, Radius: 2, SX: 0, SY: 0, DX: 0, DY: 0}
	st, e := Transfer(src, dst, c)
	if e != nil {
		t.Fatal(e)
	}
	if st.Inside != 13 || st.Copied != 6 || st.Clipped != 7 {
		t.Fatal(st)
	}
	for _, p := range []image.Point{{0, 0}, {0, 2}, {2, 0}} {
		if dst.RGBAAt(p.X, p.Y) != src.RGBAAt(p.X, p.Y) {
			t.Fatal("excluded zero/boundary")
		}
	}
	c.DX = -100
	c.DY = -100
	before := append([]byte(nil), dst.Pix...)
	if _, e = Transfer(src, dst, c); e == nil {
		t.Fatal("empty copy accepted")
	}
	if !bytes.Equal(before, dst.Pix) {
		t.Fatal("empty copy modified canvas")
	}
	sub := src.SubImage(image.Rect(1, 1, 5, 5)).(*image.RGBA)
	c.SX, c.SY, c.DX, c.DY = 1, 1, 2, 2
	if _, e = Transfer(sub, dst, c); e != nil {
		t.Fatal(e)
	}
	if dst.RGBAAt(2, 2) != src.RGBAAt(1, 1) {
		t.Fatal("nonzero bounds")
	}
}
func TestStateMachineAndFrozenParameters(t *testing.T) {
	d := Document{}
	c := Default()
	if e := d.Copy(c); e == nil {
		t.Fatal("copied before source/canvas")
	}
	if e := d.AddAxes(); e == nil {
		t.Fatal("axes before copy")
	}
	if e := d.AddGraph(); e == nil {
		t.Fatal("plot before axes")
	}
	if e := d.Create(c); e != nil {
		t.Fatal(e)
	}
	d.SetSource(Demo())
	if d.Stage != Blank || d.Images[0] == nil {
		t.Fatal("opening source lost canvas")
	}
	changed := c
	changed.Width++
	if e := d.Copy(changed); e == nil {
		t.Fatal("resized without create")
	}
	if e := d.Copy(c); e != nil {
		t.Fatal(e)
	}
	copied := append([]byte(nil), d.Current().Pix...)
	if e := d.AddGraph(); e == nil {
		t.Fatal("graph before axes")
	}
	if d.Stage != Copied || !bytes.Equal(copied, d.Current().Pix) {
		t.Fatal("failed operation changed result")
	}
	if e := d.AddAxes(); e != nil {
		t.Fatal(e)
	}
	if e := d.AddGraph(); e != nil {
		t.Fatal(e)
	}
	result := append([]byte(nil), d.Current().Pix...)
	if e := d.AddGraph(); e != nil || !bytes.Equal(result, d.Current().Pix) {
		t.Fatal("plot not idempotent")
	}
	if e := d.Copy(c); e != nil || d.Images[2] != nil || d.Images[3] != nil {
		t.Fatal("stale layers after copy")
	}
	d.SetSource(pattern(800, 600))
	if d.Stage != Blank || d.Images[1] != nil {
		t.Fatal("stale source layers")
	}
}
func TestCubicMappingAndPreservedDisk(t *testing.T) {
	src := Demo()
	d, e := Build(src, Default())
	if e != nil {
		t.Fatal(e)
	}
	p, e := Layout(d.Config)
	if e != nil {
		t.Fatal(e)
	}
	for _, x := range []float64{-2, -1, 0, 1, 2} {
		q := p.Point(x, x*x*x)
		if !q.In(p.Rect) || d.Current().RGBAAt(q.X, q.Y) != CurveColor {
			t.Fatal("curve misses exact point", x, q)
		}
	}
	if p.Point(0, 8).Y >= p.Point(0, -8).Y {
		t.Fatal("Y direction")
	}
	for y := 0; y < 2*d.Config.Radius+1; y++ {
		for x := 0; x < d.Config.Width; x++ {
			if d.Images[1].RGBAAt(x, y) != d.Current().RGBAAt(x, y) {
				t.Fatal("axes/curve damaged disk")
			}
		}
	}
	// Every plotted blue pixel inside the plot is near the analytic curve in pixels.
	for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
		for x := p.Rect.Min.X; x < p.Rect.Max.X; x++ {
			if d.Current().RGBAAt(x, y) != CurveColor {
				continue
			}
			arg := (float64(x) - p.OX) / p.SX
			want := p.OY - p.SY*arg*arg*arg
			if math.Abs(want-float64(y)) > 2 {
				t.Fatal("curve error", x, y, want)
			}
		}
	}
}
func TestIOAlphaAndOverwriteProtection(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 128})
	src.SetNRGBA(1, 0, color.NRGBA{10, 20, 30, 255})
	var b bytes.Buffer
	if e := png.Encode(&b, src); e != nil {
		t.Fatal(e)
	}
	decoded, e := Decode(&b)
	if e != nil {
		t.Fatal(e)
	}
	if decoded.RGBAAt(0, 0) != (color.RGBA{255, 127, 127, 255}) || decoded.RGBAAt(1, 0) != (color.RGBA{10, 20, 30, 255}) {
		t.Fatal("alpha/RGB conversion")
	}
	d, e := Build(Demo(), Default())
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	if e = d.Export(dir, ""); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(dir, "result.png")
	got, e := Load(p)
	if e != nil || !bytes.Equal(got.Pix, d.Current().Pix) {
		t.Fatal("PNG roundtrip", e)
	}
	before, _ := os.ReadFile(p)
	if e = d.SaveCurrent(dir, p); e == nil {
		t.Fatal("overwrote source")
	}
	link := filepath.Join(dir, "alias.png")
	if e = os.Link(p, link); e != nil {
		t.Fatal(e)
	}
	if e = Protect(link, p); e == nil {
		t.Fatal("hardlink alias")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("source changed")
	}
	if _, e = Decode(bytes.NewBufferString("not an image")); e == nil {
		t.Fatal("bad image accepted")
	}
}
func TestInvalidParameters(t *testing.T) {
	for _, c := range []Config{{Width: -1, Height: 720, Radius: 70}, {Width: 1000, Height: 720, Radius: 0}, {Width: 1000, Height: 720, Radius: 2049}, {Width: 1000, Height: 720, Radius: 70, SX: 99999}} {
		if c.Validate() == nil {
			t.Fatal("invalid accepted", c)
		}
	}
	c := Default()
	c.Radius = 200
	if _, e := c.Resolve(image.Rect(0, 0, 800, 600)); e == nil {
		t.Fatal("wrong source quadrant")
	}
	c = Default()
	c.Width = 100
	if _, e := c.Resolve(image.Rect(0, 0, 800, 600)); e == nil {
		t.Fatal("circle does not fit")
	}
	c = Default()
	c.Height = 240
	if _, e := Layout(c); e == nil {
		t.Fatal("unreadable graph accepted")
	}
}
