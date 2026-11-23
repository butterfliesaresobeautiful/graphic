package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"

	"graphiclabs/lb10/internal/bezier"
	"graphiclabs/lb10/internal/scene"
)

type curveExample struct {
	Degree       int                 `json:"degree"`
	Control      []bezier.Point      `json:"control"`
	Segments     int                 `json:"segments"`
	T            float64             `json:"t"`
	Divisions    int                 `json:"divisions_per_segment"`
	Samples      int                 `json:"samples"`
	Construction bezier.Construction `json:"first_segment"`
	At           bezier.Point        `json:"at"`
}

type surfaceExample struct {
	Control    bezier.Grid    `json:"control_rows"`
	U          float64        `json:"u"`
	V          float64        `json:"v"`
	DivisionsU int            `json:"divisions_u"`
	DivisionsV int            `json:"divisions_v"`
	Samples    int            `json:"sample_count"`
	Patches    [4]bezier.Grid `json:"refined_patches"`
	At         bezier.Point   `json:"at"`
}

func saveExamples(out string) error {
	data := struct {
		Curves  []curveExample `json:"curves"`
		Surface surfaceExample `json:"surface"`
	}{}
	bounds := image.Rect(0, 0, 960, 660)
	layers := scene.Layers{Original: true, Refined: true, Selected: -1}
	for i, name := range []string{"quadratic", "cubic"} {
		degree := i + 2
		control, t, n := bezier.DemoCurve(degree)
		result, err := bezier.SampleCurve(control, degree, n)
		if err != nil {
			return err
		}
		step := bezier.Construct(result.Segments[0], t)
		m := image.NewRGBA(bounds)
		scene.DrawCurve(m, bounds, scene.Fit(control, false), scene.CurveView{
			Control: control, Curve: result, Construction: step, Stage: degree, Layers: layers,
		})
		if err := scene.SavePNG(filepath.Join(out, name+".png"), m); err != nil {
			return err
		}
		data.Curves = append(data.Curves, curveExample{
			Degree: degree, Control: control, Segments: len(result.Segments), T: t,
			Divisions: n, Samples: len(result.Samples), Construction: step, At: step.At(),
		})
	}
	grid := bezier.DemoGrid()
	u, v, nu, nv := bezier.DemoU, bezier.DemoV, bezier.DemoNU, bezier.DemoNV
	patch, err := bezier.SampleSurface(grid, u, v, nu, nv)
	if err != nil {
		return err
	}
	m := image.NewRGBA(bounds)
	scene.DrawSurface(m, bounds, scene.Fit(grid.Points(), true), scene.SurfaceView{
		Control: grid, Surface: patch, Layers: layers,
	})
	if err := scene.SavePNG(filepath.Join(out, "surface.png"), m); err != nil {
		return err
	}
	data.Surface = surfaceExample{
		Control: grid, U: u, V: v, DivisionsU: nu, DivisionsV: nv,
		Samples: len(patch.Samples) * len(patch.Samples[0]),
		Patches: patch.Patches, At: patch.At,
	}
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "metrics.json"), append(encoded, '\n'), 0o644)
}
