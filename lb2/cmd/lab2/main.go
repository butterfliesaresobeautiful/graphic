package main

import (
	"errors"
	"flag"
	"fmt"
	"graphiclabs/lb2/internal/desktop"
	"graphiclabs/lb2/internal/imaging"
	"image"
	"os"
	"path/filepath"
)

func main() {
	if e := run(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", e)
		os.Exit(1)
	}
}
func run(args []string) error {
	c := imaging.Default()
	f := flag.NewFlagSet("lab2", flag.ContinueOnError)
	mode := f.String("mode", "gui", "gui: окно; batch: обработка файла без экрана; demo: полный пример без экрана")
	source := f.String("source", "", "PNG, JPEG или GIF")
	out := f.String("out", "output", "каталог результатов")
	f.IntVar(&c.Width, "width", c.Width, "ширина холста")
	f.IntVar(&c.Height, "height", c.Height, "высота холста")
	f.IntVar(&c.Radius, "radius", c.Radius, "радиус круга")
	f.BoolVar(&c.Auto, "auto", true, "позиции варианта 8; false — ручные координаты")
	f.IntVar(&c.SX, "sx", 0, "X центра в источнике при -auto=false")
	f.IntVar(&c.SY, "sy", 0, "Y центра в источнике")
	f.IntVar(&c.DX, "dx", 0, "X центра в результате")
	f.IntVar(&c.DY, "dy", 0, "Y центра в результате")
	if e := f.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if f.NArg() > 0 {
		return errors.New("неожиданные позиционные аргументы")
	}
	if *mode != "gui" && *mode != "batch" && *mode != "demo" {
		return errors.New("режим: gui, batch или demo")
	}
	if e := c.Validate(); e != nil {
		return e
	}
	var src *image.RGBA
	var err error
	if *source != "" {
		src, err = imaging.Load(*source)
		if err != nil {
			return err
		}
	} else {
		if *mode == "batch" {
			return errors.New("для batch укажите -source")
		}
		src = imaging.Demo()
	}
	if *mode == "gui" {
		if *out == "output" {
			exe, e := os.Executable()
			if e != nil {
				return e
			}
			base := filepath.Dir(exe)
			if filepath.Base(base) == "bin" {
				base = filepath.Dir(base)
			}
			*out = filepath.Join(base, "output")
		}
		return desktop.Run(src, *source, *out, c)
	}
	d, err := imaging.Build(src, c)
	if err != nil {
		return err
	}
	if err = d.Export(*out, *source); err != nil {
		return err
	}
	fmt.Printf("ЛБ2: 28 %% 20 = 8. (%d,%d) → (%d,%d), R=%d. Скопировано %d; отсечено %d.\nСохранено: %s\n", d.Config.SX, d.Config.SY, d.Config.DX, d.Config.DY, d.Config.Radius, d.Stats.Copied, d.Stats.Clipped, *out)
	return nil
}
