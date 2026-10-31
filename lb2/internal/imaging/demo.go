package imaging

import (
	"image"
	"image/color"
	"math"
)

// Demo is generated here in Go. Its lower-left flower has varied RGB values,
// so an exact pixel transfer is visibly distinguishable from a solid fill.
func Demo() *image.RGBA {
	m, _ := Canvas(800, 600)
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			r, g, b := uint8(35+150*x/799), uint8(75+140*y/599), uint8(180-80*y/599)
			if x/50%2 == y/50%2 {
				r = uint8(min(255, int(r)+28))
				g = uint8(min(255, int(g)+18))
			}
			if x%50 == 0 || y%50 == 0 {
				r, g, b = 210, 223, 235
			}
			if x == 400 || y == 300 {
				r, g, b = 45, 63, 85
			}
			m.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}
	// Sun and a small house make orientation and different source regions clear.
	for y := 60; y < 180; y++ {
		for x := 600; x < 720; x++ {
			if (x-660)*(x-660)+(y-120)*(y-120) <= 55*55 {
				m.SetRGBA(x, y, color.RGBA{255, 211, 60, 255})
			}
		}
	}
	for y := 120; y < 370; y++ {
		for x := 250; x < 500; x++ {
			if y >= 210 && x >= 290 && x < 460 {
				m.SetRGBA(x, y, color.RGBA{244, 176, 72, 255})
			}
			if y < 210 && y >= 120+abs(x-375)*9/10 {
				m.SetRGBA(x, y, color.RGBA{174, 49, 72, 255})
			}
			if x >= 355 && x < 395 && y >= 285 {
				m.SetRGBA(x, y, color.RGBA{43, 70, 102, 255})
			}
		}
	}
	for y := 459; y < 600; y++ {
		for x := 0; x <= 140; x++ {
			dx, dy := float64(x-70), float64(y-529)
			r := math.Hypot(dx, dy)
			t := math.Atan2(dy, dx)
			if r <= 70 {
				col := color.RGBA{35, uint8(120 + r), uint8(110 + r), 255}
				if r < 48+17*math.Cos(8*t) {
					col = color.RGBA{uint8(180 + 65*math.Cos(t)*math.Cos(t)), uint8(45 + r*2), uint8(65 + 90*math.Sin(t)*math.Sin(t)), 255}
				}
				if r < 16 {
					col = color.RGBA{255, 215, uint8(40 + 5*r), 255}
				}
				m.SetRGBA(x, y, col)
			}
		}
	}
	return m
}
