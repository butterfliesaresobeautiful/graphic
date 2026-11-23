// Package bezier evaluates and subdivides Bézier curves and tensor-product surfaces.
package bezier

import (
	"fmt"
	"math"
)

const (
	MaxPoints       = 128
	MaxGridSide     = 16
	MaxDivisions    = 64
	CoordinateLimit = 10000
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (p Point) Add(q Point) Point {
	return Point{p.X + q.X, p.Y + q.Y, p.Z + q.Z}
}

func (p Point) Scale(k float64) Point {
	return Point{k * p.X, k * p.Y, k * p.Z}
}

func lerp(a, b Point, t float64) Point {
	return a.Scale(1 - t).Add(b.Scale(t))
}

func (p Point) Valid() bool {
	return math.Abs(p.X) <= CoordinateLimit &&
		math.Abs(p.Y) <= CoordinateLimit && math.Abs(p.Z) <= CoordinateLimit
}

func ValidParameter(t float64) bool {
	return t >= 0 && t <= 1
}

type Construction struct {
	Levels [][]Point  `json:"levels"`
	Split  [2][]Point `json:"split"`
}

// Construct builds the de Casteljau triangle. Its two sides are the
// control polygons of the two subcurves, in their original orientation.
func Construct(control []Point, t float64) Construction {
	n := len(control)
	result := Construction{
		Levels: make([][]Point, n),
		Split:  [2][]Point{make([]Point, n), make([]Point, n)},
	}
	result.Levels[0] = control
	for level := 0; level < n; level++ {
		if level > 0 {
			prev := result.Levels[level-1]
			row := make([]Point, len(prev)-1)
			for i := range row {
				row[i] = lerp(prev[i], prev[i+1], t)
			}
			result.Levels[level] = row
		}
		row := result.Levels[level]
		result.Split[0][level] = row[0]
		result.Split[1][n-1-level] = row[len(row)-1]
	}
	return result
}

func (c Construction) At() Point {
	return c.Levels[len(c.Levels)-1][0]
}

type Curve struct {
	Segments [][]Point
	Samples  []Point
}

// SampleCurve partitions the list into segments with a shared endpoint.
// Degree elevation preserves the shape of an incomplete final segment.
func SampleCurve(control []Point, degree, divisions int) (Curve, error) {
	if degree != 2 && degree != 3 {
		return Curve{}, fmt.Errorf("степень кривой должна быть 2 или 3")
	}
	if len(control) < 2 || len(control) > MaxPoints {
		return Curve{}, fmt.Errorf("нужно от 2 до %d вершин", MaxPoints)
	}
	if divisions < 1 || divisions > MaxDivisions {
		return Curve{}, fmt.Errorf("число разбиений должно быть от 1 до %d", MaxDivisions)
	}
	for _, point := range control {
		if !point.Valid() {
			return Curve{}, fmt.Errorf("координаты должны быть в пределах ±%d", CoordinateLimit)
		}
	}
	result := Curve{Samples: []Point{control[0]}}
	for start := 0; start < len(control)-1; start += degree {
		end := min(start+degree+1, len(control))
		segment := control[start:end]
		for len(segment) < degree+1 {
			segment = elevate(segment)
		}
		result.Segments = append(result.Segments, segment)
		for k := 1; k <= divisions; k++ {
			var work [4]Point
			copy(work[:], segment)
			t := float64(k) / float64(divisions)
			for count := degree; count > 0; count-- {
				for i := 0; i < count; i++ {
					work[i] = lerp(work[i], work[i+1], t)
				}
			}
			result.Samples = append(result.Samples, work[0])
		}
	}
	return result, nil
}

func elevate(control []Point) []Point {
	n := len(control)
	out := make([]Point, n+1)
	out[0], out[n] = control[0], control[n-1]
	for i := 1; i < n; i++ {
		out[i] = lerp(control[i], control[i-1], float64(i)/float64(n))
	}
	return out
}
