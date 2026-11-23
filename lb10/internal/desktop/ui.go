package desktop

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"
	"strings"

	"graphiclabs/lb10/internal/scene"
)

var (
	ink      = color.RGBA{26, 40, 62, 255}
	muted    = color.RGBA{75, 92, 113, 255}
	accent   = color.RGBA{24, 81, 146, 255}
	base     = color.RGBA{240, 245, 250, 255}
	white    = color.RGBA{255, 255, 255, 255}
	border   = color.RGBA{208, 220, 233, 255}
	inactive = color.RGBA{229, 237, 246, 255}
	plotBox  = image.Rect(24, 184, 817, 743)
	sidebar  = image.Rect(836, 184, 1176, 743)
)

type button struct {
	rect   image.Rectangle
	action func()
}

type inputID int

const (
	noInput inputID = iota
	inputT
	inputU
	inputV
	inputNU
	inputNV
	inputX
	inputY
	inputZ
	inputCols
	inputRows
)

type input struct {
	rect image.Rectangle
	id   inputID
}

type editor struct {
	focus   inputID
	text    string
	replace bool
}

type app struct {
	window
	curves             [2]curveState
	surface            surfaceState
	mode               mode
	original, refined  bool
	buttons            []button
	inputs             []input
	edit               editor
	dragging, rotating bool
	lastMouse          image.Point
	status, out        string
}

func newApp(out string) (*app, error) {
	a := &app{original: true, refined: true, out: out}
	for _, m := range []mode{quadratic, cubic, surface} {
		a.mode = m
		if err := a.reset(); err != nil {
			return nil, err
		}
	}
	a.mode = quadratic
	return a, nil
}

func (a *app) paintScene(m *image.RGBA, box image.Rectangle) {
	layers := scene.Layers{Original: a.original, Refined: a.refined, Selected: *a.selected()}
	if a.mode != surface {
		c := &a.curves[a.mode]
		scene.DrawCurve(m, box, c.camera, scene.CurveView{
			Control: c.control, Curve: c.curve, Construction: c.construction,
			Stage: c.stage, Layers: layers,
		})
	} else {
		s := &a.surface
		scene.DrawSurface(m, box, s.camera, scene.SurfaceView{
			Control: s.control, Surface: s.patch, Layers: layers,
		})
	}
}

func (a *app) render() *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	a.buttons = a.buttons[:0]
	a.inputs = a.inputs[:0]
	fill(m, m.Bounds(), base)
	a.text(m, 24, 38, 26, "Кривые и поверхности Безье", ink)
	a.text(m, 24, 63, 14, "Лабораторная работа № 10   ·   Вариант 12", muted)
	titles := []string{"1  Квадратичная кривая", "2  Кубическая кривая", "4  Поверхность"}
	for i, title := range titles {
		w := 255
		x := 24 + i*(w+10)
		tab := mode(i)
		a.addButton(m, image.Rect(x, 78, x+w, 115), title, tab == a.mode, func() {
			a.mode, a.status = tab, ""
		})
	}
	a.parameters(m)
	a.paintScene(m, plotBox)
	fill(m, sidebar, white)
	a.vertexPanel(m)
	if a.mode == surface {
		a.surfacePanel(m)
	} else {
		a.curvePanel(m)
	}
	a.toolbar(m)
	status := a.status
	if status == "" {
		if a.mode == surface {
			status = "Выберите вершину щелчком. Перетаскивание фона вращает сетку, колесо меняет масштаб."
		} else {
			status = "Щелчок на свободном месте добавляет вершину. Перетащите вершину для изменения координат."
		}
	}
	a.text(m, 24, 819, 14, a.fitText(status, 14, 1150), muted)
	return m
}

func (a *app) parameters(m *image.RGBA) {
	if a.mode != surface {
		a.labeledInput(m, 24, 125, 143, 90, "Параметр t", inputT)
		a.labeledInput(m, 264, 125, 472, 80, "Разбиений на сегмент", inputNU)
	} else {
		a.labeledInput(m, 24, 125, 62, 89, "u", inputU)
		a.labeledInput(m, 174, 125, 212, 89, "v", inputV)
		a.labeledInput(m, 330, 125, 482, 72, "Разбиений по u", inputNU)
		a.labeledInput(m, 583, 125, 735, 72, "Разбиений по v", inputNV)
	}
	a.text(m, 868, 147, 14, "Enter применяет введённое значение", muted)
}

func (a *app) vertexPanel(m *image.RGBA) {
	control := a.controls()
	index := *a.selected()
	label := fmt.Sprintf("Вершина P%d  (%d из %d)", index, index+1, len(control))
	if a.mode == surface {
		cols := len(a.surface.control[0])
		label = fmt.Sprintf("Вершина P[%d, %d]  (%d из %d)", index%cols, index/cols, index+1, len(control))
	}
	a.text(m, 853, 217, 16, label, ink)
	a.arrows(m, 194, func(n int) { a.moveVertex(n) })
	a.labeledInput(m, 853, 234, 875, 119, "X", inputX)
	a.labeledInput(m, 1010, 234, 1032, 125, "Y", inputY)
	if a.mode == surface {
		a.labeledInput(m, 853, 275, 875, 119, "Z", inputZ)
	} else {
		a.addButton(m, image.Rect(853, 275, 1038, 311), "Удалить вершину", false, a.deleteVertex)
	}
	fill(m, image.Rect(853, 326, 1158, 327), border)
}

