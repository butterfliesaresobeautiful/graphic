package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLIValidation(t *testing.T) {
	for _, args := range [][]string{{"-mode", "bad"}, {"-shape", "oval"}, {"-size", "0"}, {"-source", "/does-not-exist.png"}, {"unexpected"}} {
		if err := run(args); err == nil {
			t.Fatalf("invalid accepted %v", args)
		}
	}
	if err := run([]string{"-h"}); err != nil {
		t.Fatal(err)
	}
}
func TestBatch(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("fourth method requires X11")
	}
	dir := t.TempDir()
	if err := run([]string{"-mode", "batch", "-out", dir, "-size", "80", "-cx", "60", "-cy", "60", "-width", "120", "-height", "120"}); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"equation.png", "parametric.png", "bresenham.png", "native.png", "overlay.png", "difference.png", "zoom.png", "metrics.json"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Fatal(err)
		}
	}
}
