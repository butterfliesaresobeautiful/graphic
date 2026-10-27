// Package raster implements the RGB file operations for lab 6.
package raster

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

const MaxSide = 4096
const MaxPixels = 16_000_000
const MaxFile = 32 << 20

func Dimensions(w, h int) error {
	if w < 1 || h < 1 || w > MaxSide || h > MaxSide || w*h > MaxPixels {
		return fmt.Errorf("размеры: 1…%d, не более %d пикселей", MaxSide, MaxPixels)
	}
	return nil
}

// New returns an opaque white canvas, including every boundary pixel.
func New(w, h int) (*image.RGBA, error) {
	if err := Dimensions(w, h); err != nil {
		return nil, err
	}
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range m.Pix {
		m.Pix[i] = 255
	}
	return m, nil
}

// Decode limits input and allocation before decoding. Transparency is composed
// over white; opaque RGB values are unchanged. GIF uses its first frame.
func Decode(r io.Reader) (*image.RGBA, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxFile+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFile {
		return nil, errors.New("файл больше 32 МиБ")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ожидается PNG, JPEG или GIF: %w", err)
	}
	if err = Dimensions(cfg.Width, cfg.Height); err != nil {
		return nil, err
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("повреждённое изображение: %w", err)
	}
	dst, _ := New(cfg.Width, cfg.Height)
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			r, g, b, a := src.At(x+src.Bounds().Min.X, y+src.Bounds().Min.Y).RGBA()
			dst.SetRGBA(x, y, color.RGBA{
				uint8((r + 65535 - a) >> 8),
				uint8((g + 65535 - a) >> 8),
				uint8((b + 65535 - a) >> 8), 255,
			})
		}
	}
	return dst, nil
}

func Load(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("открытие: %w", err)
	}
	defer f.Close()
	return Decode(f)
}

// Save writes PNG atomically; a failed encode does not truncate an old result.
func Save(path string, m image.Image) error {
	if m == nil {
		return errors.New("нет изображения для сохранения")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".lab2-*.tmp")
	if err != nil {
		return fmt.Errorf("создание файла: %w", err)
	}
	defer os.Remove(f.Name())
	if err = png.Encode(f, m); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
