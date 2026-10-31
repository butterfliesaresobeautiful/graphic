package native

import (
	"graphiclabs/lb4/internal/raster"
	"math"
	"os"
	"testing"
)

func TestRealX11CircleAndReuleaux(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("requires X11/Xvfb")
	}
	for _, circle := range []bool{true, false} {
		c := raster.Default()
		c.Circle = circle
		m, err := Rasterize(c)
		if err != nil {
			t.Fatal(err)
		}
		if m.Count() < 100 {
			t.Fatal("empty native contour")
		}
		for i, on := range m.Bits {
			if !on {
				continue
			}
			x, y := float64(i%m.W), float64(i/m.W)
			nearest := math.Inf(1)
			for _, a := range c.Arcs() {
				distance := math.Hypot(x-a.Center.X, y-a.Center.Y) - float64(a.Radius)
				nearest = math.Min(nearest, math.Abs(distance))
				if !circle && distance > 1.5 {
					t.Fatal("wrong native arc orientation")
				}
			}
			if nearest > 1.5 {
				t.Fatal("native radial error", nearest)
			}
		}
	}
}
