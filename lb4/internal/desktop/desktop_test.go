package desktop

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"graphiclabs/lb4/internal/raster"
)

// This is an actual X11 window test, not a painted mock-up.
func TestNativeWindowWorkflow(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("requires X11/Xvfb; run with DISPLAY for native integration")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "source.png")
	out := filepath.Join(dir, "output")
	src, _ := raster.New(640, 480)
	if err := raster.Save(input, src); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	if binary := os.Getenv("LAB4_GUI_BINARY"); binary != "" {
		cmd := exec.Command(binary, "-source", input, "-out", out, "-size", "180")
		cmd.Env = []string{"PATH=/nonexistent", "DISPLAY=" + os.Getenv("DISPLAY")}
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		go func() { done <- cmd.Run() }()
	} else {
		go func() { done <- Run(src, input, out, raster.Default()) }()
	}
	c, err := xgb.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	screen := xproto.Setup(c).DefaultScreen(c)
	var win xproto.Window
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && win == 0 {
		tree, err := xproto.QueryTree(c, screen.Root).Reply()
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range tree.Children {
			p, e := xproto.GetProperty(c, false, w, xproto.AtomWmName, xproto.AtomString, 0, 100).Reply()
			if e == nil && strings.HasPrefix(string(p.Value), "Lab4 -") {
				win = w
				break
			}
		}
		if win == 0 {
			select {
			case err := <-done:
				t.Fatal("window failed", err)
			default:
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	if win == 0 {
		t.Fatal("window did not appear")
	}
	defer func() {
		atom, _ := xproto.InternAtom(c, false, 16, "WM_DELETE_WINDOW").Reply()
		data := xproto.ClientMessageDataUnionData32New([]uint32{uint32(atom.Atom), 0, 0, 0, 0})
		e := xproto.ClientMessageEvent{Format: 32, Window: win, Data: data}
		xproto.SendEvent(c, false, win, 0, string(e.Bytes()))
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("window did not close")
		}
	}()
	click := func(x, y int) {
		geom, e := xproto.GetGeometry(c, xproto.Drawable(win)).Reply()
		if e != nil {
			t.Fatal(e)
		}
		scale := min(float64(geom.Width)/frameW, float64(geom.Height)/frameH)
		px := int((float64(geom.Width)-frameW*scale)/2 + float64(x)*scale)
		py := int((float64(geom.Height)-frameH*scale)/2 + float64(y)*scale)
		ev := xproto.ButtonPressEvent{Detail: 1, Root: screen.Root, Event: win, EventX: int16(px), EventY: int16(py), SameScreen: true}
		if err := xproto.SendEventChecked(c, false, win, xproto.EventMaskButtonPress, string(ev.Bytes())).Check(); err != nil {
			t.Fatal(err)
		}
	}
	click(100, 98)  // Actual file chooser.
	click(300, 253) // First entry is the saved source.png.
	// Exercise the real width field: replace the release binary's initial 180 with 280.
	click(140, 148)
	setup := xproto.Setup(c)
	keys, e := xproto.GetKeyboardMapping(c, setup.MinKeycode, byte(int(setup.MaxKeycode)-int(setup.MinKeycode)+1)).Reply()
	if e != nil {
		t.Fatal(e)
	}
	for _, ch := range "280" {
		found := false
		for k := int(setup.MinKeycode); k <= int(setup.MaxKeycode); k++ {
			idx := (k - int(setup.MinKeycode)) * int(keys.KeysymsPerKeycode)
			if keys.Keysyms[idx] != xproto.Keysym(ch) {
				continue
			}
			ev := xproto.KeyPressEvent{Detail: xproto.Keycode(k), Root: screen.Root, Event: win, SameScreen: true}
			if err := xproto.SendEventChecked(c, false, win, xproto.EventMaskKeyPress, string(ev.Bytes())).Check(); err != nil {
				t.Fatal(err)
			}
			found = true
			break
		}
		if !found {
			t.Fatal("keyboard digit missing", ch)
		}
	}
	click(815, 98)  // All four methods.
	click(1030, 98) // Save all.
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(out, "native.png")); err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	want, err := Process(src, raster.Default())
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"equation.png", "parametric.png", "bresenham.png", "native.png"} {
		got, err := raster.Load(filepath.Join(out, name))
		if err != nil {
			t.Fatal("UI did not save", err)
		}
		expected := want.Images[i]
		if !bytes.Equal(got.Pix, expected.Pix) {
			t.Fatal("native result differs", name)
		}
	}
	for shot, path := range []string{os.Getenv("LAB4_GUI_CAPTURE"), os.Getenv("LAB4_GUI_COMPARISON")} {
		if path == "" {
			continue
		}
		if shot == 1 {
			click(900, 148)
		}
		// Let the final completed paint reach the X server, then read its window pixels.
		time.Sleep(250 * time.Millisecond)
		geom, err := xproto.GetGeometry(c, xproto.Drawable(win)).Reply()
		if err != nil {
			t.Fatal(err)
		}
		pixels, err := xproto.GetImage(c, xproto.ImageFormatZPixmap, xproto.Drawable(win), 0, 0, geom.Width, geom.Height, 0xffffffff).Reply()
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rect(0, 0, int(geom.Width), int(geom.Height)))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = pixels.Data[i+2], pixels.Data[i+1], pixels.Data[i], 255
		}
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, img)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
}

func TestInvalidParametersAndSaveOrder(t *testing.T) {
	src, _ := raster.New(640, 480)
	a := &app{source: src, config: raster.Default(), size: "bad", cx: "320", cy: "240", radius: "140", out: t.TempDir()}
	if err := a.action("all"); err == nil {
		t.Fatal("invalid width accepted")
	}
	if err := a.action("save-all"); err == nil {
		t.Fatal("saved before rendering")
	}
	a.size = "280"
	c, err := a.parameters()
	if err != nil || c != raster.Default() {
		t.Fatal("parameters", err)
	}
	if err = a.action("shape"); err != nil || !a.config.Circle {
		t.Fatal("shape toggle")
	}
}
