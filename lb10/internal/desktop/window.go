// Package desktop presents the laboratory in a native X11 window.
package desktop

import (
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed assets/DejaVuSans.ttf
var fontData []byte

const frameW, frameH = 1200, 840
const windowTitle = "Lab10 - Bezier curves and surfaces"

type window struct {
	conn          *xgb.Conn
	win           xproto.Window
	gc            xproto.Gcontext
	screen        *xproto.ScreenInfo
	wmDelete      xproto.Atom
	keys          *xproto.GetKeyboardMappingReply
	minKey        xproto.Keycode
	width, height int
	faces         map[int]font.Face
}

func Run(out string) error {
	c, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("не удалось открыть окно X11: %w. Требуется сеанс X11/XWayland", err)
	}
	defer c.Close()
	setup := xproto.Setup(c)
	screen := setup.DefaultScreen(c)
	if err := checkVisual(setup, screen); err != nil {
		return err
	}
	a, err := newApp(out)
	if err != nil {
		return err
	}
	a.conn, a.screen = c, screen
	a.width = min(frameW, int(screen.WidthInPixels)-32)
	a.height = min(frameH, int(screen.HeightInPixels)-48)
	if a.width < 600 || a.height < 420 {
		return errors.New("для окна требуется экран не менее 632×468")
	}
	a.win, err = xproto.NewWindowId(c)
	if err != nil {
		return err
	}
	a.gc, err = xproto.NewGcontextId(c)
	if err != nil {
		return err
	}
	fontFile, err := opentype.Parse(fontData)
	if err != nil {
		return err
	}
	a.faces = make(map[int]font.Face, 3)
	for _, size := range []int{14, 16, 26} {
		face, err := opentype.NewFace(fontFile, &opentype.FaceOptions{
			Size: float64(size), DPI: 72, Hinting: font.HintingFull,
		})
		if err != nil {
			return err
		}
		a.faces[size] = face
		defer face.Close()
	}
	a.minKey = setup.MinKeycode
	keyCount := byte(int(setup.MaxKeycode) - int(setup.MinKeycode) + 1)
	a.keys, err = xproto.GetKeyboardMapping(c, a.minKey, keyCount).Reply()
	if err != nil {
		return err
	}
	mask := uint32(xproto.EventMaskExposure | xproto.EventMaskButtonPress |
		xproto.EventMaskButtonRelease | xproto.EventMaskButtonMotion |
		xproto.EventMaskKeyPress | xproto.EventMaskStructureNotify)
	err = xproto.CreateWindowChecked(c, screen.RootDepth, a.win, screen.Root, 0, 0,
		uint16(a.width), uint16(a.height), 0, xproto.WindowClassInputOutput,
		screen.RootVisual, xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{screen.WhitePixel, mask}).Check()
	if err != nil {
		return err
	}
	if err := xproto.CreateGCChecked(c, a.gc, xproto.Drawable(a.win), 0, nil).Check(); err != nil {
		return err
	}
	defer xproto.FreeGC(c, a.gc)
	if err := a.registerWindow(); err != nil {
		return err
	}
	if err := xproto.MapWindowChecked(c, a.win).Check(); err != nil {
		return err
	}
	return a.events()
}

func checkVisual(setup *xproto.SetupInfo, screen *xproto.ScreenInfo) error {
	valid := false
	for _, depth := range screen.AllowedDepths {
		for _, v := range depth.Visuals {
			if v.VisualId == screen.RootVisual && v.RedMask == 0xff0000 &&
				v.GreenMask == 0xff00 && v.BlueMask == 0xff {
				valid = true
			}
		}
	}
	for _, format := range setup.PixmapFormats {
		if valid && format.Depth == screen.RootDepth && format.BitsPerPixel == 32 &&
			setup.ImageByteOrder == xproto.ImageOrderLSBFirst {
			return nil
		}
	}
	return errors.New("нужен X11 TrueColor с 32-битными пикселями LSBFirst")
}

func (w *window) registerWindow() error {
	c := w.conn
	xproto.ChangeProperty(c, xproto.PropModeReplace, w.win,
		xproto.AtomWmName, xproto.AtomString, 8, uint32(len(windowTitle)), []byte(windowTitle))
	protocols, err := xproto.InternAtom(c, false, 12, "WM_PROTOCOLS").Reply()
	if err != nil {
		return err
	}
	closeAtom, err := xproto.InternAtom(c, false, 16, "WM_DELETE_WINDOW").Reply()
	if err != nil {
		return err
	}
	w.wmDelete = closeAtom.Atom
	data := make([]byte, 4)
	xgb.Put32(data, uint32(w.wmDelete))
	xproto.ChangeProperty(c, xproto.PropModeReplace, w.win,
		protocols.Atom, xproto.AtomAtom, 32, 1, data)
	return nil
}

