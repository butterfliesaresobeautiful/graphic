package bezier

import (
	"fmt"
	"math"
)

// Grid is indexed by v (row), then u (column).
type Grid [][]Point

func (g Grid) Points() []Point {
	points := make([]Point, 0, len(g)*len(g[0]))
	for _, row := range g {
		points = append(points, row...)
	}
	return points
}

type Surface struct {
	Samples Grid
	Patches [4]Grid // Low v/low u, low v/high u, high v/low u, high v/high u.
	At      Point
}

func SampleSurface(grid Grid, u, v float64, nu, nv int) (Surface, error) {
	if len(grid) < 2 || len(grid) > MaxGridSide ||
		len(grid[0]) < 2 || len(grid[0]) > MaxGridSide {
		return Surface{}, fmt.Errorf("сетка должна иметь от 2 до %d точек по каждой оси", MaxGridSide)
	}
	for _, row := range grid {
		if len(row) != len(grid[0]) {
			return Surface{}, fmt.Errorf("строки контрольной сетки должны быть одной длины")
		}
		for _, point := range row {
			if !point.Valid() {
				return Surface{}, fmt.Errorf("координаты должны быть в пределах ±%d", CoordinateLimit)
			}
		}
	}
	if !ValidParameter(u) || !ValidParameter(v) {
		return Surface{}, fmt.Errorf("u и v должны принадлежать [0, 1]")
	}
	if nu < 1 || nv < 1 || nu > MaxDivisions || nv > MaxDivisions {
		return Surface{}, fmt.Errorf("число разбиений должно быть от 1 до %d", MaxDivisions)
	}
	uWeights := make([][]float64, nu+1)
	for i := range uWeights {
		uWeights[i] = bernstein(len(grid[0]), float64(i)/float64(nu))
	}
	result := Surface{Samples: makeGrid(nu+1, nv+1)}
	for j := range result.Samples {
		wv := bernstein(len(grid), float64(j)/float64(nv))
		for i, wu := range uWeights {
			result.Samples[j][i] = blend(grid, wu, wv)
		}
	}
	result.At = blend(grid, bernstein(len(grid[0]), u), bernstein(len(grid), v))
	result.Patches = splitGrid(grid, u, v)
	return result, nil
}

func makeGrid(cols, rows int) Grid {
	grid := make(Grid, rows)
	for j := range grid {
		grid[j] = make([]Point, cols)
	}
	return grid
}

// ResizeGrid keeps the intersection with the old grid. New rows and
// columns start as an extension of the original boundary.
func ResizeGrid(grid Grid, cols, rows int) Grid {
	out := makeGrid(cols, rows)
	for j := range out {
		for i := range out[j] {
			row := grid[min(j, len(grid)-1)]
			p := row[min(i, len(row)-1)]
			p.X += float64(max(0, i-len(row)+1))
			p.Y += float64(max(0, j-len(grid)+1))
			p.X = min(p.X, CoordinateLimit)
			p.Y = min(p.Y, CoordinateLimit)
			out[j][i] = p
		}
	}
	return out
}

func bernstein(count int, t float64) []float64 {
	n := count - 1
	w := make([]float64, count)
	binomial := 1.0
	for i := range w {
		w[i] = binomial * math.Pow(t, float64(i)) * math.Pow(1-t, float64(n-i))
		binomial *= float64(n-i) / float64(i+1)
	}
	return w
}

func blend(grid Grid, wu, wv []float64) Point {
	var sum Point
	for j, row := range grid {
		for i, p := range row {
			sum = sum.Add(p.Scale(wu[i] * wv[j]))
		}
	}
	return sum
}

func splitGrid(grid Grid, u, v float64) [4]Grid {
	rows, cols := len(grid), len(grid[0])
	var halves [2]Grid
	halves[0], halves[1] = make(Grid, rows), make(Grid, rows)
	for j, row := range grid {
		split := Construct(row, u).Split
		halves[0][j], halves[1][j] = split[0], split[1]
	}
	var patches [4]Grid
	for i := range patches {
		patches[i] = makeGrid(cols, rows)
	}
	column := make([]Point, rows)
	for hu, half := range halves {
		for i := 0; i < cols; i++ {
			for j := range column {
				column[j] = half[j][i]
			}
			split := Construct(column, v).Split
			for hv, side := range split {
				for j, point := range side {
					patches[2*hv+hu][j][i] = point
				}
			}
		}
	}
	return patches
}
