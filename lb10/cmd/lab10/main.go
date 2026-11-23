package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"graphiclabs/lb10/internal/desktop"
)

func main() {
	if err := run(os.Args[1:]); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "ЛБ10.", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("lab10", flag.ContinueOnError)
	mode := flags.String("mode", "gui", "gui — окно X11, demo — сохранить три примера без окна")
	out := flags.String("out", defaultOutput(), "каталог для результатов PNG")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("неожиданный аргумент %q", flags.Arg(0))
	}
	switch *mode {
	case "gui":
		return desktop.Run(*out)
	case "demo":
		if err := saveExamples(*out); err != nil {
			return err
		}
		fmt.Println("Три примера и расчёты сохранены в", *out)
		return nil
	default:
		return fmt.Errorf("режим %q не поддерживается. Допустимы gui и demo", *mode)
	}
}

func defaultOutput() string {
	exe, err := os.Executable()
	if err != nil {
		return "output"
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "bin" {
		dir = filepath.Dir(dir)
	}
	return filepath.Join(dir, "output")
}
