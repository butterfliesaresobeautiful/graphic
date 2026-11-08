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
	"graphiclabs/lb8/internal/imaging"
)

// This is an actual X11 window test, not a painted mock-up.
func TestNativeWindowWorkflow(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("requires X11/Xvfb; run with DISPLAY for native integration")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "source.png")
	out := filepath.Join(dir, "output")
	src := imaging.Demo()
	if err := imaging.SavePNG(input, src); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	if binary := os.Getenv("LAB8_GUI_BINARY"); binary != "" {
		cmd := exec.Command(binary, "-source", input, "-out", out, "-sx", "0.9")
		cmd.Env = []string{"PATH=/nonexistent", "DISPLAY=" + os.Getenv("DISPLAY")}
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		go func() { done <- cmd.Run() }()
	} else {
		go func() { done <- Run(src, input, out, imaging.Default()) }()
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
			if e == nil && strings.HasPrefix(string(p.Value), "Lab8 -") {
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
	// Replace the initial width, then create, load, copy, draw and save.
	click(140, 148)
	setup := xproto.Setup(c)
	keys, e := xproto.GetKeyboardMapping(c, setup.MinKeycode, byte(int(setup.MaxKeycode)-int(setup.MinKeycode)+1)).Reply()
	if e != nil {
		t.Fatal(e)
	}
	for _, ch := range "1.25" {
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
	click(100, 98) // Open the file dialog and choose the real source.
	click(300, 253)
	click(800, 98)  // Transform all three methods.
	click(1040, 98) // Save all PNGs and metrics.
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(out, "bicubic.png")); err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	want, err := imaging.Transform(src, imaging.Default())
	if err != nil {
		t.Fatal(err)
	}
	got, err := imaging.Load(filepath.Join(out, "bicubic.png"))
	if err != nil {
		t.Fatal("GUI did not save", err)
	}
	if !bytes.Equal(got.Pix, want.Images[2].Pix) {
		t.Fatal("GUI bicubic differs from core")
	}
	for i, name := range imaging.Methods {
		img, err := imaging.Load(filepath.Join(out, name+".png"))
		if err != nil || !bytes.Equal(img.Pix, want.Images[i].Pix) {
			t.Fatal("GUI result differs", name, err)
		}
	}
	for shot, path := range []string{os.Getenv("LAB8_GUI_CAPTURE"), os.Getenv("LAB8_GUI_COMPARISON")} {
		if path == "" {
			continue
		}
		if shot == 1 {
			click(1100, 200)
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

func TestActionsRejectInvalidAndKeepState(t *testing.T) {
	a := &app{source: imaging.Demo(), out: t.TempDir()}
	a.setConfig(imaging.Default())
	if e := a.action("save"); e == nil {
		t.Fatal("saved before transform")
	}
	a.sx = "1,25"
	if e := a.action("all"); e != nil {
		t.Fatal(e)
	}
	old := a.result
	a.hx = "bad"
	if e := a.action("all"); e == nil {
		t.Fatal("bad input")
	}
	if a.result != old {
		t.Fatal("discarded previous result")
	}
	a.setConfig(imaging.Config{ScaleX: 1, ScaleY: 1, ShearX: 1, ShearY: 1})
	if e := a.action("all"); e == nil {
		t.Fatal("singular accepted")
	}
	if a.result != old {
		t.Fatal("changed previous result")
	}
	if e := a.action("demo"); e != nil || a.result != nil {
		t.Fatal("stale result after source change")
	}
}
