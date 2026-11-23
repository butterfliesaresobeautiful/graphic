package scene

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
)

func SavePNG(path string, img image.Image) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".lab10-*.png")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = png.Encode(file, img)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}