func (a *app) events() error {
	if err := a.paint(); err != nil {
		return err
	}
	for {
		event, err := a.conn.WaitForEvent()
		if err != nil || event == nil {
			return err
		}
		switch ev := event.(type) {
		case xproto.ClientMessageEvent:
			if ev.Format == 32 && ev.Data.Data32[0] == uint32(a.wmDelete) {
				return nil
			}
			continue
		case xproto.DestroyNotifyEvent:
			return nil
		case xproto.ConfigureNotifyEvent:
			a.width, a.height = int(ev.Width), int(ev.Height)
		case xproto.ExposeEvent:
			if ev.Count != 0 {
				continue
			}
		case xproto.ButtonPressEvent:
			a.click(a.toFrame(ev.EventX, ev.EventY), ev.Detail)
		case xproto.ButtonReleaseEvent:
			if ev.Detail == 1 || ev.Detail == 3 {
				a.dragging, a.rotating = false, false
			}
			continue
		case xproto.MotionNotifyEvent:
			if !a.dragging && !a.rotating {
				continue
			}
			a.dragTo(a.toFrame(ev.EventX, ev.EventY))
		case xproto.KeyPressEvent:
			a.key(ev)
		default:
			continue
		}
		if err := a.paint(); err != nil {
			return err
		}
	}
}

func (w *window) viewport() image.Rectangle {
	scale := min(float64(w.width)/frameW, float64(w.height)/frameH)
	width, height := int(frameW*scale), int(frameH*scale)
	x, y := (w.width-width)/2, (w.height-height)/2
	return image.Rect(x, y, x+width, y+height)
}

func (w *window) toFrame(x, y int16) image.Point {
	r := w.viewport()
	if r.Empty() {
		return image.Pt(-1, -1)
	}
	return image.Pt((int(x)-r.Min.X)*frameW/r.Dx(), (int(y)-r.Min.Y)*frameH/r.Dy())
}

func (a *app) paint() error {
	if a.width < 1 || a.height < 1 || a.width > 8192 || a.height > 8192 {
		return nil
	}
	canvas := a.render()
	view := image.NewRGBA(image.Rect(0, 0, a.width, a.height))
	fill(view, view.Bounds(), base)
	xdraw.ApproxBiLinear.Scale(view, a.viewport(), canvas, canvas.Bounds(), draw.Src, nil)
	// X11 PutImage uses B, G, R, padding. Stay below the X11 request size.
	rows := max(1, (int(xproto.Setup(a.conn).MaximumRequestLength)*4-64)/view.Stride)
	buffer := make([]byte, min(a.height, rows)*view.Stride)
	for y := 0; y < a.height; y += rows {
		count := min(rows, a.height-y)
		src := view.Pix[y*view.Stride : (y+count)*view.Stride]
		dst := buffer[:len(src)]
		for j := 0; j < len(src); j += 4 {
			dst[j], dst[j+1], dst[j+2] = src[j+2], src[j+1], src[j]
		}
		err := xproto.PutImageChecked(a.conn, xproto.ImageFormatZPixmap,
			xproto.Drawable(a.win), a.gc, uint16(a.width), uint16(count),
			0, int16(y), 0, a.screen.RootDepth, dst).Check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *window) symbol(ev xproto.KeyPressEvent) xproto.Keysym {
	count := int(w.keys.KeysymsPerKeycode)
	index := (int(ev.Detail) - int(w.minKey)) * count
	if index < 0 || index+count > len(w.keys.Keysyms) || count == 0 {
		return 0
	}
	base := w.keys.Keysyms[index]
	if count > 1 {
		alt := w.keys.Keysyms[index+1]
		if alt != 0 && (ev.State&xproto.KeyButMaskShift != 0 || alt >= 0xffae && alt <= 0xffb9) {
			return alt
		}
	}
	return base
}

func fill(m *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(m, r, image.NewUniform(c), image.Point{}, draw.Src)
}

func (w *window) text(m *image.RGBA, x, y, size int, s string, c color.RGBA) {
	d := font.Drawer{Dst: m, Src: image.NewUniform(c), Face: w.faces[size], Dot: fixed.P(x, y)}
	d.DrawString(s)
}

func (w *window) textWidth(s string, size int) int {
	return font.MeasureString(w.faces[size], s).Ceil()
}

func (w *window) fitText(s string, size, width int) string {
	if w.textWidth(s, size) <= width {
		return s
	}
	runes := []rune(s)
	for n := len(runes) - 1; n > 0; n-- {
		cut := string(runes[:n]) + "…"
		if w.textWidth(cut, size) <= width {
			return cut
		}
	}
	return "…"
}
