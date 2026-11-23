package scene

import (
	"image"
	"image/color"
	"math"

	"graphiclabs/lb10/internal/bezier"
)

func (p plot) surface(grid bezier.Grid) {
	projected := make([][]pixel, len(grid))
	for j, row := range grid {
		projected[j] = make([]pixel, len(row))
		for i, point := range row {
			projected[j][i] = p.mapper.mapPoint(point)
		}
	}
	depth := make([]float64, p.box.Dx()*p.box.Dy())
	for i := range depth {
		depth[i] = math.Inf(-1)
	}
	for j := 1; j < len(projected); j++ {
		prev, row := projected[j-1], projected[j]
		for i := 1; i < len(row); i++ {
			p.triangle(prev[i-1], row[i-1], row[i], depth)
			p.triangle(prev[i-1], row[i], prev[i], depth)
		}
	}
}

func (p plot) triangle(a, b, c pixel, depth []float64) {
	ux, uy, uz := b.x-a.x, b.y-a.y, b.depth-a.depth
	vx, vy, vz := c.x-a.x, c.y-a.y, c.depth-a.depth
	nx, ny, nz := uy*vz-uz*vy, uz*vx-ux*vz, ux*vy-uy*vx
	if math.Abs(nz) < 1e-9 {
		return
	}
	light := (-0.35*nx - 0.3*ny + 0.89*nz) / math.Sqrt(nx*nx+ny*ny+nz*nz)
	light = math.Abs(light)
	ink := color.RGBA{
		uint8(80 + 109*light), uint8(154 + 64*light), uint8(185 + 50*light), 255,
	}
	r := image.Rect(
		int(math.Floor(min(a.x, b.x, c.x))), int(math.Floor(min(a.y, b.y, c.y))),
		int(math.Ceil(max(a.x, b.x, c.x))), int(math.Ceil(max(a.y, b.y, c.y))),
	).Intersect(p.box)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			px, py := float64(x)+0.5-a.x, float64(y)+0.5-a.y
			beta := (px*vy - py*vx) / nz
			gamma := (ux*py - uy*px) / nz
			if beta < -1e-9 || gamma < -1e-9 || beta+gamma > 1+1e-9 {
				continue
			}
			z := a.depth + beta*uz + gamma*vz
			i := (y-p.box.Min.Y)*p.box.Dx() + x - p.box.Min.X
			if z > depth[i] {
				depth[i] = z
				p.image.SetRGBA(x, y, ink)
			}
		}
	}
}
