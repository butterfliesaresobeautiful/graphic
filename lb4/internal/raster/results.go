package raster

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
)

var Names = [4]string{"equation", "parametric", "bresenham", "native"}

type Metrics struct {
	Config                   Config
	MethodNames              [4]string
	Pixels                   [4]int
	PairwiseXOR              [4][4]int
	Common, Union, Different int
	NativePrimitive          string
}
type Results struct {
	Images                    [4]*image.RGBA
	Overlay, Difference, Zoom *image.RGBA
	Masks                     [4]*Mask
	Metrics                   Metrics
}

func Compose(background *image.RGBA, c Config, masks [4]*Mask) (Results, error) {
	r := Results{Masks: masks, Metrics: Metrics{Config: c, MethodNames: Names, NativePrimitive: "X11 PolyArc; line width 0; no antialiasing"}}
	if background == nil || background.Rect != image.Rect(0, 0, c.Width, c.Height) {
		return r, errors.New("фон должен соответствовать размерам холста")
	}
	for _, m := range masks {
		if m == nil || m.W != c.Width || m.H != c.Height {
			return r, errors.New("неверный размер маски")
		}
	}
	for i, m := range masks {
		r.Images[i], _ = New(c.Width, c.Height)
		draw.Draw(r.Images[i], background.Rect, background, image.Point{}, draw.Src)
		r.Metrics.Pixels[i] = m.Count()
	}
	r.Overlay, _ = New(c.Width, c.Height)
	r.Difference, _ = New(c.Width, c.Height)
	palette := [4]color.RGBA{{30, 95, 210, 255}, {220, 115, 0, 255}, {0, 140, 90, 255}, {150, 55, 190, 255}}
	black := color.RGBA{0, 0, 0, 255}
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			n := 0
			sum := [3]int{}
			for i, m := range masks {
				if m.At(x, y) {
					r.Images[i].SetRGBA(x, y, black)
					n++
					sum[0] += int(palette[i].R)
					sum[1] += int(palette[i].G)
					sum[2] += int(palette[i].B)
				}
			}
			if n > 0 {
				r.Metrics.Union++
				r.Overlay.SetRGBA(x, y, color.RGBA{uint8(sum[0] / n), uint8(sum[1] / n), uint8(sum[2] / n), 255})
			}
			if n == 4 {
				r.Metrics.Common++
				r.Overlay.SetRGBA(x, y, black)
				r.Difference.SetRGBA(x, y, color.RGBA{180, 180, 180, 255})
			} else if n > 0 {
				r.Metrics.Different++
				r.Difference.SetRGBA(x, y, color.RGBA{220, 35, 50, 255})
			}
			for i := 0; i < 4; i++ {
				for j := 0; j < 4; j++ {
					if masks[i].At(x, y) != masks[j].At(x, y) {
						r.Metrics.PairwiseXOR[i][j]++
					}
				}
			}
		}
	}
	r.Zoom = Zoom(c, masks)
	return r, nil
}

// Zoom shows the same 32x32 crop for all four methods, using 8x pixel cells.
func Zoom(c Config, m [4]*Mask) *image.RGBA {
	dst, _ := New(528, 528)
	region := c.ZoomRect()
	for i, mask := range m {
		ox, oy := (i%2)*272, (i/2)*272
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				col := color.RGBA{245, 247, 250, 255}
				if mask.At(region.Min.X+x, region.Min.Y+y) {
					col = color.RGBA{30, 45, 65, 255}
				}
				draw.Draw(dst, image.Rect(ox+x*8, oy+y*8, ox+x*8+7, oy+y*8+7), image.NewUniform(col), image.Point{}, draw.Src)
			}
		}
	}
	return dst
}

// SaveAll writes separate method outputs and comparisons. Reject input aliases
// before any write so a loaded background cannot accidentally be overwritten.
func (r Results) SaveAll(out, source string) error {
	if out == "" {
		return errors.New("не выбран каталог результатов")
	}
	items := map[string]*image.RGBA{"overlay.png": r.Overlay, "difference.png": r.Difference, "zoom.png": r.Zoom}
	for i, name := range Names {
		items[name+".png"] = r.Images[i]
	}
	for name, img := range items {
		if img == nil {
			return errors.New("сначала выполните построение")
		}
		if err := protect(source, filepath.Join(out, name)); err != nil {
			return err
		}
	}
	if err := protect(source, filepath.Join(out, "metrics.json")); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	for name, img := range items {
		if err := Save(filepath.Join(out, name), img); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(r.Metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "metrics.json"), append(data, '\n'), 0644)
}
func protect(source, target string) error {
	if source == "" {
		return nil
	}
	a, _ := filepath.Abs(source)
	b, _ := filepath.Abs(target)
	if a == b {
		return errors.New("выходной файл совпадает с фоном; выберите другой каталог")
	}
	sa, ea := os.Stat(source)
	sb, eb := os.Stat(target)
	if ea == nil && eb == nil && os.SameFile(sa, sb) {
		return errors.New("выходной файл ссылается на фон; выберите другой каталог")
	}
	return nil
}
