// Package imaging contains the new lab 2 core. It uses only Go's standard library.
package imaging

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
const MaxPixels = 16000000
const MaxFile = 32 << 20

func Dimensions(w, h int) error {
	if w < 1 || h < 1 || w > MaxSide || h > MaxSide || w*h > MaxPixels {
		return errors.New("стороны 1…4096; площадь не более 16 000 000 пикселей")
	}
	return nil
}
func Canvas(w, h int) (*image.RGBA, error) {
	if err := Dimensions(w, h); err != nil {
		return nil, err
	}
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range m.Pix {
		m.Pix[i] = 255
	}
	return m, nil
}
func Clone(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Rect)
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			dst.SetRGBA(x, y, src.RGBAAt(x, y))
		}
	}
	return dst
}
func Decode(r io.Reader) (*image.RGBA, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxFile+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxFile {
		return nil, errors.New("файл превышает 32 МиБ")
	}
	c, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("ожидается PNG, JPEG или GIF: %w", err)
	}
	if err = Dimensions(c.Width, c.Height); err != nil {
		return nil, err
	}
	src, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	dst, _ := Canvas(c.Width, c.Height)
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			r, g, b, a := src.At(x+src.Bounds().Min.X, y+src.Bounds().Min.Y).RGBA()
			dst.SetRGBA(x, y, color.RGBA{uint8((r + 65535 - a) >> 8), uint8((g + 65535 - a) >> 8), uint8((b + 65535 - a) >> 8), 255})
		}
	}
	return dst, nil
}
func Load(path string) (*image.RGBA, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return Decode(f)
}
func Protect(source, target string) error {
	if source == "" {
		return nil
	}
	a, _ := filepath.Abs(source)
	b, _ := filepath.Abs(target)
	if a == b {
		return errors.New("выходной файл совпадает с исходным")
	}
	sa, ea := os.Stat(a)
	sb, eb := os.Stat(b)
	if ea == nil && eb == nil && os.SameFile(sa, sb) {
		return errors.New("выходной файл ссылается на исходный")
	}
	return nil
}
func SavePNG(path string, m *image.RGBA) error {
	if m == nil {
		return errors.New("сначала создайте изображение")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".lb2-*.png")
	if err != nil {
		return err
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
