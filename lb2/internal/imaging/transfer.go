package imaging

import (
	"errors"
	"fmt"
	"image"
	"image/color"
)

type Config struct {
	Width, Height, Radius, SX, SY, DX, DY int
	Auto                                  bool
}
type Stats struct{ Inside, Copied, Clipped int }

func Default() Config { return Config{Width: 1000, Height: 720, Radius: 70, Auto: true} }
func (c Config) Validate() error {
	if e := Dimensions(c.Width, c.Height); e != nil {
		return e
	}
	if c.Radius < 1 || c.Radius > 2048 {
		return errors.New("радиус должен быть от 1 до 2048")
	}
	for _, v := range []int{c.SX, c.SY, c.DX, c.DY} {
		if v < -8192 || v > 8192 {
			return errors.New("координаты должны быть от −8192 до 8192")
		}
	}
	return nil
}
func (c Config) Resolve(src image.Rectangle) (Config, error) {
	if e := c.Validate(); e != nil {
		return c, e
	}
	if src.Empty() {
		return c, errors.New("нет исходного изображения")
	}
	if c.Auto {
		limit := (min(src.Dx()/2, src.Dy()/2) - 1) / 2
		if c.Radius > limit {
			return c, fmt.Errorf("для круга в нижней левой четверти нужен радиус не более %d", max(0, limit))
		}
		if 2*c.Radius+1 > c.Width || 2*c.Radius+1 > c.Height {
			return c, errors.New("круг не помещается на новом холсте")
		}
		c.SX, c.SY = src.Min.X+c.Radius, src.Max.Y-1-c.Radius
		c.DX, c.DY = c.Width/2, c.Radius
	}
	return c, nil
}

// Transfer copies actual RGB values in two explicit loops. No rectangle-copy
// helper, interpolation, antialiasing, or outline alters the selected colours.
func Transfer(src, dst *image.RGBA, c Config) (Stats, error) {
	var s Stats
	if src == nil || dst == nil {
		return s, errors.New("загрузите источник и создайте холст")
	}
	if src == dst {
		return s, errors.New("источник и результат должны быть разными изображениями")
	}
	if e := c.Validate(); e != nil {
		return s, e
	}
	if dst.Rect != image.Rect(0, 0, c.Width, c.Height) {
		return s, errors.New("размер холста не соответствует параметрам")
	}
	for oy := -c.Radius; oy <= c.Radius; oy++ {
		for ox := -c.Radius; ox <= c.Radius; ox++ {
			if ox*ox+oy*oy > c.Radius*c.Radius {
				continue
			}
			s.Inside++
			from, to := image.Pt(c.SX+ox, c.SY+oy), image.Pt(c.DX+ox, c.DY+oy)
			if !from.In(src.Rect) || !to.In(dst.Rect) {
				s.Clipped++
				continue
			}
			p := src.RGBAAt(from.X, from.Y)
			dst.SetRGBA(to.X, to.Y, color.RGBA{p.R, p.G, p.B, 255})
			s.Copied++
		}
	}
	if s.Copied == 0 {
		return s, errors.New("ни один пиксель круга не попал одновременно в источник и результат")
	}
	return s, nil
}
