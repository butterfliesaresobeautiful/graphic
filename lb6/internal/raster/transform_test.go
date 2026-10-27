package raster

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"path/filepath"
	"testing"
)

func pattern(w, h int) *image.RGBA {
	m, _ := New(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetRGBA(x, y, color.RGBA{uint8(x % 251), uint8(y % 251), uint8((x*7 + y*3) % 251), 255})
		}
	}
	return m
}

func TestMatrixOrderAndInverse(t *testing.T) {
	for _, axis := range []string{"vertical", "horizontal"} {
		m, err := Variant8(640, 420, 30, axis)
		if err != nil {
			t.Fatal(err)
		}
		inv, err := m.Inverse()
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range [][2]float64{{0, 0}, {300, 200}, {639, 419}, {319.5, 209.5}} {
			x, y := m.Point(p[0], p[1])
			xx, yy := inv.Point(x, y)
			if math.Abs(xx-p[0]) > 1e-10 || math.Abs(yy-p[1]) > 1e-10 {
				t.Fatal("inverse", p, xx, yy)
			}
			// Independently rotate, then reflect, around the exact centre.
			u, v := p[0]-319.5, p[1]-209.5
			c, s := math.Cos(math.Pi/6), math.Sin(math.Pi/6)
			rx, ry := c*u-s*v, s*u+c*v
			if axis == "vertical" {
				rx = -rx
			} else {
				ry = -ry
			}
			if math.Abs(x-(rx+319.5)) > 1e-10 || math.Abs(y-(ry+209.5)) > 1e-10 {
				t.Fatal("wrong order")
			}
		}
	}
	for _, m := range []Matrix{{}, {1, 2, 0, 2, 4, 0}, {math.NaN(), 0, 0, 0, 1, 0}} {
		if _, err := m.Inverse(); err == nil {
			t.Fatal("bad matrix")
		}
	}
}

func TestReflectionRGBAndSizes(t *testing.T) {
	for _, size := range []image.Point{{1, 1}, {2, 3}, {13, 7}, {640, 420}} {
		src := pattern(size.X, size.Y)
		before := append([]byte(nil), src.Pix...)
		for _, axis := range []string{"vertical", "horizontal"} {
			m, _ := Variant8(size.X, size.Y, 0, axis)
			dst, st, err := Warp(src, m)
			if err != nil || st.Outside != 0 || st.Mapped != size.X*size.Y {
				t.Fatal(st, err)
			}
			for y := 0; y < size.Y; y++ {
				for x := 0; x < size.X; x++ {
					u, v := x, y
					if axis == "vertical" {
						u = size.X - 1 - x
					} else {
						v = size.Y - 1 - y
					}
					if dst.RGBAAt(x, y) != src.RGBAAt(u, v) {
						t.Fatal("RGB mismatch", x, y)
					}
				}
			}
			inv, _ := m.Inverse()
			back, _, err := Warp(dst, inv)
			if err != nil || !bytes.Equal(src.Pix, back.Pix) {
				t.Fatal("exact mirror round-trip")
			}
		}
		if !bytes.Equal(before, src.Pix) {
			t.Fatal("source changed")
		}
	}
}

func TestNinetyDegreesAndClipping(t *testing.T) {
	src := pattern(9, 9)
	m, _ := Variant8(9, 9, 90, "vertical")
	dst, st, err := Warp(src, m)
	if err != nil || st.Outside != 0 {
		t.Fatal(st, err)
	}
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			if dst.RGBAAt(x, y) != src.RGBAAt(y, x) {
				t.Fatal("90-degree transpose")
			}
		}
	}
	rect := pattern(73, 39)
	m, _ = Variant8(73, 39, 37, "horizontal")
	dst, st, err = Warp(rect, m)
	if err != nil || st.Outside == 0 || st.Mapped+st.Outside != 73*39 {
		t.Fatal(st, err)
	}
	white := color.RGBA{255, 255, 255, 255}
	count := 0
	for y := 0; y < 39; y++ {
		for x := 0; x < 73; x++ {
			if dst.RGBAAt(x, y) == white {
				count++
			}
		}
	}
	if count != st.Outside {
		t.Fatal("clipping background", count, st)
	}
}

func TestRestoresTransformedRaster(t *testing.T) {
	src := pattern(100, 70)
	r, err := Process(src, 30, "vertical")
	if err != nil {
		t.Fatal(err)
	}
	mae, _ := MAE(src, r.Restored)
	if mae <= 0 {
		t.Fatal("losses were incorrectly hidden")
	}
	// Changing the transformed raster must affect the restored image.
	for i := 0; i < len(r.Affine.Pix); i += 4 {
		r.Affine.Pix[i], r.Affine.Pix[i+1], r.Affine.Pix[i+2] = 200, 10, 80
	}
	inv, _ := r.Matrix.Inverse()
	back, _, err := Warp(r.Affine, inv)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(back.Pix, r.Restored.Pix) {
		t.Fatal("restoration ignores transformed input")
	}
}

func TestExactNonlinearFormula(t *testing.T) {
	for x, want := range map[float64]float64{-64: -1024, -1: -1, 0: 0, 1: 1, 64: 1024} {
		if math.Abs(Power53(x)-want) > 1e-9 {
			t.Fatal(x, Power53(x))
		}
		if math.Abs(Power35(Power53(x))-x) > 1e-10 {
			t.Fatal("inverse power", x)
		}
	}
	for _, size := range []image.Point{{1, 1}, {2, 3}, {33, 17}, {640, 420}} {
		src := pattern(size.X, size.Y)
		dst, st, err := Nonlinear(src)
		if err != nil || st.Outside != 0 {
			t.Fatal(st, err)
		}
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				// Literal pixel coordinates: no unit-interval normalisation allowed.
				u := int(math.Round(math.Pow(float64(x), .6)))
				if dst.RGBAAt(x, y) != src.RGBAAt(u, y) {
					t.Fatal("nonlinear mapping", x, y, u)
				}
			}
		}
	}
}

func TestErrorsAndPNG(t *testing.T) {
	for _, angle := range []float64{math.NaN(), math.Inf(1), 361} {
		if _, err := Variant8(10, 10, angle, "vertical"); err == nil {
			t.Fatal(angle)
		}
	}
	if _, err := Variant8(10, 10, 0, "diagonal"); err == nil {
		t.Fatal("invalid axis")
	}
	if _, _, err := Nonlinear(nil); err == nil {
		t.Fatal("nil")
	}
	if _, err := New(-1, 7); err == nil {
		t.Fatal("invalid canvas")
	}
	if _, err := Decode(bytes.NewBufferString("broken image")); err == nil {
		t.Fatal("corrupt file")
	}
	src := pattern(31, 20)
	path := filepath.Join(t.TempDir(), "a.png")
	if err := Save(path, src); err != nil {
		t.Fatal(err)
	}
	read, err := Load(path)
	if err != nil || !bytes.Equal(read.Pix, src.Pix) {
		t.Fatal("PNG roundtrip", err)
	}
	if err := Save(filepath.Join(path, "bad.png"), src); err == nil {
		t.Fatal("unwritable destination")
	}
	if _, err := Load(path + "missing"); err == nil {
		t.Fatal("missing file")
	}
}
