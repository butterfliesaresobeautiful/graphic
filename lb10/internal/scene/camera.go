// Package scene draws the construction and the sampled surface in software.
package scene

import (
	"image"
	"math"

	"graphiclabs/lb10/internal/bezier"
)

type Camera struct {
	Center     bezier.Point
	RadiusX    float64
	RadiusY    float64
	Zoom       float64
	Yaw, Pitch float64
	Spatial    bool
}

func Fit(points []bezier.Point, spatial bool) Camera {
	lo, hi := points[0], points[0]
	for _, p := range points {
		lo = bezier.Point{X: min(lo.X, p.X), Y: min(lo.Y, p.Y), Z: min(lo.Z, p.Z)}
		hi = bezier.Point{X: max(hi.X, p.X), Y: max(hi.Y, p.Y), Z: max(hi.Z, p.Z)}
	}
	half := hi.Add(lo.Scale(-1)).Scale(0.5)
	view := Camera{
		Center: lo.Add(hi).Scale(0.5), RadiusX: max(1, half.X),
		RadiusY: max(1, half.Y), Zoom: 1,
		Yaw: -0.49, Pitch: 0.55, Spatial: spatial,
	}
	if spatial {
		radius := math.Sqrt(half.X*half.X + half.Y*half.Y + half.Z*half.Z)
		view.RadiusX, view.RadiusY = max(1, radius), max(1, radius)
	}
	return view
}

type Projection struct {
	camera      Camera
	x, y, scale float64
	sy, cy      float64
	sp, cp      float64
}

type pixel struct {
	x, y, depth float64
}

func (c Camera) Projection(box image.Rectangle) Projection {
	sy, cy := math.Sincos(c.Yaw)
	sp, cp := math.Sincos(c.Pitch)
	return Projection{
		camera: c,
		x:      float64(box.Min.X+box.Max.X) * 0.5,
		y:      float64(box.Min.Y+box.Max.Y) * 0.5,
		scale:  0.43 * c.Zoom * min(float64(box.Dx())/c.RadiusX, float64(box.Dy())/c.RadiusY),
		sy:     sy, cy: cy, sp: sp, cp: cp,
	}
}

func (p Projection) mapPoint(point bezier.Point) pixel {
	x, y := point.X-p.camera.Center.X, point.Y-p.camera.Center.Y
	if !p.camera.Spatial {
		return pixel{p.x + x*p.scale, p.y - y*p.scale, 0}
	}
	z := point.Z - p.camera.Center.Z
	xx, yy := p.cy*x-p.sy*y, p.sy*x+p.cy*y
	return pixel{
		p.x + p.scale*xx,
		p.y - p.scale*(p.sp*yy+p.cp*z),
		p.scale * (-p.cp*yy + p.sp*z),
	}
}

func (p Projection) Screen(point bezier.Point) image.Point {
	s := p.mapPoint(point)
	return image.Pt(int(math.Round(s.x)), int(math.Round(s.y)))
}

func (p Projection) Plane(point image.Point) bezier.Point {
	return bezier.Point{
		X: p.camera.Center.X + (float64(point.X)-p.x)/p.scale,
		Y: p.camera.Center.Y - (float64(point.Y)-p.y)/p.scale,
	}
}
