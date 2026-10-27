package desktop

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jezek/xgb/xproto"
	xdraw "golang.org/x/image/draw"
	"graphiclabs/lb6/internal/raster"
)

var ink = color.RGBA{36, 51, 73, 255}
var blue = color.RGBA{35, 91, 175, 255}
var muted = color.RGBA{87, 105, 129, 255}
var white = color.RGBA{255, 255, 255, 255}

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
	// A bounded text viewport; long paths keep their last characters visible.
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
	a.text(m, 24, 38, 26, "Геометрические преобразования", ink)
	a.text(m, 24, 64, 14, "Лабораторная 06  /  Индивидуальный номер 28  /  Вариант 08", muted)
	a.add(m, image.Rect(24, 80, 244, 116), "Открыть изображение", "open")
	a.add(m, image.Rect(256, 80, 352, 116), "Демо", "demo")
	a.text(m, 375, 104, 16, "Угол, °", ink)
	a.field(m, image.Rect(452, 80, 542, 116), a.angle, 1)
	ax := "вертикаль"
	if a.axis == "horizontal" {
		ax = "горизонталь"
	}
	a.add(m, image.Rect(562, 80, 846, 116), "Отражение: "+ax, "axis")
	a.text(m, 865, 103, 14, "Положительный угол — по часовой", muted)
	a.add(m, image.Rect(24, 130, 276, 166), "Поворот + отражение", "affine")
	a.add(m, image.Rect(288, 130, 468, 166), "Восстановить", "restore")
	a.add(m, image.Rect(480, 130, 730, 166), "Нелинейное x′^(5/3)", "nonlinear")
	a.add(m, image.Rect(742, 130, 944, 166), "Выполнить всё", "all")
	a.add(m, image.Rect(956, 130, 1176, 166), "Сохранить всё", "save-all")
	a.text(m, 24, 206, 16, "Каталог PNG", ink)
	a.field(m, image.Rect(142, 182, 952, 218), a.out, 2)
	a.add(m, image.Rect(966, 182, 1176, 218), "Выбрать каталог…", "directory")
	for i, line := range wrap(a.status, 118) {
		if i == 2 {
			break
		}
		a.text(m, 24, 245+i*20, 14, line, muted)
	}
	imgs := []*image.RGBA{a.source, a.results.Affine, a.results.Restored, a.results.Nonlinear}
	labels := []string{"1. Исходное изображение", "2. Поворот → отражение", "3. Обратное преобразование", "4. i = x′^(5/3), j = y′"}
	for i, src := range imgs {
		x, y := 24+(i%2)*584, 284+(i/2)*264
		card := image.Rect(x, y, x+568, y+248)
		fill(m, card, white)
		a.text(m, x+12, y+25, 16, labels[i], ink)
		if i > 0 {
			a.add(m, image.Rect(x+400, y+4, x+556, y+38), "Сохранить PNG", fmt.Sprint("save-", i))
		}
		r := image.Rect(x+12, y+46, x+556, y+236)
		fill(m, r, color.RGBA{231, 237, 245, 255})
		if src == nil {
			a.text(m, x+40, y+140, 16, "Результат ещё не сформирован", muted)
			continue
		}
		scale := min(float64(r.Dx())/float64(src.Rect.Dx()), float64(r.Dy())/float64(src.Rect.Dy()))
		w, h := int(float64(src.Rect.Dx())*scale), int(float64(src.Rect.Dy())*scale)
		left, top := r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2
		xdraw.NearestNeighbor.Scale(m, image.Rect(left, top, left+w, top+h), src, src.Bounds(), draw.Src, nil)
	}
	a.text(m, 24, 832, 14, "RGB · Ближайший сосед · Размер холста сохраняется · Результаты сохраняются без потерь в PNG", muted)
	if a.dialog != "" {
		a.renderDialog(m)
	}
	return m
}

func wrap(s string, n int) []string {
	var lines []string
	for _, word := range strings.Fields(s) {
		if len(lines) == 0 || len([]rune(lines[len(lines)-1]+" "+word)) > n {
			lines = append(lines, word)
		} else {
			lines[len(lines)-1] += " " + word
		}
	}
	return lines
}

