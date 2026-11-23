package bezier

import (
	"math"
	"testing"
)

func closePoint(t *testing.T, got, want Point) {
	t.Helper()
	for _, d := range []float64{got.X - want.X, got.Y - want.Y, got.Z - want.Z} {
		if !(math.Abs(d) <= 1e-10) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}

func TestCurveStagesAndSubcurves(t *testing.T) {
	quadratic, _, _ := DemoCurve(2)
	construction := Construct(quadratic[:3], 0.4)
	closePoint(t, construction.Levels[1][0], Point{X: -4.6, Y: 0.4})
	closePoint(t, construction.Levels[1][1], Point{X: -2.8, Y: 2.8})
	closePoint(t, construction.At(), Point{X: -3.88, Y: 1.36})
	cubic, _, _ := DemoCurve(3)
	closePoint(t, Construct(cubic[:4], 0.5).At(), Point{X: -2.875, Y: 1.875})
	for _, control := range [][]Point{quadratic[:3], cubic[:4]} {
		for _, cut := range []float64{0, 0.4, 1} {
			split := Construct(control, cut).Split
			for _, q := range []float64{0, 0.23, 0.67, 1} {
				closePoint(t, Construct(split[0], q).At(), Construct(control, q*cut).At())
				closePoint(t, Construct(split[1], q).At(), Construct(control, cut+q*(1-cut)).At())
			}
		}
	}
}

func TestCurveChainAndDegreeElevation(t *testing.T) {
	control := []Point{
		{X: -5, Y: -2}, {X: -4, Y: 4}, {X: -1, Y: 1},
		{X: 1, Y: -4}, {X: 5, Y: 1}, {X: 7, Y: 4},
	}
	for _, degree := range []int{2, 3} {
		const divisions = 8
		curve, err := SampleCurve(control, degree, divisions)
		if err != nil {
			t.Fatal(err)
		}
		n := (len(control) - 2 + degree) / degree
		if len(curve.Segments) != n || len(curve.Samples) != n*divisions+1 {
			t.Fatalf("degree %d: %d segments and %d samples", degree, len(curve.Segments), len(curve.Samples))
		}
		for i, segment := range curve.Segments {
			if len(segment) != degree+1 {
				t.Fatalf("degree %d: segment %d has %d points", degree, i, len(segment))
			}
			closePoint(t, curve.Samples[i*divisions], control[i*degree])
		}
		closePoint(t, curve.Samples[n*divisions], control[len(control)-1])
		last := control[(n-1)*degree:]
		want := Construct(last, 0.5).At()
		closePoint(t, curve.Samples[(n-1)*divisions+divisions/2], want)
	}
}

func TestSurfaceBernsteinAndRefinement(t *testing.T) {
	grid := DemoGrid()
	const u, v = 0.42, 0.6
	result, err := SampleSurface(grid, u, v, 16, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Samples) != 13 || len(result.Samples[0]) != 17 {
		t.Fatal("incorrect sample grid dimensions")
	}
	// Compare Bernstein evaluation against nested de Casteljau.
	evaluate := func(g Grid, u, v float64) Point {
		column := make([]Point, len(g))
		for j, row := range g {
			column[j] = Construct(row, u).At()
		}
		return Construct(column, v).At()
	}
	closePoint(t, result.At, evaluate(grid, u, v))
	for j, row := range result.Samples {
		for i, p := range row {
			closePoint(t, p, evaluate(grid, float64(i)/16, float64(j)/12))
		}
	}
	for k, patch := range result.Patches {
		us, vs := []float64{0, u, 1}, []float64{0, v, 1}
		i, j := k%2, k/2
		for _, q := range []float64{0, 0.37, 1} {
			for _, r := range []float64{0, 0.73, 1} {
				want := evaluate(grid, us[i]+q*(us[i+1]-us[i]), vs[j]+r*(vs[j+1]-vs[j]))
				closePoint(t, evaluate(patch, q, r), want)
			}
		}
	}
	wide := ResizeGrid(grid, 5, 2)
	closePoint(t, wide[1][3], grid[1][3])
	resized, err := SampleSurface(wide, 0, 1, 1, 64)
	if err != nil {
		t.Fatal(err)
	}
	closePoint(t, resized.At, wide[1][0])
	closePoint(t, resized.Samples[64][1], wide[1][4])
}

func TestInputLimits(t *testing.T) {
	control := []Point{{}, {X: math.NaN()}}
	if _, err := SampleCurve(control, 2, 4); err == nil {
		t.Fatal("nonfinite curve coordinate accepted")
	}
	if _, err := SampleCurve([]Point{{}, {X: 1}}, 3, 0); err == nil {
		t.Fatal("zero divisions accepted")
	}
	if _, err := SampleSurface(DemoGrid(), math.NaN(), 0.5, 10, 10); err == nil {
		t.Fatal("nonfinite surface parameter accepted")
	}
	if _, err := SampleSurface(Grid{{{}, {}}, {{}}}, 0, 1, 10, 10); err == nil {
		t.Fatal("ragged control grid accepted")
	}
}
