package main

import (
	"errors"
	"flag"
	"fmt"
	"graphiclabs/lb8/internal/desktop"
	"graphiclabs/lb8/internal/imaging"
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
	c := imaging.Default()
	f := flag.NewFlagSet("lab8", flag.ContinueOnError)
	mode := f.String("mode", "gui", "gui: окно; batch: файл; demo: пример; suite: 8 экспериментов без экрана")
	source := f.String("source", "", "PNG/JPEG/GIF; для batch обязателен")
	out := f.String("out", "output", "каталог PNG и metrics.json")
	f.Float64Var(&c.ScaleX, "sx", c.ScaleX, "масштаб X: 0.05…8")
	f.Float64Var(&c.ScaleY, "sy", c.ScaleY, "масштаб Y: 0.05…8")
	f.Float64Var(&c.Angle, "angle", c.Angle, "градусы по часовой стрелке: −360…360")
	f.Float64Var(&c.ShearX, "hx", c.ShearX, "горизонтальный скос: −3…3")
	f.Float64Var(&c.ShearY, "hy", c.ShearY, "вертикальный скос: −3…3")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("неожиданные позиционные аргументы")
	}
	if *mode != "gui" && *mode != "batch" && *mode != "demo" && *mode != "suite" {
		return errors.New("режим: gui, batch, demo или suite")
	}
	if err := c.Validate(); err != nil {
		return err
	}
	src := imaging.Demo()
	if *source != "" {
		var err error
		src, err = imaging.Load(*source)
		if err != nil {
			return err
		}
	} else if *mode == "batch" {
		return errors.New("для batch укажите -source")
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
	if *mode == "suite" {
		return imaging.Suite(src, *out, *source)
	}
	r, err := imaging.Transform(src, c)
	if err != nil {
		return err
	}
	if err = r.Save(*out, *source); err != nil {
		return err
	}
	fmt.Printf("ЛБ8. %d×%d → %d×%d; три метода; сохранено: %s\n", src.Rect.Dx(), src.Rect.Dy(), r.Metrics.Geometry.Width, r.Metrics.Geometry.Height, *out)
	return nil
}
