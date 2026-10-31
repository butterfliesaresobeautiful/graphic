package raster

import (
	"image"
	"math"
)

// Mask is independent of background colour, so comparison never confuses a
// source pixel with a contour pixel. Clipping occurs only in Set.
type Mask struct {
	W, H    int
	Bits    []bool
	Clipped int
}

func NewMask(w, h int) *Mask { return &Mask{W: w, H: h, Bits: make([]bool, w*h)} }
func (m *Mask) Set(x, y int) {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		m.Clipped++
		return
	}
	m.Bits[y*m.W+x] = true
}
func (m *Mask) At(x, y int) bool { return x >= 0 && y >= 0 && x < m.W && y < m.H && m.Bits[y*m.W+x] }
func (m *Mask) Count() int {
	n := 0
	for _, v := range m.Bits {
		if v {
			n++
		}
	}
	return n
}
func (m *Mask) point(p Point) { m.Set(int(math.Round(p.X)), int(math.Round(p.Y))) }
func (m *Mask) symmetric(a Arc, x, y int) {
	for _, p := range [8]image.Point{{x, y}, {y, x}, {-x, y}, {-y, x}, {x, -y}, {y, -x}, {-x, -y}, {-y, -x}} {
		if a.Contains(p.X, p.Y) {
			m.point(Point{a.Center.X + float64(p.X), a.Center.Y + float64(p.Y)})
		}
	}
}

// Equation evaluates one octant and uses eightfold symmetry. Restricting x<=y
// avoids the large slope / missing-pixel issue of scanning an entire quadrant.
func Equation(m *Mask, a Arc) {
	r := float64(a.Radius)
	for x := 0; float64(x) <= math.Ceil(r/math.Sqrt2); x++ {
		y := int(math.Round(math.Sqrt(r*r - float64(x*x))))
		if x > y {
			break
		}
		m.symmetric(a, x, y)
	}
	endpoints(m, a)
}

// Step records the integer decision variable before a Bresenham iteration.
type Step struct{ X, Y, D int }

func Trace(radius int) []Step {
	x, y, d := 0, radius, 3-2*radius
	steps := make([]Step, 0, radius+1)
	for x <= y {
		steps = append(steps, Step{x, y, d})
		if d < 0 {
			d += 4*x + 6
		} else {
			d += 4*(x-y) + 10
			y--
		}
		x++
	}
	return steps
}
func Bresenham(m *Mask, a Arc) {
	for _, s := range Trace(a.Radius) {
		m.symmetric(a, s.X, s.Y)
	}
	endpoints(m, a)
}

// line joins neighbouring parametric samples; no external rasteriser is used.
func line(m *Mask, a, b image.Point) {
	dx, dy := abs(b.X-a.X), -abs(b.Y-a.Y)
	sx, sy := -1, -1
	if a.X < b.X {
		sx = 1
	}
	if a.Y < b.Y {
		sy = 1
	}
	err := dx + dy
	for {
		m.Set(a.X, a.Y)
		if a == b {
			return
		}
		e := 2 * err
		if e >= dy {
			err += dy
			a.X += sx
		}
		if e <= dx {
			err += dx
			a.Y += sy
		}
	}
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func Parametric(m *Mask, a Arc) {
	// Uniform arc-length step <= 1 pixel, with both endpoints included.
	n := max(1, int(math.Ceil(a.Sweep*math.Pi/180*float64(a.Radius))))
	var prev image.Point
	for k := 0; k <= n; k++ {
		p := a.At(a.Start + a.Sweep*float64(k)/float64(n))
		cur := image.Pt(int(math.Round(p.X)), int(math.Round(p.Y)))
		if k == 0 {
			m.Set(cur.X, cur.Y)
		} else {
			line(m, prev, cur)
		}
		prev = cur
	}
}
func endpoints(m *Mask, a Arc) {
	if a.Sweep < 360 {
		m.point(a.At(a.Start))
		m.point(a.At(a.Start + a.Sweep))
	}
}
func Custom(c Config) ([3]*Mask, error) {
	var masks [3]*Mask
	if err := c.Validate(); err != nil {
		return masks, err
	}
	fs := []func(*Mask, Arc){Equation, Parametric, Bresenham}
	for i, f := range fs {
		masks[i] = NewMask(c.Width, c.Height)
		for _, a := range c.Arcs() {
			f(masks[i], a)
		}
	}
	return masks, nil
}
