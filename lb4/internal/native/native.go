// Package native uses the real X11 circle primitive as the fourth method.
// No private approximation is substituted for a server-rendered arc.
package native

import (
	"errors"
	"fmt"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"graphiclabs/lb4/internal/raster"
	"math"
)

func Rasterize(cfg raster.Config) (*raster.Mask, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	c, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("четвёртому методу нужен X11/XWayland и DISPLAY: %w", err)
	}
	defer c.Close()
	setup := xproto.Setup(c)
	screen := setup.DefaultScreen(c)
	valid := false
	for _, d := range screen.AllowedDepths {
		for _, v := range d.Visuals {
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
		return nil, errors.New("требуется X11 TrueColor с 32-битным хранением LSBFirst")
	}
	pix, err := xproto.NewPixmapId(c)
	if err != nil {
		return nil, err
	}
	if err = xproto.CreatePixmapChecked(c, screen.RootDepth, pix, xproto.Drawable(screen.Root), uint16(cfg.Width), uint16(cfg.Height)).Check(); err != nil {
		return nil, err
	}
	defer xproto.FreePixmap(c, pix)
	gc, err := xproto.NewGcontextId(c)
	if err != nil {
		return nil, err
	}
	if err = xproto.CreateGCChecked(c, gc, xproto.Drawable(pix), xproto.GcForeground|xproto.GcLineWidth, []uint32{0, 0}).Check(); err != nil {
		return nil, err
	}
	defer xproto.FreeGC(c, gc)
	if err = xproto.PolyFillRectangleChecked(c, xproto.Drawable(pix), gc, []xproto.Rectangle{{Width: uint16(cfg.Width), Height: uint16(cfg.Height)}}).Check(); err != nil {
		return nil, err
	}
	if err = xproto.ChangeGCChecked(c, gc, xproto.GcForeground, []uint32{0xffffff}).Check(); err != nil {
		return nil, err
	}
	var arcs []xproto.Arc
	var ends []xproto.Point
	for _, a := range cfg.Arcs() {
		// X11 angles are counter-clockwise, while our screen angles increase clockwise.
		arcs = append(arcs, xproto.Arc{X: int16(math.Round(a.Center.X)) - int16(a.Radius), Y: int16(math.Round(a.Center.Y)) - int16(a.Radius), Width: uint16(2 * a.Radius), Height: uint16(2 * a.Radius), Angle1: int16(-a.Start * 64), Angle2: int16(-a.Sweep * 64)})
		if a.Sweep < 360 {
			for _, t := range []float64{a.Start, a.Start + a.Sweep} {
				p := a.At(t)
				ends = append(ends, xproto.Point{X: int16(math.Round(p.X)), Y: int16(math.Round(p.Y))})
			}
		}
	}
	if err = xproto.PolyArcChecked(c, xproto.Drawable(pix), gc, arcs).Check(); err != nil {
		return nil, err
	}
	// Explicit shared endpoints use the same rounded vertices as the custom methods.
	if len(ends) > 0 {
		if err = xproto.PolyPointChecked(c, xproto.CoordModeOrigin, xproto.Drawable(pix), gc, ends).Check(); err != nil {
			return nil, err
		}
	}
	reply, err := xproto.GetImage(c, xproto.ImageFormatZPixmap, xproto.Drawable(pix), 0, 0, uint16(cfg.Width), uint16(cfg.Height), 0xffffffff).Reply()
	if err != nil {
		return nil, err
	}
	if len(reply.Data) != cfg.Width*cfg.Height*4 {
		return nil, errors.New("неожиданный формат растра X11")
	}
	m := raster.NewMask(cfg.Width, cfg.Height)
	for i := range m.Bits {
		k := 4 * i
		m.Bits[i] = reply.Data[k] != 0 || reply.Data[k+1] != 0 || reply.Data[k+2] != 0
	}
	return m, nil
}
