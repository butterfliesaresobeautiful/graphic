package scene

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"graphiclabs/lb10/internal/bezier"
)

var (
	Background = color.RGBA{249, 252, 254, 255}
	Original   = color.RGBA{83, 103, 131, 255}
	Refined    = color.RGBA{189, 91, 38, 255}
	Step       = color.RGBA{117, 73, 159, 255}
	Result     = color.RGBA{1, 109, 133, 255}
	Selection  = color.RGBA{18, 31, 48, 255}
	gridLine   = color.RGBA{232, 239, 245, 255}
	axisLine   = color.RGBA{201, 215, 229, 255}
	white      = color.RGBA{255, 255, 255, 255}
)

type Layers struct {
	Original, Refined bool
	Selected          int // -1 hides the selection ring.
}

type CurveView struct {
	Control      []bezier.Point
	Curve        bezier.Curve
	Construction bezier.Construction
	Stage        int
	Layers       Layers
}

type SurfaceView struct {
	Control bezier.Grid
	Surface bezier.Surface
	Layers  Layers
}

type plot struct {
	image  *image.RGBA
	box    image.Rectangle
	mapper Projection
}

func newPlot(img *image.RGBA, box image.Rectangle, view Camera) plot {
	box = box.Intersect(img.Bounds())
	draw.Draw(img, box, image.NewUniform(Background), image.Point{}, draw.Src)
	return plot{img, box, view.Projection(box)}
}

func DrawCurve(img *image.RGBA, box image.Rectangle, view Camera, c CurveView) {
	p := newPlot(img, box, view)
	p.planeGrid()
	if c.Layers.Original {
		p.polyline(c.Control, Original, 1)
	}
	if c.Layers.Refined {
		for _, side := range c.Construction.Split {
			p.polyline(side, Refined, 1)
			p.points(side, Refined, 4)
		}
	}
	for i := 1; i <= c.Stage; i++ {
		level := c.Construction.Levels[i]
		p.polyline(level, Step, 1)
		p.points(level, Step, 4)
	}
	p.polyline(c.Curve.Samples, Result, 3)
	p.points(c.Curve.Samples, Result, 2)
	if c.Stage == len(c.Construction.Levels)-1 {
		p.marker(c.Construction.At(), Result)
	}
	if c.Layers.Original {
		p.points(c.Control, Original, 5)
		p.selection(c.Control, c.Layers.Selected)
	}
}

func DrawSurface(img *image.RGBA, box image.Rectangle, view Camera, s SurfaceView) {
	p := newPlot(img, box, view)
	p.surface(s.Surface.Samples)
	p.mesh(s.Surface.Samples, Result, 0)
	if s.Layers.Refined {
		for _, patch := range s.Surface.Patches {
			p.mesh(patch, Refined, 3)
		}
	}
	if s.Layers.Original {
		p.mesh(s.Control, Original, 5)
		p.selection(s.Control.Points(), s.Layers.Selected)
	}
	p.marker(s.Surface.At, Result)
}

func (p plot) planeGrid() {
	lo := p.mapper.Plane(image.Pt(p.box.Min.X, p.box.Max.Y))
	hi := p.mapper.Plane(image.Pt(p.box.Max.X, p.box.Min.Y))
	step := math.Pow10(int(math.Floor(math.Log10(70 / p.mapper.scale))))
	if step*p.mapper.scale < 32 {
		step *= 5
	}
	for x := math.Ceil(lo.X/step) * step; x <= hi.X; x += step {
		screen := p.mapper.Screen(bezier.Point{X: x})
		p.line(pixel{x: float64(screen.X), y: float64(p.box.Min.Y)},
			pixel{x: float64(screen.X), y: float64(p.box.Max.Y - 1)}, gridLine, 1)
	}
	for y := math.Ceil(lo.Y/step) * step; y <= hi.Y; y += step {
		screen := p.mapper.Screen(bezier.Point{Y: y})
		p.line(pixel{x: float64(p.box.Min.X), y: float64(screen.Y)},
			pixel{x: float64(p.box.Max.X - 1), y: float64(screen.Y)}, gridLine, 1)
	}
	p.polyline([]bezier.Point{{X: lo.X}, {X: hi.X}}, axisLine, 1)
	p.polyline([]bezier.Point{{Y: lo.Y}, {Y: hi.Y}}, axisLine, 1)
}

func (p plot) selection(points []bezier.Point, selected int) {
	if selected >= 0 && selected < len(points) {
		p.marker(points[selected], Selection)
	}
}

func (p plot) marker(point bezier.Point, ink color.RGBA) {
	center := p.mapper.Screen(point)
	p.dot(center, 8, white)
	p.dot(center, 6, ink)
	p.dot(center, 2, white)
}

func (p plot) points(points []bezier.Point, ink color.RGBA, radius int) {
	for _, point := range points {
		p.dot(p.mapper.Screen(point), radius, ink)
	}
}

func (p plot) dot(center image.Point, radius int, ink color.RGBA) {
	r := image.Rect(center.X-radius, center.Y-radius, center.X+radius+1, center.Y+radius+1)
	r = r.Intersect(p.box)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dx, dy := x-center.X, y-center.Y
			if dx*dx+dy*dy <= radius*radius {
				p.image.SetRGBA(x, y, ink)
			}
		}
	}
}

func (p plot) polyline(points []bezier.Point, ink color.RGBA, width int) {
	for i := 1; i < len(points); i++ {
		p.line(p.mapper.mapPoint(points[i-1]), p.mapper.mapPoint(points[i]), ink, width)
	}
}

func (p plot) mesh(g bezier.Grid, ink color.RGBA, radius int) {
	for j, row := range g {
		p.polyline(row, ink, 1)
		for i, point := range row {
			if j > 0 {
				p.line(p.mapper.mapPoint(g[j-1][i]), p.mapper.mapPoint(point), ink, 1)
			}
			if radius > 0 {
				p.dot(p.mapper.Screen(point), radius, ink)
			}
		}
	}
}

func (p plot) line(a, b pixel, ink color.RGBA, width int) {
	dx, dy := b.x-a.x, b.y-a.y
	enter, leave := 0.0, 1.0
	for _, dim := range [2][4]float64{
		{a.x, dx, float64(p.box.Min.X), float64(p.box.Max.X - 1)},
		{a.y, dy, float64(p.box.Min.Y), float64(p.box.Max.Y - 1)},
	} {
		start, delta, lo, hi := dim[0], dim[1], dim[2], dim[3]
		if delta == 0 {
			if start < lo || start > hi {
				return
			}
			continue
		}
		t0, t1 := (lo-start)/delta, (hi-start)/delta
		enter = max(enter, min(t0, t1))
		leave = min(leave, max(t0, t1))
	}
	if enter > leave {
		return
	}
	steps := max(1, int(math.Ceil((leave-enter)*max(math.Abs(dx), math.Abs(dy)))))
	for i := 0; i <= steps; i++ {
		t := enter + (leave-enter)*float64(i)/float64(steps)
		point := image.Pt(int(math.Round(a.x+t*dx)), int(math.Round(a.y+t*dy)))
		if width == 1 {
			p.image.SetRGBA(point.X, point.Y, ink)
		} else {
			p.dot(point, width/2, ink)
		}
	}
}
