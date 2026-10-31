package imaging

import (
	"encoding/json"
	"errors"
	"image"
	"os"
	"path/filepath"
)

const (
	Empty    = 0
	Blank    = 1
	Copied   = 2
	WithAxes = 3
	Complete = 4
)

type Document struct {
	Source *image.RGBA
	Config Config
	Stage  int
	Images [4]*image.RGBA
	Stats  Stats
}

func (d *Document) Create(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	m, err := Canvas(c.Width, c.Height)
	if err != nil {
		return err
	}
	d.Config = c
	d.Images = [4]*image.RGBA{m, nil, nil, nil}
	d.Stage = Blank
	d.Stats = Stats{}
	return nil
}
func (d *Document) Copy(c Config) error {
	if d.Source == nil {
		return errors.New("загрузите исходное изображение")
	}
	if d.Stage < Blank {
		return errors.New("сначала создайте холст")
	}
	if c.Width != d.Config.Width || c.Height != d.Config.Height {
		return errors.New("размеры изменены: сначала нажмите «Создать холст»")
	}
	c, err := c.Resolve(d.Source.Bounds())
	if err != nil {
		return err
	}
	dst, _ := Canvas(c.Width, c.Height)
	st, err := Transfer(d.Source, dst, c)
	if err != nil {
		return err
	}
	d.Config = c
	d.Images[1] = dst
	d.Images[2] = nil
	d.Images[3] = nil
	d.Stage = Copied
	d.Stats = st
	return nil
}
func (d *Document) AddAxes() error {
	if d.Stage < Copied {
		return errors.New("сначала перенесите круг")
	}
	dst := Clone(d.Images[1])
	if err := Axes(dst, d.Config); err != nil {
		return err
	}
	d.Images[2] = dst
	d.Images[3] = nil
	d.Stage = WithAxes
	return nil
}
func (d *Document) AddGraph() error {
	if d.Stage < WithAxes {
		return errors.New("сначала добавьте оси")
	}
	dst := Clone(d.Images[2])
	if err := Cubic(dst, d.Config); err != nil {
		return err
	}
	d.Images[3] = dst
	d.Stage = Complete
	return nil
}
func (d *Document) Current() *image.RGBA {
	if d.Stage < Blank || d.Stage > Complete {
		return nil
	}
	return d.Images[d.Stage-1]
}
func Build(src *image.RGBA, c Config) (Document, error) {
	d := Document{Source: src}
	if e := d.Create(c); e != nil {
		return d, e
	}
	if e := d.Copy(c); e != nil {
		return d, e
	}
	if e := d.AddAxes(); e != nil {
		return d, e
	}
	if e := d.AddGraph(); e != nil {
		return d, e
	}
	return d, nil
}
func (d *Document) SaveCurrent(out, source string) error {
	if d.Current() == nil {
		return errors.New("нет результата для сохранения")
	}
	if out == "" {
		return errors.New("выберите каталог")
	}
	target := filepath.Join(out, "result.png")
	if err := Protect(source, target); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	return SavePNG(target, d.Current())
}
func (d *Document) Export(out, source string) error {
	if d.Stage != Complete {
		return errors.New("сначала выполните все этапы")
	}
	if out == "" {
		return errors.New("выберите каталог")
	}
	names := []string{"source.png", "blank.png", "transferred.png", "axes.png", "result.png", "metrics.json"}
	for _, name := range names {
		if err := Protect(source, filepath.Join(out, name)); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	imgs := []*image.RGBA{d.Source, d.Images[0], d.Images[1], d.Images[2], d.Images[3]}
	for i, m := range imgs {
		if err := SavePNG(filepath.Join(out, names[i]), m); err != nil {
			return err
		}
	}
	p, _ := Layout(d.Config)
	data, err := json.MarshalIndent(struct {
		Lab, Individual, MaxVariant, Variant int
		Config                               Config
		SourceWidth, SourceHeight            int
		Stats                                Stats
		Plot                                 Plot
	}{2, 28, 20, 8, d.Config, d.Source.Rect.Dx(), d.Source.Rect.Dy(), d.Stats, p}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "metrics.json"), append(data, '\n'), 0644)
}

// SetSource keeps an existing blank canvas but invalidates every derived stage.
func (d *Document) SetSource(src *image.RGBA) {
	d.Source = src
	d.Images[1], d.Images[2], d.Images[3] = nil, nil, nil
	d.Stats = Stats{}
	if d.Images[0] != nil {
		d.Stage = Blank
	} else {
		d.Stage = Empty
	}
}
