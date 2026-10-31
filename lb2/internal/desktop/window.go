// Package desktop is a native X11 client written in Go, without CGO.
package desktop

import (
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"graphiclabs/lb2/internal/imaging"
)

//go:embed assets/DejaVuSans.ttf
var fontData []byte

const frameW, frameH = 1200, 840

type button struct {
	rect          image.Rectangle
	label, action string
}
type app struct {
	conn                                                                   *xgb.Conn
	win                                                                    xproto.Window
	gc                                                                     xproto.Gcontext
	screen                                                                 *xproto.ScreenInfo
	wmDelete                                                               xproto.Atom
	keys                                                                   *xproto.GetKeyboardMappingReply
	minKey                                                                 xproto.Keycode
	width, height                                                          int
	faces                                                                  map[int]font.Face
	buttons                                                                []button
	source                                                                 *image.RGBA
	document                                                               imaging.Document
	sourcePath, out, widthText, heightText, radius, sx, sy, dx, dy, status string
	auto                                                                   bool
	view                                                                   int
	focus                                                                  int
	selectAll                                                              bool
	dialog, dir, dirInput                                                  string
	entries                                                                []os.DirEntry
	page                                                                   int
}

func Run(src *image.RGBA, sourcePath, out string, cfg imaging.Config) error {
	c, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("не удалось открыть окно X11: %w; требуется графический сеанс X11/XWayland", err)
	}
	defer c.Close()
	setup := xproto.Setup(c)
	screen := setup.DefaultScreen(c)
	valid := false
	for _, depth := range screen.AllowedDepths {
		for _, v := range depth.Visuals {
			if v.VisualId == screen.RootVisual && v.RedMask == 0xff0000 && v.GreenMask == 0xff00 && v.BlueMask == 0xff {
				valid = true
			}
		}
	}
	bpp := byte(0)
	for _, f := range setup.PixmapFormats {
		if f.Depth == screen.RootDepth {
			bpp = f.BitsPerPixel
		}
	}
	if !valid || bpp != 32 || setup.ImageByteOrder != xproto.ImageOrderLSBFirst {
		return errors.New("нужен X11 TrueColor с 32-битными пикселями LSBFirst (обычный Linux desktop)")
	}
	win, err := xproto.NewWindowId(c)
	if err != nil {
		return err
	}
	gc, err := xproto.NewGcontextId(c)
	if err != nil {
		return err
	}
	a := &app{conn: c, win: win, gc: gc, screen: screen, source: src, sourcePath: sourcePath,
		out: out, widthText: fmt.Sprint(cfg.Width), heightText: fmt.Sprint(cfg.Height), radius: fmt.Sprint(cfg.Radius),
		sx: fmt.Sprint(cfg.SX), sy: fmt.Sprint(cfg.SY), dx: fmt.Sprint(cfg.DX), dy: fmt.Sprint(cfg.DY), auto: cfg.Auto, faces: map[int]font.Face{},
		status: "Создайте холст и перенесите круг или нажмите «Выполнить всё».", focus: 1,
		width: min(frameW, int(screen.WidthInPixels)-40), height: min(frameH, int(screen.HeightInPixels)-80)}
	a.document.Source = src
	a.syncPositions()
	if a.width < 480 || a.height < 400 {
		return errors.New("для окна требуется экран не менее 520×480")
	}
	fontFile, err := opentype.Parse(fontData)
	if err != nil {
		return err
	}
	for _, size := range []int{14, 16, 18, 26} {
		face, e := opentype.NewFace(fontFile, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
		if e != nil {
			return e
		}
		a.faces[size] = face
		defer face.Close()
	}
	a.minKey = setup.MinKeycode
	a.keys, err = xproto.GetKeyboardMapping(c, setup.MinKeycode, byte(int(setup.MaxKeycode)-int(setup.MinKeycode)+1)).Reply()
	if err != nil {
		return err
	}
	mask := uint32(xproto.EventMaskExposure | xproto.EventMaskButtonPress | xproto.EventMaskKeyPress | xproto.EventMaskStructureNotify)
	err = xproto.CreateWindowChecked(c, screen.RootDepth, win, screen.Root, 0, 0, uint16(a.width), uint16(a.height), 0,
		xproto.WindowClassInputOutput, screen.RootVisual, xproto.CwBackPixel|xproto.CwEventMask, []uint32{screen.WhitePixel, mask}).Check()
	if err != nil {
		return err
	}
	defer xproto.DestroyWindow(c, win)
	if err = xproto.CreateGCChecked(c, gc, xproto.Drawable(win), 0, nil).Check(); err != nil {
		return err
	}
	defer xproto.FreeGC(c, gc)
	title := "Lab2 - Image composition - Variant 8"
	xproto.ChangeProperty(c, xproto.PropModeReplace, win, xproto.AtomWmName, xproto.AtomString, 8, uint32(len(title)), []byte(title))
	protocols, e := xproto.InternAtom(c, false, 12, "WM_PROTOCOLS").Reply()
	if e != nil {
		return e
	}
	closeAtom, e := xproto.InternAtom(c, false, 16, "WM_DELETE_WINDOW").Reply()
	if e != nil {
		return e
	}
	a.wmDelete = closeAtom.Atom
	data := make([]byte, 4)
	xgb.Put32(data, uint32(closeAtom.Atom))
	xproto.ChangeProperty(c, xproto.PropModeReplace, win, protocols.Atom, xproto.AtomAtom, 32, 1, data)
	if err = xproto.MapWindowChecked(c, win).Check(); err != nil {
		return err
	}
	if err = a.paint(); err != nil {
		return err
	}
	for {
		event, e := c.WaitForEvent()
		if e != nil {
			return e
		}
		if event == nil {
			return nil
		}
		switch ev := event.(type) {
		case xproto.ClientMessageEvent:
			if ev.Data.Data32[0] == uint32(a.wmDelete) {
				return nil
			}
		case xproto.DestroyNotifyEvent:
			return nil
		case xproto.ConfigureNotifyEvent:
			a.width, a.height = int(ev.Width), int(ev.Height)
		case xproto.ExposeEvent:
			if ev.Count != 0 {
				continue
			}
		case xproto.ButtonPressEvent:
			x, y := a.toFrame(int(ev.EventX), int(ev.EventY))
			a.click(x, y, ev.Detail)
		case xproto.KeyPressEvent:
			a.key(ev)
		default:
			continue
		}
		if err = a.paint(); err != nil {
			return err
		}
	}
}

