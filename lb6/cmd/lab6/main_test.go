package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDemoAndBatch(t *testing.T) {
	dir := t.TempDir()
	if err := run([]string{"-mode", "demo", "-out", dir}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "batch")
	if err := run([]string{"-mode", "batch", "-source", filepath.Join(dir, "source.png"), "-out", out}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"affine.png", "restored.png", "nonlinear.png", "metrics.json"} {
		a, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(out, name))
		if err != nil || !bytes.Equal(a, b) {
			t.Fatal(name, err)
		}
	}
	for _, args := range [][]string{{"-mode", "bad"}, {"-mode", "batch"}, {"-angle", "NaN"}, {"-axis", "bad"}, {"-source", "missing.png"}} {
		if err := run(args); err == nil {
			t.Fatal("bad input accepted", args)
		}
	}
}
