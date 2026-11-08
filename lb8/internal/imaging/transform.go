package imaging

import (
	"encoding/json"
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
)

type Difference struct {
	First, Second   string
	MAE, RMSE       float64
	Max             int
	DifferentPixels int
}
type Metrics struct {
	Lab, Individual           int
	Variant                   *int
	VariantNote               string
	Config                    Config
	SourceWidth, SourceHeight int
	Geometry                  Geometry
	CoveredPixels             int
	ClippedChannels           [3]int
	Differences               []Difference
}
type Result struct {
	Source  *image.RGBA
	Images  [3]*image.RGBA
	Mask    []bool
	Metrics Metrics
}

func Transform(src *image.RGBA, c Config) (*Result, error) {
	w, h, err := sourceSize(src)
	if err != nil {
		return nil, err
	}
	g, err := Plan(w, h, c)
	if err != nil {
		return nil, err
	}
	r := &Result{Source: src, Mask: make([]bool, g.Width*g.Height), Metrics: Metrics{Lab: 8, Individual: 28, VariantNote: "В ЛР8.docx нет таблицы вариантов и максимального номера; выполнено общее задание", Config: c, SourceWidth: w, SourceHeight: h, Geometry: g}}
	for i := range r.Images {
		r.Images[i], _ = Canvas(g.Width, g.Height)
	}
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			u, v := g.SourcePoint(x, y, w, h)
			if !covered(u, v, w, h) {
				continue
			}
			r.Mask[y*g.Width+x] = true
			r.Metrics.CoveredPixels++
			for method, dst := range r.Images {
				p := Sample(src, u, v, Method(method))
				offset := dst.PixOffset(x, y)
				for ch, value := range p {
					if value < -1e-8 || value > 255+1e-8 {
						r.Metrics.ClippedChannels[method]++
					}
					dst.Pix[offset+ch] = uint8(math.Round(math.Max(0, math.Min(255, value))))
				}
			}
		}
	}
	if r.Metrics.CoveredPixels == 0 {
		return nil, errors.New("на выходную сетку не попало ни одного пикселя; увеличьте масштабы")
	}
	for _, pair := range [][2]int{{0, 1}, {0, 2}, {1, 2}} {
		r.Metrics.Differences = append(r.Metrics.Differences, compare(r, pair[0], pair[1]))
	}
	return r, nil
}
func compare(r *Result, a, b int) Difference {
	d := Difference{First: Methods[a], Second: Methods[b]}
	sum, squares := 0., 0.
	for i, in := range r.Mask {
		if in {
			changed := false
			for c := 0; c < 3; c++ {
				delta := int(r.Images[a].Pix[4*i+c]) - int(r.Images[b].Pix[4*i+c])
				if delta < 0 {
					delta = -delta
				}
				if delta > 0 {
					changed = true
				}
				sum += float64(delta)
				squares += float64(delta * delta)
				d.Max = max(d.Max, delta)
			}
			if changed {
				d.DifferentPixels++
			}
		}
	}
	n := float64(r.Metrics.CoveredPixels * 3)
	d.MAE = sum / n
	d.RMSE = math.Sqrt(squares / n)
	return d
}
func (r *Result) Save(out, source string) error {
	if r == nil {
		return errors.New("сначала выполните преобразование")
	}
	if out == "" {
		return errors.New("укажите каталог")
	}
	names := []string{"source.png", "nearest.png", "bilinear.png", "bicubic.png", "metrics.json"}
	for _, n := range names {
		if err := Protect(source, filepath.Join(out, n)); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	for i, m := range []*image.RGBA{r.Source, r.Images[0], r.Images[1], r.Images[2]} {
		if err := SavePNG(filepath.Join(out, names[i]), m); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(r.Metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "metrics.json"), append(data, '\n'), 0644)
}
func Suite(src *image.RGBA, out, source string) error {
	if out == "" {
		return errors.New("укажите каталог")
	}
	// Check every target before writing any suite results.
	for _, p := range Presets() {
		for _, n := range []string{"source.png", "nearest.png", "bilinear.png", "bicubic.png", "metrics.json"} {
			if err := Protect(source, filepath.Join(out, p.Name, n)); err != nil {
				return err
			}
		}
	}
	for _, p := range Presets() {
		r, err := Transform(src, p.Config)
		if err != nil {
			return err
		}
		if err = r.Save(filepath.Join(out, p.Name), source); err != nil {
			return err
		}
	}
	return nil
}
