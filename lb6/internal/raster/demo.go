package raster

import (
	"image"
	"image/color"
	"math"
)

// Demo creates an asymmetric RGB calibration scene entirely in Go.
// Fine detail on the left makes the literal nonlinear stretching visible.
func Demo() *image.RGBA {
	m, _ := New(640, 420)
	for y := 0; y < 420; y++ {
		for x := 0; x < 640; x++ {
			c := color.RGBA{uint8(40 + x/4), uint8(80 + y/3), uint8(190 - y/5), 255}
			if x%40 == 0 || y%40 == 0 {
				c = color.RGBA{220, 230, 244, 255}
			}
			if x < 64 {
				palette := []color.RGBA{{237, 70, 80, 255}, {255, 192, 60, 255}, {44, 180, 120, 255}, {35, 109, 220, 255}}
				c = palette[(x/8+y/30)%len(palette)]
				if x%8 == 0 || y%30 == 0 {
					c = color.RGBA{25, 35, 55, 255}
				}
			}
			// House, roof and asymmetric windows.
			if x >= 130 && x <= 340 && y >= 190 && y <= 350 {
				c = color.RGBA{242, 178, 92, 255}
			}
			if y >= 90 && y < 190 && math.Abs(float64(x-235)) < float64(y-90)*1.25 {
				c = color.RGBA{172, 54, 72, 255}
			}
			if x > 155 && x < 205 && y > 220 && y < 270 {
				c = color.RGBA{99, 211, 238, 255}
			}
			if x > 265 && x < 310 && y > 250 && y < 350 {
				c = color.RGBA{68, 75, 102, 255}
			}
			if (x-505)*(x-505)+(y-110)*(y-110) < 62*62 {
				c = color.RGBA{255, 210, 62, 255}
			}
			if (x-510)*(x-510)+(y-290)*(y-290) < 55*55 {
				c = color.RGBA{65, 193, 133, 255}
			}
			if x > 490 && x < 505 && y > 260 && y < 380 {
				c = color.RGBA{92, 60, 53, 255}
			}
			// White arrow to the right, visibly reversed after reflection.
			if y >= 35 && y <= 48 && x >= 120 && x < 270 || x >= 260 && x <= 290 && math.Abs(float64(y-42)) <= float64(290-x) {
				c = color.RGBA{255, 255, 255, 255}
			}
			m.SetRGBA(x, y, c)
		}
	}
	return m
}
