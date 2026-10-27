package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"graphiclabs/lb6/internal/desktop"
	"graphiclabs/lb6/internal/raster"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	f := flag.NewFlagSet("lab6", flag.ContinueOnError)
	mode := f.String("mode", "gui", "gui: окно X11; batch: обработка; demo: пример")
	source := f.String("source", "", "исходный PNG, JPEG или GIF")
	out := f.String("out", "output", "каталог результатов PNG")
	angle := f.Float64("angle", 30, "угол в градусах по часовой стрелке")
	axis := f.String("axis", "vertical", "ось отражения: vertical или horizontal")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("неожиданные позиционные аргументы")
	}
	if *mode != "gui" && *mode != "batch" && *mode != "demo" {
		return errors.New("режим: gui, batch или demo")
	}
	if _, err := raster.Variant8(1, 1, *angle, *axis); err != nil {
		return err
	}
	var src *image.RGBA
	var err error
	if *source != "" {
		src, err = raster.Load(*source)
		if err != nil {
			return err
		}
	} else {
		if *mode == "batch" {
			return errors.New("для batch укажите -source")
		}
		src = raster.Demo()
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
		return desktop.Run(src, *source, *out, *angle, *axis)
	}
	r, err := raster.Process(src, *angle, *axis)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	outputs := []struct {
		name string
		img  image.Image
	}{
		{"affine.png", r.Affine}, {"restored.png", r.Restored}, {"nonlinear.png", r.Nonlinear},
	}
	if *mode == "demo" {
		outputs = append(outputs, struct {
			name string
			img  image.Image
		}{"source.png", src})
	}
	for _, item := range outputs {
		if *source != "" {
			a, _ := filepath.Abs(*source)
			b, _ := filepath.Abs(filepath.Join(*out, item.name))
			if a == b {
				return errors.New("выходной файл совпадает с исходным: выберите другой каталог")
			}
		}
	}
	for _, item := range outputs {
		if err = raster.Save(filepath.Join(*out, item.name), item.img); err != nil {
			return err
		}
	}
	mae, _ := raster.MAE(src, r.Restored)
	metrics := struct {
		Lab, Individual, Variant int
		Angle                    float64
		Axis                     string
		Width, Height            int
		Matrix                   raster.Matrix
		Stats                    [3]raster.Stats
		RestorationMAE           float64
	}{6, 28, 28 % 20, *angle, *axis, src.Rect.Dx(), src.Rect.Dy(), r.Matrix, r.Stats, mae}
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*out, "metrics.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("ЛБ6, вариант 28 %% 20 = 8. PNG сохранены: %s\n", *out)
	fmt.Printf("Угол: %g°; отражение: %s; MAE восстановления RGB: %.6f\n", *angle, *axis, mae)
	return nil
}
