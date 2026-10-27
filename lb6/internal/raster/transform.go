package raster

import (
	"errors"
	"fmt"
	"image"
	"math"
)

// Matrix maps source pixel centres to destination pixel centres.
// [x2;y2] = [A B;D E] [x1;y1] + [C;F].
type Matrix struct{ A, B, C, D, E, F float64 }

func (m Matrix) Point(x, y float64) (float64, float64) {
	return m.A*x + m.B*y + m.C, m.D*x + m.E*y + m.F
}

func (m Matrix) Inverse() (Matrix, error) {
	for _, v := range []float64{m.A, m.B, m.C, m.D, m.E, m.F} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return Matrix{}, errors.New("матрица содержит нечисловые значения")
		}
	}
	det := m.A*m.E - m.B*m.D
	if math.Abs(det) < 1e-12 {
		return Matrix{}, errors.New("вырожденная матрица: обратного преобразования нет")
	}
	return Matrix{m.E / det, -m.B / det, (m.B*m.F - m.E*m.C) / det,
		-m.D / det, m.A / det, (m.C*m.D - m.A*m.F) / det}, nil
}

// Variant8 first rotates clockwise in screen coordinates, then reflects
// about the chosen central axis. No extra scaling or translation is added.
func Variant8(w, h int, angle float64, axis string) (Matrix, error) {
	if err := Dimensions(w, h); err != nil {
		return Matrix{}, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) || angle < -360 || angle > 360 {
		return Matrix{}, errors.New("угол должен быть конечным числом от −360 до 360 градусов")
	}
	sx, sy := 1.0, 1.0
	switch axis {
	case "vertical":
		sx = -1
	case "horizontal":
		sy = -1
	default:
		return Matrix{}, errors.New("ось отражения: vertical или horizontal")
	}
	c, s := math.Cos(angle*math.Pi/180), math.Sin(angle*math.Pi/180)
	m := Matrix{A: sx * c, B: -sx * s, D: sy * s, E: sy * c}
	cx, cy := float64(w-1)/2, float64(h-1)/2
	m.C = cx - m.A*cx - m.B*cy
	m.F = cy - m.D*cx - m.E*cy
	return m, nil
}

type Stats struct {
	Mapped  int `json:"mapped"`
	Outside int `json:"outside"`
}

// sample visits every destination pixel. Rounding is nearest-neighbour.
// Out-of-range samples keep the white background, with no out-of-bounds reads.
func sample(src *image.RGBA, back func(float64, float64) (float64, float64)) (*image.RGBA, Stats, error) {
	var st Stats
	if src == nil || src.Bounds().Empty() {
		return nil, st, errors.New("сначала загрузите изображение")
	}
	b := src.Bounds()
	dst, err := New(b.Dx(), b.Dy())
	if err != nil {
		return nil, st, err
	}
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			u, v := back(float64(x), float64(y))
			u, v = math.Round(u), math.Round(v)
			if math.IsNaN(u) || math.IsNaN(v) || u < 0 || v < 0 || u >= float64(b.Dx()) || v >= float64(b.Dy()) {
				st.Outside++
				continue
			}
			dst.SetRGBA(x, y, src.RGBAAt(int(u)+b.Min.X, int(v)+b.Min.Y))
			st.Mapped++
		}
	}
	return dst, st, nil
}

// Warp implements a forward geometric transform using inverse raster mapping.
func Warp(src *image.RGBA, forward Matrix) (*image.RGBA, Stats, error) {
	back, err := forward.Inverse()
	if err != nil {
		return nil, Stats{}, err
	}
	return sample(src, back.Point)
}

// Power53 is the real-valued odd-root interpretation, also defined for x<0.
func Power53(x float64) float64 { return math.Copysign(math.Pow(math.Abs(x), 5.0/3.0), x) }
func Power35(x float64) float64 { return math.Copysign(math.Pow(math.Abs(x), 3.0/5.0), x) }

// Nonlinear uses the exact variant formula i=x'^(5/3), j=y'.
// Pixel coordinates are NOT normalised. The canvas keeps its original size.
func Nonlinear(src *image.RGBA) (*image.RGBA, Stats, error) {
	return sample(src, func(i, j float64) (float64, float64) { return Power35(i), j })
}

type Results struct {
	Affine, Restored, Nonlinear *image.RGBA
	Matrix                      Matrix
	Stats                       [3]Stats
}

func Process(src *image.RGBA, angle float64, axis string) (Results, error) {
	var r Results
	if src == nil {
		return r, errors.New("не задан источник")
	}
	m, err := Variant8(src.Bounds().Dx(), src.Bounds().Dy(), angle, axis)
	if err != nil {
		return r, err
	}
	r.Matrix = m
	r.Affine, r.Stats[0], err = Warp(src, m)
	if err != nil {
		return r, err
	}
	inv, err := m.Inverse()
	if err != nil {
		return r, err
	}
	// Restoration reads the transformed raster, never the original image.
	r.Restored, r.Stats[1], err = Warp(r.Affine, inv)
	if err != nil {
		return r, err
	}
	r.Nonlinear, r.Stats[2], err = Nonlinear(src)
	return r, err
}

func MAE(a, b *image.RGBA) (float64, error) {
	if a == nil || b == nil || a.Bounds() != b.Bounds() {
		return 0, fmt.Errorf("для сравнения нужны одинаковые размеры")
	}
	var sum float64
	for y := a.Rect.Min.Y; y < a.Rect.Max.Y; y++ {
		for x := a.Rect.Min.X; x < a.Rect.Max.X; x++ {
			p, q := a.RGBAAt(x, y), b.RGBAAt(x, y)
			sum += math.Abs(float64(p.R)-float64(q.R)) + math.Abs(float64(p.G)-float64(q.G)) + math.Abs(float64(p.B)-float64(q.B))
		}
	}
	return sum / float64(a.Rect.Dx()*a.Rect.Dy()*3), nil
}
