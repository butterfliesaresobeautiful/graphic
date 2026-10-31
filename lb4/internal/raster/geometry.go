package raster

import (
	"errors"
	"image"
	"math"
)

// Config uses screen coordinates: X right, Y down. Size is the Reuleaux width.
type Config struct {
	Width, Height        int
	CX, CY, Size, Radius int
	Circle               bool
}
type Point struct{ X, Y float64 }
type Arc struct {
	Center       Point
	Radius       int
	Start, Sweep float64
}

func Default() Config { return Config{640, 480, 320, 240, 280, 140, false} }
func (c Config) Validate() error {
	if err := Dimensions(c.Width, c.Height); err != nil {
		return err
	}
	if c.Size < 1 || c.Size > 2048 || c.Radius < 1 || c.Radius > 2048 {
		return errors.New("ширина и радиус: целые числа 1…2048")
	}
	if c.CX < -4096 || c.CX > 4096 || c.CY < -4096 || c.CY > 4096 {
		return errors.New("координаты центра: −4096…4096")
	}
	return nil
}
func (c Config) Vertices() [3]Point {
	x, y, w := float64(c.CX), float64(c.CY), float64(c.Size)
	return [3]Point{{x, y - w/math.Sqrt(3)}, {x - w/2, y + w/(2*math.Sqrt(3))}, {x + w/2, y + w/(2*math.Sqrt(3))}}
}
func (c Config) Arcs() []Arc {
	if c.Circle {
		return []Arc{{Point{float64(c.CX), float64(c.CY)}, c.Radius, 0, 360}}
	}
	v := c.Vertices()
	return []Arc{{v[0], c.Size, 60, 60}, {v[1], c.Size, 300, 60}, {v[2], c.Size, 180, 60}}
}
func (a Arc) At(degrees float64) Point {
	t := degrees * math.Pi / 180
	return Point{a.Center.X + float64(a.Radius)*math.Cos(t), a.Center.Y + float64(a.Radius)*math.Sin(t)}
}
func (a Arc) Contains(x, y int) bool {
	if a.Sweep >= 360 {
		return true
	}
	t := math.Atan2(float64(y), float64(x)) * 180 / math.Pi
	d := math.Mod(t-a.Start+720, 360)
	return d <= a.Sweep+1e-9
}
func (c Config) ZoomRect() image.Rectangle {
	a := c.Arcs()[0]
	angle := 90.0
	if c.Circle {
		angle = 45
	} else {
		a = c.Arcs()[1]
		angle = 330
	}
	p := a.At(angle)
	x, y := int(math.Round(p.X)), int(math.Round(p.Y))
	return image.Rect(x-16, y-16, x+16, y+16)
}
