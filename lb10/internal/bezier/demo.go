package bezier

const (
	DemoU  = 0.42
	DemoV  = 0.60
	DemoNU = 16
	DemoNV = 12
)

func DemoCurve(degree int) (control []Point, t float64, divisions int) {
	if degree == 2 {
		return []Point{
			{X: -5, Y: -2}, {X: -4, Y: 4}, {X: -1, Y: 1},
			{X: 1, Y: -4}, {X: 5, Y: 1},
		}, 0.4, 10
	}
	return []Point{
		{X: -5, Y: -2}, {X: -4, Y: 3}, {X: -2, Y: 3}, {X: 0, Y: -1},
		{X: 2, Y: -4}, {X: 3, Y: -3}, {X: 4, Y: 1}, {X: 5.5, Y: 2.4},
	}, 0.5, 16
}

func DemoGrid() Grid {
	z := [3][4]float64{
		{-1.1, 0.5, 2.2, 0.4},
		{0, 5, 3.8, -1.2},
		{0.9, 2.7, -2.2, -0.3},
	}
	out := makeGrid(4, 3)
	for j := range out {
		for i := range out[j] {
			out[j][i] = Point{X: -4 + 8*float64(i)/3, Y: -3 + 3*float64(j), Z: z[j][i]}
		}
	}
	return out
}
