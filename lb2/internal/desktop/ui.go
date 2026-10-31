package desktop

import (
	"errors"
	"fmt"
	"github.com/jezek/xgb/xproto"
	xdraw "golang.org/x/image/draw"
	"graphiclabs/lb2/internal/imaging"
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
var fields = map[int]image.Rectangle{1: image.Rect(100, 130, 180, 166), 4: image.Rect(270, 130, 350, 166), 5: image.Rect(400, 130, 470, 166), 6: image.Rect(560, 130, 635, 166), 7: image.Rect(675, 130, 750, 166), 8: image.Rect(860, 130, 935, 166), 9: image.Rect(980, 130, 1055, 166), 2: image.Rect(142, 232, 952, 268)}

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
	if a.auto && id >= 6 {
		fill(m, r.Inset(2), color.RGBA{230, 235, 242, 255})
	}
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
	a.text(m, 24, 38, 26, "Программа для формирования изображений", ink)
	a.text(m, 24, 64, 14, "ЛБ2 / 28 % 20 = 8 / Круг: снизу слева → сверху по центру / Функция y = x³", muted)
	a.add(m, image.Rect(24, 80, 200, 116), "Создать холст", "new")
	a.add(m, image.Rect(212, 80, 400, 116), "Открыть файл…", "open")
	a.add(m, image.Rect(412, 80, 530, 116), "Демо", "demo")
	mode := "вариант 8"
	if !a.auto {
		mode = "ручные"
	}
	a.add(m, image.Rect(542, 80, 850, 116), "Позиции: "+mode, "positions")
	a.add(m, image.Rect(862, 80, 1176, 116), "Выполнить всё", "all")
	for _, f := range []struct {
		id, x        int
		label, value string
	}{{1, 24, "Ширина", a.widthText}, {4, 200, "Высота", a.heightText}, {5, 370, "R", a.radius}, {6, 490, "Ист. X", a.sx}, {7, 655, "Y", a.sy}, {8, 770, "Новый X", a.dx}, {9, 955, "Y", a.dy}} {
		a.text(m, f.x, 154, 16, f.label, ink)
		a.field(m, fields[f.id], f.value, f.id)
	}
	a.add(m, image.Rect(24, 182, 246, 218), "Перенести круг", "copy")
	a.add(m, image.Rect(258, 182, 438, 218), "Добавить оси", "axes")
	a.add(m, image.Rect(450, 182, 690, 218), "Построить y = x³", "plot")
	a.add(m, image.Rect(702, 182, 912, 218), "Сохранить PNG", "save")
	view := "Источник и результат"
	if a.view == 1 {
		view = "Этапы обработки"
	}
	a.add(m, image.Rect(924, 182, 1176, 218), view, "view")
	a.text(m, 24, 256, 16, "Каталог PNG", ink)
	a.field(m, fields[2], a.out, 2)
	a.add(m, image.Rect(966, 232, 1176, 268), "Выбрать каталог…", "directory")
	for i, s := range wrap(a.status, 118) {
		if i == 2 {
			break
		}
		a.text(m, 24, 294+i*19, 14, s, muted)
	}
	imgs := []*image.RGBA{a.source, a.document.Current()}
	labels := []string{"Исходное изображение", "Результат текущего этапа"}
	height := 480
	if a.view == 1 {
		imgs = a.document.Images[:]
		labels = []string{"1. Белый холст", "2. Круг перенесён", "3. Добавлены оси", "4. График y = x³"}
		height = 232
	}
	for i, src := range imgs {
		x, y := 24+(i%2)*584, 320+(i/2)*248
		fill(m, image.Rect(x, y, x+568, y+height), white)
		a.text(m, x+12, y+25, 16, labels[i], ink)
		r := image.Rect(x+12, y+40, x+556, y+height-12)
		fill(m, r, color.RGBA{231, 237, 245, 255})
		if src == nil {
			a.text(m, x+32, y+100, 16, "Этап ещё не выполнен", muted)
			continue
		}
		scale := min(float64(r.Dx())/float64(src.Rect.Dx()), float64(r.Dy())/float64(src.Rect.Dy()))
		w, h := int(float64(src.Rect.Dx())*scale), int(float64(src.Rect.Dy())*scale)
		l, t := r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2
		if scale < 1 {
			xdraw.CatmullRom.Scale(m, image.Rect(l, t, l+w, t+h), src, src.Rect, draw.Src, nil)
		} else {
			xdraw.NearestNeighbor.Scale(m, image.Rect(l, t, l+w, t+h), src, src.Rect, draw.Src, nil)
		}
	}
	a.text(m, 24, 830, 14, "RGB копируются точно. Изменённые параметры применяются при создании / переносе. Сохраняется текущий этап.", muted)
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
func (a *app) syncPositions() {
	if !a.auto || a.source == nil {
		return
	}
	r, e := strconv.Atoi(a.radius)
	w, f := strconv.Atoi(a.widthText)
	if e != nil || f != nil {
		return
	}
	a.sx = fmt.Sprint(r)
	a.sy = fmt.Sprint(a.source.Rect.Dy() - 1 - r)
	a.dx = fmt.Sprint(w / 2)
	a.dy = fmt.Sprint(r)
}
func (a *app) parameters() (imaging.Config, error) {
	c := imaging.Config{Auto: a.auto}
	items := []struct {
		s string
		p *int
	}{{a.widthText, &c.Width}, {a.heightText, &c.Height}, {a.radius, &c.Radius}}
	if !a.auto {
		items = append(items, struct {
			s string
			p *int
		}{a.sx, &c.SX}, struct {
			s string
			p *int
		}{a.sy, &c.SY}, struct {
			s string
			p *int
		}{a.dx, &c.DX}, struct {
			s string
			p *int
		}{a.dy, &c.DY})
	}
	for _, f := range items {
		n, e := strconv.Atoi(strings.TrimSpace(f.s))
		if e != nil {
			return c, errors.New("параметры должны быть целыми числами")
		}
		*f.p = n
	}
	return c, c.Validate()
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
				if a.auto && id >= 6 {
					a.status = "Координаты вычисляются по варианту 8. Для их изменения выберите ручные позиции."
					return
				}
				a.focus = id
				a.selectAll = true
				return
			}
		}
	}
	for _, b := range a.buttons {
		if p.In(b.rect) {
			if e := a.action(b.action); e != nil {
				a.status = "Ошибка: " + e.Error()
			}
			return
		}
	}
}
func (a *app) action(action string) error {
	switch action {
	case "open", "directory":
		a.dialog = action
		a.page = 0
		a.focus = 3
		dir, e := os.Getwd()
		if e != nil {
			return e
		}
		if action == "directory" && a.out != "" {
			dir = a.out
		}
		if action == "open" && a.sourcePath != "" {
			dir = filepath.Dir(a.sourcePath)
		}
		if e = a.readDir(dir); e != nil {
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
	case "demo":
		a.source = imaging.Demo()
		a.sourcePath = ""
		a.document.SetSource(a.source)
		a.syncPositions()
		a.status = "Демонстрационный источник 800×600. Создайте холст или выполните всё."
	case "positions":
		a.auto = !a.auto
		a.focus = 1
		a.syncPositions()
		a.status = "Новые позиции применятся при следующем переносе. Ручные позиции могут отличаться от варианта 8."
	case "view":
		a.view = 1 - a.view
	case "new", "copy", "all":
		c, e := a.parameters()
		if e != nil {
			return e
		}
		switch action {
		case "new":
			if e = a.document.Create(c); e != nil {
				return e
			}
			a.status = "Создан белый холст. Следующий этап — перенос круга."
		case "copy":
			if e = a.document.Copy(c); e != nil {
				return e
			}
			a.status = fmt.Sprintf("Круг перенесён: %d пикселей; отсечено %d. Следующий этап — оси.", a.document.Stats.Copied, a.document.Stats.Clipped)
		case "all":
			d, e := imaging.Build(a.source, c)
			if e != nil {
				return e
			}
			a.document = d
			a.status = fmt.Sprintf("Все этапы выполнены. Скопировано %d RGB-пикселей; отсечено %d.", d.Stats.Copied, d.Stats.Clipped)
		}
		a.syncPositions()
	case "axes":
		if e := a.document.AddAxes(); e != nil {
			return e
		}
		a.status = "Оси добавлены с параметрами выполненного переноса. Следующий этап — график."
	case "plot":
		if e := a.document.AddGraph(); e != nil {
			return e
		}
		a.status = "Построено y=x³: x от −2 до 2, y от −8 до 8. Можно сохранить PNG."
	case "save":
		if e := a.document.SaveCurrent(a.out, a.sourcePath); e != nil {
			return e
		}
		a.status = "Сохранён текущий этап: " + filepath.Join(a.out, "result.png")
	default:
		if !strings.HasPrefix(action, "entry-") {
			return errors.New("неизвестное действие")
		}
		i, e := strconv.Atoi(strings.TrimPrefix(action, "entry-"))
		if e != nil || i < 0 || i >= len(a.entries) {
			return errors.New("неверный файл")
		}
		path := filepath.Join(a.dir, a.entries[i].Name())
		if a.entries[i].IsDir() {
			return a.readDir(path)
		}
		src, e := imaging.Load(path)
		if e != nil {
			return e
		}
		a.source = src
		a.sourcePath = path
		a.document.SetSource(src)
		a.dialog = ""
		a.focus = 1
		a.syncPositions()
		a.status = fmt.Sprintf("Загружен %s, %d×%d. Создайте холст или выполните всё.", filepath.Base(path), src.Rect.Dx(), src.Rect.Dy())
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
			order := []int{1, 4, 5, 6, 7, 8, 9, 2}
			if a.auto {
				order = []int{1, 4, 5, 2}
			}
			next := order[0]
			for i, v := range order {
				if v == a.focus {
					next = order[(i+1)%len(order)]
					break
				}
			}
			a.focus = next
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
		field = &a.widthText
	case 2:
		field = &a.out
	case 3:
		field = &a.dirInput
	case 4:
		field = &a.heightText
	case 5:
		field = &a.radius
	case 6:
		field = &a.sx
	case 7:
		field = &a.sy
	case 8:
		field = &a.dx
	case 9:
		field = &a.dy
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
	a.syncPositions()
}