func (a *app) viewport() image.Rectangle {
	scale := min(float64(a.width)/frameW, float64(a.height)/frameH)
	w, h := int(frameW*scale), int(frameH*scale)
	x, y := (a.width-w)/2, (a.height-h)/2
	return image.Rect(x, y, x+w, y+h)
}
func (a *app) toFrame(x, y int) (int, int) {
	r := a.viewport()
	if r.Empty() {
		return -1, -1
	}
	return (x - r.Min.X) * frameW / r.Dx(), (y - r.Min.Y) * frameH / r.Dy()
}

func (a *app) paint() error {
	if a.width < 1 || a.height < 1 || a.width > 8192 || a.height > 8192 {
		return nil
	}
	canvas := a.render()
	view := image.NewRGBA(image.Rect(0, 0, a.width, a.height))
	draw.Draw(view, view.Bounds(), image.NewUniform(color.RGBA{231, 236, 243, 255}), image.Point{}, draw.Src)
	xdraw.ApproxBiLinear.Scale(view, a.viewport(), canvas, canvas.Bounds(), draw.Src, nil)
	// X11 PutImage uses B,G,R,pad. Chunk rows to stay below request size limits.
	rows := max(1, (int(xproto.Setup(a.conn).MaximumRequestLength)*4-64)/(a.width*4))
	for y := 0; y < a.height; y += rows {
		n := min(rows, a.height-y)
		buf := make([]byte, a.width*n*4)
		for j := 0; j < n; j++ {
			for x := 0; x < a.width; x++ {
				p := view.RGBAAt(x, y+j)
				k := (j*a.width + x) * 4
				buf[k], buf[k+1], buf[k+2] = p.B, p.G, p.R
			}
		}
		if err := xproto.PutImageChecked(a.conn, xproto.ImageFormatZPixmap, xproto.Drawable(a.win), a.gc,
			uint16(a.width), uint16(n), 0, int16(y), 0, a.screen.RootDepth, buf).Check(); err != nil {
			return err
		}
	}
	return nil
}

func fill(m *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(m, r, image.NewUniform(c), image.Point{}, draw.Src)
}
func (a *app) text(m *image.RGBA, x, y, size int, s string, c color.RGBA) {
	d := font.Drawer{Dst: m, Src: image.NewUniform(c), Face: a.faces[size], Dot: fixed.P(x, y)}
	d.DrawString(s)
}
