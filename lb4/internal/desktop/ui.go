package desktop

import (
	"errors"
	"fmt"
	"github.com/jezek/xgb/xproto"
	xdraw "golang.org/x/image/draw"
	"graphiclabs/lb4/internal/raster"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ink = color.RGBA{36, 51, 73, 255}
var blue = color.RGBA{35, 91, 175, 255}
var muted = color.RGBA{87, 105, 129, 255}
var white = color.RGBA{255, 255, 255, 255}
var fields = map[int]image.Rectangle{1: image.Rect(100, 130, 180, 166), 4: image.Rect(270, 130, 350, 166), 5: image.Rect(440, 130, 520, 166), 6: image.Rect(620, 130, 700, 166), 2: image.Rect(142, 182, 952, 218)}

func (a *app) add(m *image.RGBA, r image.Rectangle, label, action string) {
	a.buttons = append(a.buttons, button{r, label, action})
	fill(m, r, blue)
	a.text(m, r.Min.X+12, r.Min.Y+25, 16, label, white)
}
func (a *app) field(m *image.RGBA, r image.Rectangle, text string, id int) {
	c := color.RGBA{186, 198, 214, 255}
	if a.focus == id {
		c = blue
	}
	fill(m, r, c)
	fill(m, r.Inset(2), white)
	limit := r.Dx()/10 - 2
	s := []rune(text)
	if len(s) > limit {
		s = s[len(s)-limit:]
	}
	a.text(m, r.Min.X+8, r.Min.Y+25, 16, string(s), ink)
}
func (a *app) render() *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	a.buttons = nil
	fill(m, m.Bounds(), color.RGBA{240, 244, 249, 255})
	a.text(m, 24, 38, 26, "Растеризация окружностей", ink)
	a.text(m, 24, 64, 14, "Лабораторная 04 / Номер 28 / Вариант 08 — треугольник Рёло заданной ширины", muted)
	a.add(m, image.Rect(24, 80, 230, 116), "Открыть фон…", "open")
	a.add(m, image.Rect(242, 80, 380, 116), "Белый фон", "blank")
	shape := "Треугольник Рёло"
	if a.config.Circle {
		shape = "Окружность"
	}
	a.add(m, image.Rect(392, 80, 700, 116), "Фигура: "+shape, "shape")
	a.add(m, image.Rect(712, 80, 932, 116), "Построить 4 метода", "all")
	a.add(m, image.Rect(944, 80, 1176, 116), "Сохранить результаты", "save-all")
	for _, f := range []struct {
		id, x        int
		label, value string
	}{{1, 24, "Ширина", a.size}, {4, 200, "Центр X", a.cx}, {5, 370, "Центр Y", a.cy}, {6, 540, "R окруж.", a.radius}} {
		a.text(m, f.x, 154, 16, f.label, ink)
		a.field(m, fields[f.id], f.value, f.id)
	}
	views := []string{"Четыре метода", "Сравнение и фон", "Пиксели крупно"}
	a.add(m, image.Rect(740, 130, 1176, 166), "Вид: "+views[a.view], "view")
	a.text(m, 24, 206, 16, "Каталог PNG", ink)
	a.field(m, fields[2], a.out, 2)
	a.add(m, image.Rect(966, 182, 1176, 218), "Выбрать каталог…", "directory")
	for i, line := range wrap(a.status, 118) {
		if i == 2 {
			break
		}
		a.text(m, 24, 245+i*20, 14, line, muted)
	}
	imgs := a.results.Images
	labels := [4]string{"1. Уравнение окружности", "2. Параметрическое уравнение", "3. Алгоритм Брезенхема", "4. Примитив X11 PolyArc"}
	if a.view == 1 {
		imgs = [4]*image.RGBA{a.results.Overlay, a.results.Difference, a.results.Zoom, a.source}
		labels = [4]string{"Наложение: общие — чёрные", "Различия — красные; общие — серые", "Одинаковый участок, пиксели ×8", "Исходный фон"}
	}
	for i, src := range imgs {
		x, y := 24+(i%2)*584, 284+(i/2)*264
		fill(m, image.Rect(x, y, x+568, y+248), white)
		a.text(m, x+12, y+25, 16, labels[i], ink)
		r := image.Rect(x+12, y+46, x+556, y+236)
		fill(m, r, color.RGBA{231, 237, 245, 255})
		if src == nil {
			a.text(m, x+32, y+140, 16, "Нажмите «Построить 4 метода»", muted)
			continue
		}
		bounds := src.Bounds()
		if a.view == 2 {
			bounds = a.results.Metrics.Config.ZoomRect()
			crop, _ := raster.New(32, 32)
			draw.Draw(crop, crop.Bounds(), src, bounds.Min, draw.Src)
			src = crop
			bounds = src.Bounds()
		}
		scale := min(float64(r.Dx())/float64(bounds.Dx()), float64(r.Dy())/float64(bounds.Dy()))
		w, h := int(float64(bounds.Dx())*scale), int(float64(bounds.Dy())*scale)
		l, t := r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2
		if scale < 1 {
			// Area-aware preview only; saved rasters retain the original pixels.
			xdraw.CatmullRom.Scale(m, image.Rect(l, t, l+w, t+h), src, bounds, draw.Src, nil)
		} else {
			xdraw.NearestNeighbor.Scale(m, image.Rect(l, t, l+w, t+h), src, bounds, draw.Src, nil)
		}
	}
	a.text(m, 24, 832, 14, "Рёло: ширина = радиус каждой из трёх дуг. R окруж. используется только в режиме окружности.", muted)
	if a.dialog != "" {
		a.renderDialog(m)
	}
	return m
}
func wrap(s string, n int) []string {
	var lines []string
	for _, w := range strings.Fields(s) {
		if len(lines) == 0 || len([]rune(lines[len(lines)-1]+" "+w)) > n {
			lines = append(lines, w)
		} else {
			lines[len(lines)-1] += " " + w
		}
	}
	return lines
}
func (a *app) click(x, y int, detail xproto.Button) {
	p := image.Pt(x, y)
	if a.dialog != "" && (detail == 4 || detail == 5) {
		if detail == 4 {
			a.page = max(0, a.page-1)
		} else {
			a.page = min(max(0, (len(a.entries)-1)/16), a.page+1)
		}
		return
	}
	if detail != 1 {
		return
	}
	if a.dialog != "" {
		if p.In(image.Rect(110, 140, 925, 176)) {
			a.focus = 3
			a.selectAll = true
			return
		}
	} else {
		for id, r := range fields {
			if p.In(r) {
				a.focus = id
				a.selectAll = true
				return
			}
		}
	}
	for _, b := range a.buttons {
		if p.In(b.rect) {
			if err := a.action(b.action); err != nil {
				a.status = "Ошибка: " + err.Error()
			}
			return
		}
	}
}
func (a *app) parameters() (raster.Config, error) {
	c := a.config
	for _, f := range []struct {
		s string
		p *int
	}{{a.size, &c.Size}, {a.cx, &c.CX}, {a.cy, &c.CY}, {a.radius, &c.Radius}} {
		n, err := strconv.Atoi(strings.TrimSpace(f.s))
		if err != nil {
			return c, errors.New("ширина, координаты и радиус должны быть целыми числами")
		}
		*f.p = n
	}
	c.Width, c.Height = a.source.Rect.Dx(), a.source.Rect.Dy()
	return c, c.Validate()
}
func (a *app) action(action string) error {
	switch action {
	case "open", "directory":
		a.dialog = action
		a.page = 0
		a.focus = 3
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		if action == "directory" && a.out != "" {
			dir = a.out
		}
		if action == "open" && a.sourcePath != "" {
			dir = filepath.Dir(a.sourcePath)
		}
		if err = a.readDir(dir); err != nil {
			fallback, _ := os.Getwd()
			return a.readDir(fallback)
		}
	case "close":
		a.dialog = ""
		a.focus = 1
	case "browse":
		return a.readDir(a.dirInput)
	case "parent":
		return a.readDir(filepath.Dir(a.dir))
	case "prev":
		a.page = max(0, a.page-1)
	case "next":
		a.page = min(max(0, (len(a.entries)-1)/16), a.page+1)
	case "choose-dir":
		a.out = a.dir
		a.dialog = ""
		a.focus = 2
	case "blank":
		a.source, _ = raster.New(640, 480)
		a.sourcePath = ""
		a.results = raster.Results{}
		a.status = "Белый холст 640×480. Нажмите «Построить»."
	case "shape":
		a.config.Circle = !a.config.Circle
		a.results = raster.Results{}
		a.status = "Фигура изменена. Нажмите «Построить»."
	case "view":
		a.view = (a.view + 1) % 3
	case "all":
		c, err := a.parameters()
		if err != nil {
			return err
		}
		r, err := Process(a.source, c)
		if err != nil {
			return err
		}
		a.results = r
		a.config = c
		a.status = fmt.Sprintf("Пикселей: %v. Общих: %d. Различающихся в объединении: %d. Параметры применены.", r.Metrics.Pixels, r.Metrics.Common, r.Metrics.Different)
	case "save-all":
		if err := a.results.SaveAll(a.out, a.sourcePath); err != nil {
			return err
		}
		a.status = "Сохранены 7 PNG и metrics.json: " + a.out
	default:
		if !strings.HasPrefix(action, "entry-") {
			return errors.New("неизвестное действие")
		}
		i, err := strconv.Atoi(strings.TrimPrefix(action, "entry-"))
		if err != nil || i < 0 || i >= len(a.entries) {
			return errors.New("неверный файл")
		}
		path := filepath.Join(a.dir, a.entries[i].Name())
		if a.entries[i].IsDir() {
			return a.readDir(path)
		}
		img, err := raster.Load(path)
		if err != nil {
			return err
		}
		a.source = img
		a.sourcePath = path
		a.results = raster.Results{}
		a.dialog = ""
		a.focus = 1
		a.status = fmt.Sprintf("Загружен фон %s, %d×%d. Проверьте положение фигуры и нажмите «Построить».", filepath.Base(path), img.Rect.Dx(), img.Rect.Dy())
	}
	return nil
}

