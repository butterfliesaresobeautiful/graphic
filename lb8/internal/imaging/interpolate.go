package imaging

import (
	"image"
	"math"
)

type Method int

const (
	Nearest Method = iota
	Bilinear
	Bicubic
)

var Methods = [3]string{"nearest", "bilinear", "bicubic"}

// Lagrange weights at nodes -1, 0, 1, 2. The tensor product is degree
// three in each coordinate (total degree at most six), not a 1D sextic.
func Weights(t float64) [4]float64 {
	return [4]float64{
		-t * (t - 1) * (t - 2) / 6,
		(t + 1) * (t - 1) * (t - 2) / 2,
		-(t + 1) * t * (t - 2) / 2,
		(t + 1) * t * (t - 1) / 6,
	}
}
func pixel(src *image.RGBA, x, y int) [3]float64 {
	x = max(src.Rect.Min.X, min(src.Rect.Max.X-1, x+src.Rect.Min.X))
	y = max(src.Rect.Min.Y, min(src.Rect.Max.Y-1, y+src.Rect.Min.Y))
	c := src.RGBAAt(x, y)
	return [3]float64{float64(c.R), float64(c.G), float64(c.B)}
}

// Sample uses edge replication; callers separately test geometric coverage.
func Sample(src *image.RGBA, x, y float64, method Method) [3]float64 {
	if method == Nearest {
		return pixel(src, int(math.Floor(x+.5)), int(math.Floor(y+.5)))
	}
	l, k := int(math.Floor(x)), int(math.Floor(y))
	a, b := x-float64(l), y-float64(k)
	var out [3]float64
	if method == Bilinear {
		for j := 0; j < 2; j++ {
			for i := 0; i < 2; i++ {
				wx, wy := a, b
				if i == 0 {
					wx = 1 - a
				}
				if j == 0 {
					wy = 1 - b
				}
				p := pixel(src, l+i, k+j)
				for c := range out {
					out[c] += wx * wy * p[c]
				}
			}
		}
	} else {
		wx, wy := Weights(a), Weights(b)
		for j := 0; j < 4; j++ {
			for i := 0; i < 4; i++ {
				p := pixel(src, l+i-1, k+j-1)
				for c := range out {
					out[c] += wx[i] * wy[j] * p[c]
				}
			}
		}
	}
	return out
}
