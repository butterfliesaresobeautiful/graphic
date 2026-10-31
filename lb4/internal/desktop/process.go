package desktop

import (
	"graphiclabs/lb4/internal/native"
	"graphiclabs/lb4/internal/raster"
	"image"
)

func Process(background *image.RGBA, c raster.Config) (raster.Results, error) {
	masks, err := raster.Custom(c)
	if err != nil {
		return raster.Results{}, err
	}
	builtin, err := native.Rasterize(c)
	if err != nil {
		return raster.Results{}, err
	}
	return raster.Compose(background, c, [4]*raster.Mask{masks[0], masks[1], masks[2], builtin})
}
