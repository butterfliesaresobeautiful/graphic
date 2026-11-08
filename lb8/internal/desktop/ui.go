package desktop

import (
	"errors"
	"fmt"
	"github.com/jezek/xgb/xproto"
	xdraw "golang.org/x/image/draw"
	"graphiclabs/lb8/internal/imaging"
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
var fields = map[int]image.Rectangle{
	1: image.Rect(100, 130, 200, 166), 4: image.Rect(300, 130, 400, 166),
	5: image.Rect(540, 130, 650, 166), 6: image.Rect(790, 130, 890, 166), 7: image.Rect(1010, 130, 1110, 166),
	2: image.Rect(142, 182, 850, 218),
}

func (a *app) add(m *image.RGBA, r image.Rectangle, label, action string) {
	fill(m, r, blue)
	a.text(m, r.Min.X+12, r.Min.Y+24, 16, label, white)
	a.buttons = append(a.buttons, button{r, label, action})
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
	a.text(m, 24, 38, 26, "Интерполяция изображений", ink)
	a.text(m, 24, 64, 14, "ЛБ8 / Номер 28 / Общее задание / Масштаб → скос → поворот / 3 метода", muted)
	a.add(m, image.Rect(24, 80, 200, 116), "Открыть файл…", "open")
	a.add(m, image.Rect(212, 80, 320, 116), "Демо", "demo")
	a.add(m, image.Rect(332, 80, 660, 116), fmt.Sprintf("Набор %d/8: следующий →", a.preset+1), "preset")
	a.add(m, image.Rect(672, 80, 920, 116), "Преобразовать", "all")
	a.add(m, image.Rect(932, 80, 1176, 116), "Сохранить всё", "save")
	for _, f := range []struct {
		id, x        int
		label, value string
	}{{1, 24, "Sx", a.sx}, {4, 240, "Sy", a.sy}, {5, 438, "Угол °", a.angle}, {6, 710, "hx", a.hx}, {7, 950, "hy", a.hy}} {
		a.text(m, f.x, 154, 16, f.label, ink)
		a.field(m, fields[f.id], f.value, f.id)
	}
	a.text(m, 24, 206, 16, "Каталог PNG", ink)
	a.field(m, fields[2], a.out, 2)
	a.add(m, image.Rect(862, 182, 1040, 218), "Выбрать папку…", "directory")
	v := "Пиксели"
	if a.zoom {
		v = "Целиком"
	}
	a.add(m, image.Rect(1052, 182, 1176, 218), v, "view")
	for i, s := range wrap(a.status, 118) {
		if i == 2 {
			break
		}
		a.text(m, 24, 244+i*19, 14, s, muted)
	}
	imgs := []*image.RGBA{a.source, nil, nil, nil}
	labels := []string{"Источник", "Ближайший сосед", "Билинейная", "Бикубическая (Лагранж)"}
	if a.result != nil {
		copy(imgs[1:], a.result.Images[:])
	}
	for i, src := range imgs {
		x, y := 24+(i%2)*584, 280+(i/2)*264
		fill(m, image.Rect(x, y, x+568, y+248), white)
		a.text(m, x+12, y+23, 16, labels[i], ink)
		box := image.Rect(x+12, y+32, x+556, y+217)
		fill(m, box, color.RGBA{231, 237, 245, 255})
		if src == nil {
			a.text(m, x+24, y+105, 16, "Нажмите «Преобразовать»", muted)
			continue
		}
		region := src.Bounds()
		if a.zoom {
			cx, cy := (region.Min.X+region.Max.X)/2, (region.Min.Y+region.Max.Y)/2
			region = image.Rect(cx-48, cy-36, cx+48, cy+36).Intersect(region)
		}
		scale := min(float64(box.Dx())/float64(region.Dx()), float64(box.Dy())/float64(region.Dy()))
		w, h := int(float64(region.Dx())*scale), int(float64(region.Dy())*scale)
		l, t := box.Min.X+(box.Dx()-w)/2, box.Min.Y+(box.Dy()-h)/2
		// Nearest preview preserves differences between interpolation methods.
		xdraw.NearestNeighbor.Scale(m, image.Rect(l, t, l+w, t+h), src, region, draw.Src, nil)
		label := fmt.Sprintf("%d×%d", src.Rect.Dx(), src.Rect.Dy())
		if a.zoom {
			label += fmt.Sprintf(" / ROI [%d,%d; %d,%d)", region.Min.X, region.Min.Y, region.Max.X, region.Max.Y)
		}
		if i >= 2 && a.result != nil {
			d := a.result.Metrics.Differences[i-2]
			label += fmt.Sprintf(" / MAE к соседу %.3f", d.MAE)
		}
		a.text(m, x+12, y+239, 14, label, muted)
	}
	a.text(m, 24, 830, 14, "Параметры применяются кнопкой «Преобразовать». MAE — различие методов, а не ошибка относительно эталона.", muted)
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
func (a *app) setConfig(c imaging.Config) {
	a.sx = fmt.Sprint(c.ScaleX)
	a.sy = fmt.Sprint(c.ScaleY)
	a.angle = fmt.Sprint(c.Angle)
	a.hx = fmt.Sprint(c.ShearX)
	a.hy = fmt.Sprint(c.ShearY)
}
func (a *app) parameters() (imaging.Config, error) {
	c := imaging.Config{}
	for _, f := range []struct {
		s string
		p *float64
	}{{a.sx, &c.ScaleX}, {a.sy, &c.ScaleY}, {a.angle, &c.Angle}, {a.hx, &c.ShearX}, {a.hy, &c.ShearY}} {
		v, e := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(f.s), ",", "."), 64)
		if e != nil {
			return c, errors.New("параметры должны быть числами")
		}
		*f.p = v
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
		a.result = nil
		a.status = "Демонстрационный рисунок 480×360. Нажмите «Преобразовать»."
	case "preset":
		a.preset = (a.preset + 1) % len(imaging.Presets())
		p := imaging.Presets()[a.preset]
		a.setConfig(p.Config)
		a.status = p.Title + ". Для применения нажмите «Преобразовать»."
	case "view":
		a.zoom = !a.zoom
	case "all":
		c, e := a.parameters()
		if e != nil {
			return e
		}
		r, e := imaging.Transform(a.source, c)
		if e != nil {
			return e
		}
		a.result = r
		a.status = fmt.Sprintf("Готово: %d×%d; Sx=%g, Sy=%g, угол=%g°, hx=%g, hy=%g. Ограничено каналов кубики: %d.", r.Metrics.Geometry.Width, r.Metrics.Geometry.Height, c.ScaleX, c.ScaleY, c.Angle, c.ShearX, c.ShearY, r.Metrics.ClippedChannels[2])
	case "save":
		if e := a.result.Save(a.out, a.sourcePath); e != nil {
			return e
		}
		a.status = "Сохранены источник, 3 PNG и metrics.json: " + a.out
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
		a.result = nil
		a.dialog = ""
		a.focus = 1
		a.status = fmt.Sprintf("Загружен %s, %d×%d. Нажмите «Преобразовать».", filepath.Base(path), src.Rect.Dx(), src.Rect.Dy())
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
			order := []int{1, 4, 5, 6, 7, 2}
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
		field = &a.sx
	case 2:
		field = &a.out
	case 3:
		field = &a.dirInput
	case 4:
		field = &a.sy
	case 5:
		field = &a.angle
	case 6:
		field = &a.hx
	case 7:
		field = &a.hy
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