func (a *app) readDir(path string) error {
	full, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return err
	}
	var list []os.DirEntry
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if e.IsDir() || a.dialog == "open" && (ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif") {
			list = append(list, e)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].IsDir() != list[j].IsDir() {
			return list[i].IsDir()
		}
		return list[i].Name() < list[j].Name()
	})
	a.dir, a.dirInput, a.entries, a.page = full, full, list, 0
	return nil
}

func (a *app) renderDialog(m *image.RGBA) {
	a.buttons = nil
	fill(m, image.Rect(88, 80, 1112, 790), color.RGBA{211, 220, 233, 255})
	fill(m, image.Rect(92, 84, 1108, 786), white)
	title := "Открыть изображение — PNG / JPEG / GIF"
	if a.dialog == "directory" {
		title = "Выбрать каталог сохранения"
	}
	a.text(m, 110, 118, 18, title, ink)
	a.field(m, image.Rect(110, 140, 925, 176), a.dirInput, 3)
	a.add(m, image.Rect(940, 140, 1088, 176), "Перейти", "browse")
	a.add(m, image.Rect(110, 190, 260, 226), "На уровень выше", "parent")
	a.add(m, image.Rect(938, 190, 1088, 226), "Отмена", "close")
	for row := 0; row < 16; row++ {
		i := a.page*16 + row
		if i >= len(a.entries) {
			break
		}
		e := a.entries[i]
		y := 240 + row*28
		r := image.Rect(110, y, 1088, y+27)
		if row%2 == 0 {
			fill(m, r, color.RGBA{238, 242, 248, 255})
		}
		label := e.Name()
		if e.IsDir() {
			label = "[каталог]  " + label
		}
		if len([]rune(label)) > 95 {
			label = string([]rune(label)[:92]) + "…"
		}
		a.text(m, 122, y+20, 16, label, ink)
		a.buttons = append(a.buttons, button{r, label, fmt.Sprint("entry-", i)})
	}
	a.add(m, image.Rect(110, 708, 240, 744), "Предыдущие", "prev")
	a.add(m, image.Rect(252, 708, 382, 744), "Следующие", "next")
	if a.dialog == "directory" {
		a.add(m, image.Rect(790, 708, 1088, 744), "Выбрать этот каталог", "choose-dir")
	}
	message := "Клик по каталогу открывает его; клик по файлу загружает изображение. Esc — отмена."
	if strings.HasPrefix(a.status, "Ошибка:") {
		lines := wrap(a.status, 120)
		if len(lines) > 0 {
			message = lines[0]
		}
	}
	a.text(m, 110, 770, 14, message, muted)
}

