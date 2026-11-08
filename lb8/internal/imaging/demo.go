package imaging

import (
	"image"
	"image/color"
	"math"
)

// Demo combines smooth gradients with discontinuities and high frequencies.
func Demo() *image.RGBA {
	m, _ := Canvas(480, 360)
	for y := 0; y < 360; y++ {
		for x := 0; x < 480; x++ {
			p := color.RGBA{uint8(30 + x*200/479), uint8(30 + y*200/359), 150, 255}
			if x < 240 && y < 180 {
				v := uint8(25)
				if (x/6+y/6)%2 == 0 {
					v = 235
				}
				p = color.RGBA{v, v, v, 255}
			}
			if x >= 240 && y < 180 {
				if math.Abs(float64(y)-.55*float64(x-240)-15) < 1.5 || math.Abs(float64(y)+.4*float64(x-240)-140) < 2 {
					p = color.RGBA{10, 10, 10, 255}
				}
			}
			if x < 240 && y >= 180 {
				v := uint8(127 + 100*math.Sin(float64(x)*.7+float64(y)*.12))
				p = color.RGBA{v, uint8(255 - int(v)), 100, 255}
			}
			dx, dy := x-355, y-265
			if dx*dx+dy*dy < 60*60 {
				p = color.RGBA{245, 160, 20, 255}
				if dx*dx+dy*dy < 30*30 {
					p = color.RGBA{30, 90, 210, 255}
				}
			}
			if x < 3 || y < 3 || x >= 477 || y >= 357 {
				p = color.RGBA{15, 30, 50, 255}
			}
			m.SetRGBA(x, y, p)
		}
	}
	return m
}
