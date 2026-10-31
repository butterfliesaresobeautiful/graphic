package imaging

import (
	"errors"
	"image"
	"image/color"
	"math"
	"strconv"
)

var AxisColor = color.RGBA{40, 56, 76, 255}
var CurveColor = color.RGBA{20, 98, 200, 255}

type Plot struct {
	Rect           image.Rectangle
	OX, OY, SX, SY float64
}

func Layout(c Config) (Plot, error) {
	r := image.Rect(64, max(c.Height/3, 2*c.Radius+52), c.Width-32, c.Height-48)
	if c.Width < 280 || r.Dx() < 160 || r.Dy() < 160 {
		return Plot{}, errors.New("для осей и графика увеличьте холст или уменьшите радиус")
	}
	return Plot{r, float64(r.Min.X+r.Max.X-1) / 2, float64(r.Min.Y+r.Max.Y-1) / 2, float64(r.Dx()-1) / 4, float64(r.Dy()-1) / 16}, nil
}
func (p Plot) Point(x, y float64) image.Point {
	return image.Pt(int(math.Round(p.OX+p.SX*x)), int(math.Round(p.OY-p.SY*y)))
}
func Line(dst *image.RGBA, a, b image.Point, col color.RGBA, clip image.Rectangle) {
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
		if a.In(clip) && a.In(dst.Rect) {
			dst.SetRGBA(a.X, a.Y, col)
		}
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
func Axes(dst *image.RGBA, c Config) error {
	if dst == nil {
		return errors.New("сначала создайте холст")
	}
	p, err := Layout(c)
	if err != nil {
		return err
	}
	r := p.Rect
	grid := color.RGBA{222, 230, 240, 255}
	ox, oy := p.Point(0, 0).X, p.Point(0, 0).Y
	for x := -2; x <= 2; x++ {
		q := p.Point(float64(x), 0)
		Line(dst, image.Pt(q.X, r.Min.Y), image.Pt(q.X, r.Max.Y-1), grid, r)
		Line(dst, image.Pt(q.X, oy-4), image.Pt(q.X, oy+4), AxisColor, dst.Rect)
		Text(dst, q.X-7, oy+12, strconv.Itoa(x), 2, AxisColor)
	}
	for y := -8; y <= 8; y += 2 {
		q := p.Point(0, float64(y))
		Line(dst, image.Pt(r.Min.X, q.Y), image.Pt(r.Max.X-1, q.Y), grid, r)
		Line(dst, image.Pt(ox-4, q.Y), image.Pt(ox+4, q.Y), AxisColor, dst.Rect)
		if y != 0 {
			Text(dst, ox-40, q.Y-7, strconv.Itoa(y), 2, AxisColor)
		}
	}
	Line(dst, image.Pt(r.Min.X, oy), image.Pt(r.Max.X+12, oy), AxisColor, dst.Rect)
	Line(dst, image.Pt(ox, r.Max.Y-1), image.Pt(ox, r.Min.Y-12), AxisColor, dst.Rect)
	for _, a := range []image.Point{{r.Max.X + 4, oy - 5}, {r.Max.X + 4, oy + 5}} {
		Line(dst, a, image.Pt(r.Max.X+12, oy), AxisColor, dst.Rect)
	}
	for _, a := range []image.Point{{ox - 5, r.Min.Y - 4}, {ox + 5, r.Min.Y - 4}} {
		Line(dst, a, image.Pt(ox, r.Min.Y-12), AxisColor, dst.Rect)
	}
	Text(dst, r.Max.X+9, oy+10, "x", 2, AxisColor)
	Text(dst, ox+12, r.Min.Y-17, "y", 2, AxisColor)
	Text(dst, r.Min.X+4, r.Min.Y-30, "y = x^3", 3, CurveColor)
	return nil
}
func Cubic(dst *image.RGBA, c Config) error {
	if dst == nil {
		return errors.New("сначала создайте холст")
	}
	p, err := Layout(c)
	if err != nil {
		return err
	}
	n := 4 * (p.Rect.Dx() - 1)
	prev := p.Point(-2, -8)
	for i := 1; i <= n; i++ {
		x := -2 + 4*float64(i)/float64(n)
		cur := p.Point(x, x*x*x)
		Line(dst, prev, cur, CurveColor, p.Rect)
		prev = cur
	}
	return nil
}

// Small bitmap labels keep plotting independent of GUI/font libraries.
var glyphs = map[rune][7]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'=': {"00000", "00000", "11111", "00000", "11111", "00000", "00000"},
	'^': {"00100", "01010", "10001", "00000", "00000", "00000", "00000"},
	'x': {"00000", "00000", "10001", "01010", "00100", "01010", "10001"},
	'y': {"00000", "00000", "10001", "10001", "01111", "00001", "01110"},
}

func Text(dst *image.RGBA, x, y int, s string, scale int, c color.RGBA) {
	for _, ch := range s {
		if g, ok := glyphs[ch]; ok {
			for row, bits := range g {
				for col, v := range bits {
					if v != '1' {
						continue
					}
					for j := 0; j < scale; j++ {
						for i := 0; i < scale; i++ {
							p := image.Pt(x+col*scale+i, y+row*scale+j)
							if p.In(dst.Rect) {
								dst.SetRGBA(p.X, p.Y, c)
							}
						}
					}
				}
			}
		}
		x += 6 * scale
	}
}