func (a *app) curvePanel(m *image.RGBA) {
	c := &a.curves[a.mode]
	a.text(m, 853, 361, 16, fmt.Sprintf("Сегмент %d из %d", c.segment+1, len(c.curve.Segments)), ink)
	a.arrows(m, 337, func(n int) {
		c.segment = max(0, min(len(c.curve.Segments)-1, c.segment+n))
		c.inspect()
	})
	a.text(m, 853, 406, 16, fmt.Sprintf("Этап %d из %d", c.stage, c.degree), ink)
	a.arrows(m, 382, func(n int) {
		c.stage = max(0, min(c.degree, c.stage+n))
	})
	a.text(m, 853, 449, 16, "Координаты на этом этапе", ink)
	for i, p := range c.construction.Levels[c.stage] {
		a.text(m, 853, 474+21*i, 14, fmt.Sprintf("P%d[%d]  (%s, %s)", i, c.stage, coord(p.X), coord(p.Y)), scene.Step)
	}
	a.text(m, 853, 567, 14, fmt.Sprintf("Отсчётов кривой: %d", len(c.curve.Samples)), ink)
	a.legend(m, 612, true)
}

func (a *app) surfacePanel(m *image.RGBA) {
	s := &a.surface
	a.text(m, 853, 356, 16, "Размеры контрольной сетки", ink)
	a.labeledInput(m, 853, 368, 941, 57, "Столбцов", inputCols)
	a.labeledInput(m, 1021, 368, 1098, 59, "Строк", inputRows)
	a.text(m, 853, 430, 14, fmt.Sprintf("Степень (%d, %d)  ·  Отсчёты %d × %d",
		len(s.control[0])-1, len(s.control)-1, s.nu+1, s.nv+1), ink)
	a.text(m, 853, 467, 16, "Точка S(u, v)", ink)
	for i, f := range []struct {
		name  string
		value float64
	}{{"X", s.patch.At.X}, {"Y", s.patch.At.Y}, {"Z", s.patch.At.Z}} {
		a.text(m, 853, 493+23*i, 14, f.name+" = "+coord(f.value), scene.Result)
	}
	a.text(m, 853, 582, 14, "Уточнение по u и v даёт 4 участка.", muted)
	a.legend(m, 623, false)
}

func (a *app) legend(m *image.RGBA, y int, steps bool) {
	entries := []struct {
		text string
		ink  color.RGBA
	}{
		{"Исходные контрольные вершины", scene.Original},
		{"Уточнённые контрольные вершины", scene.Refined},
		{"Результат", scene.Result},
	}
	if steps {
		entries = append(entries, struct {
			text string
			ink  color.RGBA
		}{"Промежуточные вершины", scene.Step})
	}
	for i, e := range entries {
		yy := y + i*23
		fill(m, image.Rect(853, yy-11, 865, yy+1), e.ink)
		a.text(m, 875, yy, 14, e.text, muted)
	}
}

func (a *app) toolbar(m *image.RGBA) {
	a.addButton(m, image.Rect(24, 755, 242, 793), "Исходные вершины", a.original, func() {
		a.original = !a.original
	})
	a.addButton(m, image.Rect(253, 755, 495, 793), "Уточнённые вершины", a.refined, func() {
		a.refined = !a.refined
	})
	a.addButton(m, image.Rect(506, 755, 637, 793), "Вписать", false, a.fit)
	a.addButton(m, image.Rect(648, 755, 755, 793), "Демо", false, func() {
		a.result(a.reset())
	})
	a.addButton(m, image.Rect(766, 755, 952, 793), "Сохранить PNG", false, a.save)
}

func (a *app) save() {
	m := image.NewRGBA(image.Rect(0, 0, plotBox.Dx(), plotBox.Dy()))
	a.paintScene(m, m.Bounds())
	path := filepath.Join(a.out, modeNames[a.mode]+".png")
	a.result(scene.SavePNG(path, m))
	if a.status == "" {
		a.status = "Сохранено " + path
	}
}

func (a *app) arrows(m *image.RGBA, y int, move func(int)) {
	a.addButton(m, image.Rect(1082, y, 1118, y+35), "‹", false, func() { move(-1) })
	a.addButton(m, image.Rect(1122, y, 1158, y+35), "›", false, func() { move(1) })
}

func (a *app) addButton(m *image.RGBA, r image.Rectangle, label string, active bool, action func()) {
	bg, fg := inactive, ink
	if active {
		bg, fg = accent, white
	}
	fill(m, r, bg)
	a.text(m, r.Min.X+(r.Dx()-a.textWidth(label, 16))/2, r.Min.Y+24, 16, label, fg)
	a.buttons = append(a.buttons, button{r, action})
}

func (a *app) labeledInput(m *image.RGBA, x, y, left, width int, label string, id inputID) {
	a.text(m, x, y+23, 16, label, ink)
	r := image.Rect(left, y, left+width, y+35)
	text := a.currentValue(id)
	a.inputs = append(a.inputs, input{r, id})
	c := border
	if a.edit.focus == id {
		text, c = a.edit.text, accent
	}
	fill(m, r, c)
	fill(m, r.Inset(2), white)
	if a.edit.focus == id && a.edit.replace {
		fill(m, r.Inset(3), inactive)
	}
	text = a.fitText(text, 16, width-16)
	a.text(m, left+8, y+23, 16, text, ink)
}

func coord(x float64) string {
	if math.Abs(x) < 0.00005 {
		x = 0
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", x), "0"), ".")
}
