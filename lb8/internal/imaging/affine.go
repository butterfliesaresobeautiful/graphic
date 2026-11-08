package imaging

import (
	"errors"
	"image"
	"math"
)

// Config applies scale, then simultaneous two-axis shear, then rotation.
// Positive angles rotate clockwise because screen Y increases downward.
type Config struct{ ScaleX, ScaleY, Angle, ShearX, ShearY float64 }

func Default() Config { return Config{1.25, .9, 28, .25, -.1} }
func (c Config) Validate() error {
	for _, v := range []float64{c.ScaleX, c.ScaleY, c.Angle, c.ShearX, c.ShearY} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("параметры должны быть конечными числами")
		}
	}
	if c.ScaleX < .05 || c.ScaleX > 8 || c.ScaleY < .05 || c.ScaleY > 8 {
		return errors.New("масштабы: от 0.05 до 8")
	}
	if math.Abs(c.Angle) > 360 || math.Abs(c.ShearX) > 3 || math.Abs(c.ShearY) > 3 {
		return errors.New("угол: −360…360; скосы: −3…3")
	}
	if math.Abs(1-c.ShearX*c.ShearY) < .01 {
		return errors.New("вырожденный или почти вырожденный скос: |1 − hx·hy| должно быть не меньше 0.01")
	}
	return nil
}

type Matrix struct{ A, B, C, D float64 }

func (m Matrix) Apply(x, y float64) (float64, float64) { return m.A*x + m.B*y, m.C*x + m.D*y }
func (m Matrix) Inverse() Matrix {
	d := m.A*m.D - m.B*m.C
	return Matrix{m.D / d, -m.B / d, -m.C / d, m.A / d}
}
func (c Config) Matrix() Matrix {
	s, t := math.Sincos(c.Angle * math.Pi / 180)
	return Matrix{(t - s*c.ShearY) * c.ScaleX, (t*c.ShearX - s) * c.ScaleY, (s + t*c.ShearY) * c.ScaleX, (s*c.ShearX + t) * c.ScaleY}
}

type Geometry struct {
	Width, Height    int
	OriginX, OriginY float64
	Forward, Inverse Matrix
}

func Plan(w, h int, c Config) (Geometry, error) {
	var g Geometry
	if err := Dimensions(w, h); err != nil {
		return g, err
	}
	if err := c.Validate(); err != nil {
		return g, err
	}
	g.Forward = c.Matrix()
	g.Inverse = g.Forward.Inverse()
	minx, miny := math.Inf(1), math.Inf(1)
	maxx, maxy := math.Inf(-1), math.Inf(-1)
	for _, p := range [][2]float64{{0, 0}, {float64(w), 0}, {0, float64(h)}, {float64(w), float64(h)}} {
		x, y := g.Forward.Apply(p[0]-float64(w)/2, p[1]-float64(h)/2)
		minx, miny = math.Min(minx, x), math.Min(miny, y)
		maxx, maxy = math.Max(maxx, x), math.Max(maxy, y)
	}
	g.OriginX, g.OriginY = minx, miny
	// Suppress floating-point noise for exact quarter turns and integer sizes.
	g.Width, g.Height = int(math.Ceil(maxx-minx-1e-9)), int(math.Ceil(maxy-miny-1e-9))
	return g, Dimensions(g.Width, g.Height)
}
func (g Geometry) SourcePoint(x, y, w, h int) (float64, float64) {
	u, v := g.Inverse.Apply(float64(x)+.5+g.OriginX, float64(y)+.5+g.OriginY)
	return u + float64(w)/2 - .5, v + float64(h)/2 - .5
}
func covered(x, y float64, w, h int) bool {
	return x >= -.5 && y >= -.5 && x < float64(w)-.5 && y < float64(h)-.5
}

type Preset struct {
	Name, Title string
	Config      Config
}

func Presets() []Preset {
	return []Preset{
		{"identity", "Без преобразования", Config{1, 1, 0, 0, 0}},
		{"scale-up", "Увеличение 1.7 × 1.4", Config{1.7, 1.4, 0, 0, 0}},
		{"scale-down", "Уменьшение 0.55 × 0.65", Config{.55, .65, 0, 0, 0}},
		{"rotate-positive", "Поворот +28°", Config{1, 1, 28, 0, 0}},
		{"rotate-negative", "Поворот −17°", Config{1, 1, -17, 0, 0}},
		{"shear-positive", "Скос hx=0.45, hy=0.15", Config{1, 1, 0, .45, .15}},
		{"shear-negative", "Скос hx=−0.35, hy=0.10", Config{1, 1, 0, -.35, .1}},
		{"combined", "Масштаб → скос → поворот", Default()},
	}
}
func sourceSize(src *image.RGBA) (int, int, error) {
	if src == nil {
		return 0, 0, errors.New("сначала откройте изображение")
	}
	w, h := src.Rect.Dx(), src.Rect.Dy()
	return w, h, Dimensions(w, h)
}
