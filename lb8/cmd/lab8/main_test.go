package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLIValidation(t *testing.T) {
	for _, args := range [][]string{{"-mode", "wrong"}, {"-mode", "batch"}, {"-sx", "-3"}, {"-source", "/missing.png"}, {"unexpected"}, {"-mode", "suite", "-out", ""}} {
		if e := run(args); e == nil {
			t.Fatal("accepted", args)
		}
	}
}
func TestHeadlessDemoAndBatch(t *testing.T) {
	t.Setenv("DISPLAY", "")
	dir := t.TempDir()
	if e := run([]string{"-mode", "demo", "-out", dir}); e != nil {
		t.Fatal(e)
	}
	second := filepath.Join(dir, "batch")
	if e := run([]string{"-mode", "batch", "-source", filepath.Join(dir, "source.png"), "-out", second}); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"source.png", "nearest.png", "bilinear.png", "bicubic.png", "metrics.json"} {
		a, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil {
			t.Fatal(e)
		}
		b, e := os.ReadFile(filepath.Join(second, name))
		if e != nil || string(a) != string(b) {
			t.Fatal("different batch", name, e)
		}
	}
}
