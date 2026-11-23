package desktop

import (
	"fmt"
	"image"
	"math"

	"graphiclabs/lb10/internal/bezier"
	"graphiclabs/lb10/internal/scene"
)

type mode int

const (
	quadratic mode = iota
	cubic
	surface
)

var modeNames = [...]string{"quadratic", "cubic", "surface"}

type curveState struct {
	control      []bezier.Point
	degree       int
	t            float64
	divisions    int
	segment      int
	stage        int
	selected     int
	curve        bezier.Curve
	construction bezier.Construction
	camera       scene.Camera
}

func (c *curveState) rebuild() error {
	curve, err := bezier.SampleCurve(c.control, c.degree, c.divisions)
	if err != nil {
		return err
	}
	c.curve = curve
	c.segment = min(c.segment, len(curve.Segments)-1)
	c.selected = min(c.selected, len(c.control)-1)
	c.inspect()
	return nil
}

func (c *curveState) inspect() {
	c.construction = bezier.Construct(c.curve.Segments[c.segment], c.t)
}

type surfaceState struct {
	control  bezier.Grid
	u, v     float64
	nu, nv   int
	selected int
	patch    bezier.Surface
	camera   scene.Camera
}

func (s *surfaceState) rebuild() error {
	patch, err := bezier.SampleSurface(s.control, s.u, s.v, s.nu, s.nv)
	if err != nil {
		return err
	}
	s.patch = patch
	s.selected = min(s.selected, len(s.control)*len(s.control[0])-1)
	return nil
}

func (a *app) reset() error {
	if a.mode != surface {
		degree := int(a.mode) + 2
		points, t, n := bezier.DemoCurve(degree)
		state := &a.curves[a.mode]
		*state = curveState{
			control: points, degree: degree, t: t, divisions: n,
			stage: degree, camera: scene.Fit(points, false),
		}
		return state.rebuild()
	}
	grid := bezier.DemoGrid()
	a.surface = surfaceState{
		control: grid, u: bezier.DemoU, v: bezier.DemoV,
		nu: bezier.DemoNU, nv: bezier.DemoNV,
		camera: scene.Fit(grid.Points(), true),
	}
	return a.surface.rebuild()
}

func (a *app) camera() *scene.Camera {
	if a.mode == surface {
		return &a.surface.camera
	}
	return &a.curves[a.mode].camera
}

func (a *app) controls() []bezier.Point {
	if a.mode == surface {
		return a.surface.control.Points()
	}
	return a.curves[a.mode].control
}

func (a *app) selected() *int {
	if a.mode == surface {
		return &a.surface.selected
	}
	return &a.curves[a.mode].selected
}

func (a *app) vertex() *bezier.Point {
	if a.mode == surface {
		s := &a.surface
		cols := len(s.control[0])
		return &s.control[s.selected/cols][s.selected%cols]
	}
	c := &a.curves[a.mode]
	return &c.control[c.selected]
}

func (a *app) fit() {
	view := a.camera()
	fitted := scene.Fit(a.controls(), a.mode == surface)
	fitted.Yaw, fitted.Pitch = view.Yaw, view.Pitch
	*view = fitted
}

func (a *app) selectVertex(index int) {
	*a.selected() = index
	if a.mode != surface {
		c := &a.curves[a.mode]
		c.segment = min(index/c.degree, len(c.curve.Segments)-1)
		c.inspect()
	}
}

func (a *app) moveVertex(increment int) {
	index := *a.selected()
	a.selectVertex(max(0, min(len(a.controls())-1, index+increment)))
}

func (a *app) deleteVertex() {
	c := &a.curves[a.mode]
	if len(c.control) <= 2 {
		a.status = "Для кривой нужны хотя бы две контрольные вершины."
		return
	}
	c.control = append(c.control[:c.selected], c.control[c.selected+1:]...)
	a.result(c.rebuild())
}

func (a *app) result(err error) {
	a.status = ""
	if err != nil {
		a.status = "Ошибка. " + err.Error()
	}
}

func (a *app) pick(in image.Point) int {
	if !a.original {
		return -1
	}
	projection := a.camera().Projection(plotBox)
	index, closest := -1, 13*13
	for i, p := range a.controls() {
		position := projection.Screen(p)
		if !position.In(plotBox) {
			continue
		}
		dx, dy := position.X-in.X, position.Y-in.Y
		distance := dx*dx + dy*dy
		if distance < closest {
			index, closest = i, distance
		}
	}
	return index
}

func (a *app) addCurvePoint(in image.Point) {
	c := &a.curves[a.mode]
	p := c.camera.Projection(plotBox).Plane(in)
	if !p.Valid() {
		a.status = fmt.Sprintf("Допустимый диапазон координат ±%d.", bezier.CoordinateLimit)
		return
	}
	if len(c.control) == bezier.MaxPoints {
		a.status = fmt.Sprintf("Достигнут предел в %d контрольных точек.", bezier.MaxPoints)
		return
	}
	c.control = append(c.control, p)
	a.result(c.rebuild())
	a.selectVertex(len(c.control) - 1)
}

func (a *app) dragTo(in image.Point) {
	if a.rotating {
		delta := in.Sub(a.lastMouse)
		a.lastMouse = in
		view := a.camera()
		view.Yaw = math.Remainder(view.Yaw+float64(delta.X)*0.01, 2*math.Pi)
		view.Pitch = min(1.55, max(-1.55, view.Pitch+float64(delta.Y)*0.01))
	}
	if a.dragging && in.In(plotBox) {
		c := &a.curves[a.mode]
		p := c.camera.Projection(plotBox).Plane(in)
		if p.Valid() {
			c.control[c.selected] = p
			a.result(c.rebuild())
		}
	}
}
