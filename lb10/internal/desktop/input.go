package desktop

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/jezek/xgb/xproto"
	"graphiclabs/lb10/internal/bezier"
)

func (a *app) click(in image.Point, button xproto.Button) {
	if button == 4 || button == 5 {
		if in.In(plotBox) {
			cam := a.camera()
			factor := 1.18
			if button == 5 {
				factor = 1 / factor
			}
			cam.Zoom = max(0.25, min(5, cam.Zoom*factor))
		}
		return
	}
	if button != 1 && button != 3 {
		return
	}
	if !a.finishEdit() {
		return
	}
	if button == 1 {
		for _, field := range a.inputs {
			if in.In(field.rect) {
				a.edit = editor{field.id, a.currentValue(field.id), true}
				return
			}
		}
		for _, b := range a.buttons {
			if in.In(b.rect) {
				b.action()
				return
			}
		}
	}
	if !in.In(plotBox) {
		return
	}
	if button == 1 {
		if index := a.pick(in); index >= 0 {
			a.selectVertex(index)
			a.dragging = a.mode != surface
			return
		}
	}
	if a.mode == surface {
		a.rotating, a.lastMouse = true, in
	} else if button == 1 {
		a.addCurvePoint(in)
	}
}

func (a *app) currentValue(id inputID) string {
	// A prior edit may have changed the grid and its selected vertex.
	p := a.vertex()
	s := &a.surface
	var value float64
	switch id {
	case inputX:
		value = p.X
	case inputY:
		value = p.Y
	case inputZ:
		value = p.Z
	case inputU:
		value = s.u
	case inputV:
		value = s.v
	case inputNV:
		value = float64(s.nv)
	case inputCols:
		value = float64(len(s.control[0]))
	case inputRows:
		value = float64(len(s.control))
	case inputT:
		value = a.curves[a.mode].t
	case inputNU:
		value = float64(s.nu)
		if a.mode != surface {
			value = float64(a.curves[a.mode].divisions)
		}
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func (a *app) finishEdit() bool {
	if a.edit.focus == noInput {
		return true
	}
	if a.edit.text == a.currentValue(a.edit.focus) {
		a.edit = editor{}
		a.status = ""
		return true
	}
	err := a.apply(a.edit.focus, a.edit.text)
	a.result(err)
	if err != nil {
		return false
	}
	a.edit = editor{}
	return true
}

func (a *app) apply(id inputID, entry string) error {
	value, err := strconv.ParseFloat(strings.ReplaceAll(entry, ",", "."), 64)
	if err != nil {
		return fmt.Errorf("введите число и нажмите Enter")
	}
	switch id {
	case inputX, inputY, inputZ:
		if !(value >= -bezier.CoordinateLimit && value <= bezier.CoordinateLimit) {
			return fmt.Errorf("координаты должны быть в пределах ±%d", bezier.CoordinateLimit)
		}
		p := a.vertex()
		switch id {
		case inputX:
			p.X = value
		case inputY:
			p.Y = value
		case inputZ:
			p.Z = value
		}
		return a.rebuild()
	case inputT, inputU, inputV:
		if !bezier.ValidParameter(value) {
			return fmt.Errorf("параметр разбиения должен быть в диапазоне [0, 1]")
		}
		switch id {
		case inputT:
			c := &a.curves[a.mode]
			c.t = value
			c.inspect()
			return nil
		case inputU:
			a.surface.u = value
		case inputV:
			a.surface.v = value
		}
	case inputNU, inputNV:
		n, err := integer(entry, 1, bezier.MaxDivisions)
		if err != nil {
			return err
		}
		switch {
		case a.mode != surface:
			a.curves[a.mode].divisions = n
		case id == inputNU:
			a.surface.nu = n
		default:
			a.surface.nv = n
		}
	case inputCols, inputRows:
		n, err := integer(entry, 2, bezier.MaxGridSide)
		if err != nil {
			return err
		}
		s := &a.surface
		cols, rows := len(s.control[0]), len(s.control)
		x, y := s.selected%cols, s.selected/cols
		if id == inputCols {
			cols = n
		} else {
			rows = n
		}
		s.control = bezier.ResizeGrid(s.control, cols, rows)
		s.selected = min(y, rows-1)*cols + min(x, cols-1)
		a.fit()
	}
	return a.rebuild()
}

func integer(entry string, lower, upper int) (int, error) {
	n, err := strconv.Atoi(entry)
	if err != nil || n < lower || n > upper {
		return 0, fmt.Errorf("введите целое число от %d до %d", lower, upper)
	}
	return n, nil
}

func (a *app) rebuild() error {
	if a.mode == surface {
		return a.surface.rebuild()
	}
	return a.curves[a.mode].rebuild()
}

func (a *app) key(ev xproto.KeyPressEvent) {
	sym := a.symbol(ev)
	if sym == 0xff1b { // Escape
		a.edit, a.status = editor{}, ""
		return
	}
	if sym == 0xff09 || sym == 0xfe20 { // Tab and Shift+Tab
		index := -1
		for i, field := range a.inputs {
			if field.id == a.edit.focus {
				index = i
				break
			}
		}
		if a.finishEdit() {
			n := len(a.inputs)
			index++
			if ev.State&xproto.KeyButMaskShift != 0 {
				index -= 2
				if index < -1 {
					index = -1
				}
			}
			field := a.inputs[(index+n)%n]
			a.edit = editor{field.id, a.currentValue(field.id), true}
		}
		return
	}
	if a.edit.focus == noInput {
		if sym == 0xffff && a.mode != surface {
			a.deleteVertex()
		}
		return
	}
	if sym == 0xff0d || sym == 0xff8d { // Enter and keypad Enter
		a.finishEdit()
		return
	}
	if ev.State&xproto.KeyButMaskControl != 0 {
		if sym == 'a' || sym == 'A' || sym == 0x06c6 || sym == 0x06e6 {
			a.edit.replace = true
		}
		return
	}
	if sym == 0xff08 || sym == 0xffff {
		if a.edit.replace {
			a.edit.text = ""
		} else if len(a.edit.text) != 0 {
			a.edit.text = a.edit.text[:len(a.edit.text)-1]
		}
		a.edit.replace = false
		return
	}
	if sym >= 0xffb0 && sym <= 0xffb9 {
		sym = '0' + sym - 0xffb0
	}
	switch sym {
	case 0xffae:
		sym = '.'
	case 0xffad:
		sym = '-'
	case 0xffab:
		sym = '+'
	}
	if (sym < '0' || sym > '9') && !strings.ContainsRune(".,+-eE", rune(sym)) {
		return
	}
	if a.edit.replace {
		a.edit.text, a.edit.replace = "", false
	}
	if len(a.edit.text) < 30 {
		a.edit.text += string(rune(sym))
	}
}