func (a *app) click(x, y int, detail xproto.Button) {
	pt := image.Pt(x, y)
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
		if pt.In(image.Rect(110, 140, 925, 176)) {
			a.focus = 3
			a.selectAll = true
			return
		}
	} else {
		if pt.In(image.Rect(452, 80, 542, 116)) {
			a.focus = 1
			a.selectAll = true
			return
		}
		if pt.In(image.Rect(142, 182, 952, 218)) {
			a.focus = 2
			a.selectAll = true
			return
		}
	}
	for _, b := range a.buttons {
		if pt.In(b.rect) {
			if err := a.action(b.action); err != nil {
				a.status = "Ошибка: " + err.Error()
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
	case "demo":
		a.source = raster.Demo()
		a.sourcePath = ""
		a.results = raster.Results{}
		a.status = "Демонстрационный источник: 640×420. Изображение создано средствами Go."
	case "axis":
		if a.axis == "vertical" {
			a.axis = "horizontal"
		} else {
			a.axis = "vertical"
		}
		a.status = "Новая ось будет использована при следующем аффинном преобразовании."
	case "affine", "all":
		angle, err := strconv.ParseFloat(strings.ReplaceAll(a.angle, ",", "."), 64)
		if err != nil {
			return errors.New("введите числовой угол")
		}
		if a.source == nil {
			return errors.New("откройте исходное изображение")
		}
		if action == "all" {
			r, err := raster.Process(a.source, angle, a.axis)
			if err != nil {
				return err
			}
			a.results = r
			a.status = "Выполнены все преобразования. Восстановление рассчитано из аффинного результата."
		} else {
			m, err := raster.Variant8(a.source.Rect.Dx(), a.source.Rect.Dy(), angle, a.axis)
			if err != nil {
				return err
			}
			img, st, err := raster.Warp(a.source, m)
			if err != nil {
				return err
			}
			a.results.Matrix = m
			a.results.Affine = img
			a.results.Restored = nil
			a.results.Stats[0] = st
			a.status = fmt.Sprintf("Угол %g°. Поворот, затем отражение. Выборок вне источника: %d.", angle, st.Outside)
		}
	case "restore":
		if a.results.Affine == nil {
			return errors.New("сначала выполните поворот и отражение")
		}
		inv, err := a.results.Matrix.Inverse()
		if err != nil {
			return err
		}
		img, st, err := raster.Warp(a.results.Affine, inv)
		if err != nil {
			return err
		}
		a.results.Restored = img
		a.results.Stats[1] = st
		a.status = "Восстановлено из преобразованного растра. Округление и обрезка не позволяют вернуть все детали."
	case "nonlinear":
		img, st, err := raster.Nonlinear(a.source)
		if err != nil {
			return err
		}
		a.results.Nonlinear = img
		a.results.Stats[2] = st
		a.status = "Точная формула: i=x′^(5/3), j=y′; обратная выборка x′=i^(3/5). Координаты не нормализованы."
	case "save-all":
		for i := 1; i <= 3; i++ {
			if err := a.save(i); err != nil {
				return err
			}
		}
		a.status = "Три результата сохранены в " + a.out
	default:
		if strings.HasPrefix(action, "entry-") {
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
			a.status = fmt.Sprintf("Открыт %s, %d×%d; результаты предыдущего источника очищены.", filepath.Base(path), img.Rect.Dx(), img.Rect.Dy())
		} else if strings.HasPrefix(action, "save-") {
			i, err := strconv.Atoi(strings.TrimPrefix(action, "save-"))
			if err != nil {
				return err
			}
			return a.save(i)
		} else {
			return errors.New("неизвестное действие")
		}
	}
	return nil
}

func (a *app) save(i int) error {
	if i < 1 || i > 3 {
		return errors.New("неверный номер результата")
	}
	imgs := []*image.RGBA{a.results.Affine, a.results.Restored, a.results.Nonlinear}
	if imgs[i-1] == nil {
		return errors.New("сначала сформируйте этот результат")
	}
	if strings.TrimSpace(a.out) == "" {
		return errors.New("выберите каталог для PNG")
	}
	if err := os.MkdirAll(a.out, 0755); err != nil {
		return err
	}
	name := []string{"affine.png", "restored.png", "nonlinear.png"}[i-1]
	path := filepath.Join(a.out, name)
	if a.sourcePath != "" {
		src, _ := filepath.Abs(a.sourcePath)
		dst, _ := filepath.Abs(path)
		if src == dst {
			return errors.New("нельзя перезаписать исходный файл: выберите другой каталог")
		}
	}
	if err := raster.Save(path, imgs[i-1]); err != nil {
		return err
	}
	a.status = "Сохранено: " + path
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
			a.focus = a.focus%2 + 1
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
		field = &a.angle
	case 2:
		field = &a.out
	case 3:
		field = &a.dirInput
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
