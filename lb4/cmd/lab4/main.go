package main

import (
	"errors"
	"flag"
	"fmt"
	"graphiclabs/lb4/internal/desktop"
	"graphiclabs/lb4/internal/raster"
	"image"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	c := raster.Default()
	f := flag.NewFlagSet("lab4", flag.ContinueOnError)
	mode := f.String("mode", "gui", "gui: настольное окно; batch: сохранить результаты (тоже нужен X11)")
	source := f.String("source", "", "фон PNG/JPEG/GIF; по умолчанию белый холст")
	out := f.String("out", "output", "каталог результатов")
	shape := f.String("shape", "reuleaux", "reuleaux или circle")
	f.IntVar(&c.Size, "size", c.Size, "ширина треугольника Рёло")
	f.IntVar(&c.Radius, "radius", c.Radius, "радиус в режиме circle")
	f.IntVar(&c.CX, "cx", c.CX, "X центра фигуры / центроида исходного треугольника")
	f.IntVar(&c.CY, "cy", c.CY, "Y центра фигуры / центроида исходного треугольника")
	f.IntVar(&c.Width, "width", c.Width, "ширина белого холста")
	f.IntVar(&c.Height, "height", c.Height, "высота белого холста")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("неожиданные позиционные аргументы")
	}
	if *mode != "gui" && *mode != "batch" {
		return errors.New("режим: gui или batch")
	}
	if *shape != "reuleaux" && *shape != "circle" {
		return errors.New("фигура: reuleaux или circle")
	}
	c.Circle = *shape == "circle"
	var src *image.RGBA
	var err error
	if *source != "" {
		src, err = raster.Load(*source)
		if err != nil {
			return err
		}
		c.Width, c.Height = src.Rect.Dx(), src.Rect.Dy()
	} else {
		src, err = raster.New(c.Width, c.Height)
		if err != nil {
			return err
		}
	}
	if err = c.Validate(); err != nil {
		return err
	}
	if *mode == "gui" {
		if *out == "output" {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			base := filepath.Dir(exe)
			if filepath.Base(base) == "bin" {
				base = filepath.Dir(base)
			}
			*out = filepath.Join(base, "output")
		}
		return desktop.Run(src, *source, *out, c)
	}
	r, err := desktop.Process(src, c)
	if err != nil {
		return err
	}
	if err = r.SaveAll(*out, *source); err != nil {
		return err
	}
	fmt.Printf("ЛБ4: 28 %% 20 = 8. Сохранено: %s\nПикселей: %v; общих: %d; различающихся: %d\n", *out, r.Metrics.Pixels, r.Metrics.Common, r.Metrics.Different)
	return nil
}