func (a *app) key(e xproto.KeyPressEvent) {
	index := (int(e.Detail) - int(a.minKey)) * int(a.keys.KeysymsPerKeycode)
	if index < 0 || index >= len(a.keys.Keysyms) {
		return
	}
	if e.State&xproto.ModMaskShift != 0 && a.keys.KeysymsPerKeycode > 1 {
		index++
	}
	sym := uint32(a.keys.Keysyms[index])
	if sym == 0xff1b {
		a.dialog = ""
		a.focus = 1
		return
	}
	if sym == 0xff09 {
		if a.dialog != "" {
			a.focus = 3
		} else {
			a.focus = a.focus%6 + 1
			if a.focus == 3 {
				a.focus = 4
			}
		}
		a.selectAll = true
		return
	}
	if e.State&xproto.ModMaskControl != 0 {
		if sym == 'a' || sym == 'A' {
			a.selectAll = true
		}
		return
	}
	var field *string
	switch a.focus {
	case 1:
		field = &a.size
	case 2:
		field = &a.out
	case 3:
		field = &a.dirInput
	case 4:
		field = &a.cx
	case 5:
		field = &a.cy
	case 6:
		field = &a.radius
	default:
		return
	}
	if sym == 0xff0d {
		action := "all"
		if a.dialog != "" {
			action = "browse"
		}
		if err := a.action(action); err != nil {
			a.status = "Ошибка: " + err.Error()
		}
		return
	}
	if sym == 0xff08 || sym == 0xffff {
		if a.selectAll {
			*field = ""
		} else if len(*field) > 0 {
			_, size := utf8.DecodeLastRuneInString(*field)
			*field = (*field)[:len(*field)-size]
		}
		a.selectAll = false
		return
	}
	ch := rune(0)
	if sym >= 32 && sym < 127 || sym >= 160 && sym <= 255 {
		ch = rune(sym)
	}
	if sym&0xff000000 == 0x01000000 {
		ch = rune(sym & 0xffffff)
	}
	if ch == 0 || len([]rune(*field)) >= 1000 {
		return
	}
	if a.selectAll {
		*field = ""
		a.selectAll = false
	}
	*field += string(ch)
}
