package desktop

import (
	"bytes"
	"context"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"graphiclabs/lb10/internal/bezier"
	"graphiclabs/lb10/internal/scene"
)

func TestNativeWorkflow(t *testing.T) {
	display, binary := os.Getenv("DISPLAY"), os.Getenv("LAB10_GUI_BINARY")
	if display == "" || binary == "" {
		t.Skip("requires DISPLAY and LAB10_GUI_BINARY")
	}
	out := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-out", out)
	cmd.Env = []string{"PATH=/nonexistent", "DISPLAY=" + display}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	conn, err := xgb.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	setup := xproto.Setup(conn)
	x := uiTest{t: t, c: conn, root: setup.DefaultScreen(conn).Root}
	x.window = x.awaitWindow(done)
	geom, err := xproto.GetGeometry(conn, xproto.Drawable(x.window)).Reply()
	if err != nil {
		t.Fatal(err)
	}
	x.view = (&window{width: int(geom.Width), height: int(geom.Height)}).viewport()
	keyCount := byte(int(setup.MaxKeycode) - int(setup.MinKeycode) + 1)
	x.keys, err = xproto.GetKeyboardMapping(conn, setup.MinKeycode, keyCount).Reply()
	if err != nil {
		t.Fatal(err)
	}
	x.minKey = setup.MinKeycode

	x.click(1100, 398) // Quadratic, stage 1.
	x.capture("interface-quadratic.png", geom)
	x.click(1140, 398)
	x.enter(171, 143, "1.5")
	x.key(0xff1b) // Dismiss invalid t.
	x.enter(171, 143, "0,25")
	x.enter(508, 143, "6")
	x.click(650, 245) // Append a sixth vertex.
	x.enter(925, 250, "4.5")
	x.enter(1088, 250, "3")
	x.click(50, 774) // Hide the original polygon.
	x.click(860, 774)

	raw, _, _ := bezier.DemoCurve(2)
	camera := scene.Fit(raw, false)
	control := append(raw, bezier.Point{X: 4.5, Y: 3})
	curve, err := bezier.SampleCurve(control, 2, 6)
	if err != nil {
		t.Fatal(err)
	}
	sceneBox := image.Rect(0, 0, plotBox.Dx(), plotBox.Dy())
	want := image.NewRGBA(sceneBox)
	scene.DrawCurve(want, sceneBox, camera, scene.CurveView{
		Control: control, Curve: curve, Construction: bezier.Construct(curve.Segments[2], 0.25),
		Stage: 2, Layers: scene.Layers{Refined: true, Selected: 5},
	})
	x.checkPNG(out, "quadratic", want)
	x.click(50, 774)

	x.click(373, 99)
	x.capture("interface-cubic.png", geom)
	x.click(630, 99)
	x.capture("interface-surface.png", geom)
	x.click(350, 774)
	x.capture("interface-surface-original.png", geom)
	x.click(350, 774)
	x.enter(963, 385, "5")
	x.enter(1125, 385, "4")
	x.enter(929, 291, "2.5")
	x.enter(91, 143, "0.25")
	x.enter(252, 143, "0.8")
	x.enter(518, 143, "8")
	x.enter(774, 143, "5")
	x.click(860, 774)
	grid := bezier.ResizeGrid(bezier.DemoGrid(), 5, 4)
	camera = scene.Fit(grid.Points(), true)
	grid[0][0].Z = 2.5
	patch, err := bezier.SampleSurface(grid, 0.25, 0.8, 8, 5)
	if err != nil {
		t.Fatal(err)
	}
	scene.DrawSurface(want, sceneBox, camera, scene.SurfaceView{
		Control: grid, Surface: patch, Layers: scene.Layers{Original: true, Refined: true, Selected: 0},
	})
	x.checkPNG(out, "surface", want)

	atom, err := xproto.InternAtom(conn, false, 16, "WM_DELETE_WINDOW").Reply()
	if err != nil {
		t.Fatal(err)
	}
	data := xproto.ClientMessageDataUnionData32New([]uint32{uint32(atom.Atom), 0, 0, 0, 0})
	x.send(xproto.ClientMessageEvent{Format: 32, Window: x.window, Data: data}.Bytes(), 0)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("the window did not close")
	}
}

type uiTest struct {
	t            *testing.T
	c            *xgb.Conn
	root, window xproto.Window
	view         image.Rectangle
	keys         *xproto.GetKeyboardMappingReply
	minKey       xproto.Keycode
}

func (x uiTest) awaitWindow(done <-chan error) xproto.Window {
	x.t.Helper()
	for until := time.Now().Add(8 * time.Second); time.Now().Before(until); {
		tree, err := xproto.QueryTree(x.c, x.root).Reply()
		if err != nil {
			x.t.Fatal(err)
		}
		for _, window := range tree.Children {
			name, err := xproto.GetProperty(x.c, false, window, xproto.AtomWmName, xproto.AtomString, 0, 100).Reply()
			if err == nil && string(name.Value) == windowTitle {
				return window
			}
		}
		select {
		case err := <-done:
			x.t.Fatal("window exited:", err)
		case <-time.After(25 * time.Millisecond):
		}
	}
	x.t.Fatal("window did not appear")
	return 0
}

func (x uiTest) send(event []byte, mask uint32) {
	x.t.Helper()
	if err := xproto.SendEventChecked(x.c, false, x.window, mask, string(event)).Check(); err != nil {
		x.t.Fatal(err)
	}
}

func (x uiTest) click(px, py int) {
	px = x.view.Min.X + px*x.view.Dx()/frameW
	py = x.view.Min.Y + py*x.view.Dy()/frameH
	event := xproto.ButtonPressEvent{
		Detail: 1, Root: x.root, Event: x.window, EventX: int16(px), EventY: int16(py), SameScreen: true,
	}
	x.send(event.Bytes(), xproto.EventMaskButtonPress)
	release := xproto.ButtonReleaseEvent(event)
	x.send(release.Bytes(), xproto.EventMaskButtonRelease)
}

func (x uiTest) key(sym xproto.Keysym) {
	x.t.Helper()
	for i := 0; i < len(x.keys.Keysyms); i += int(x.keys.KeysymsPerKeycode) {
		if x.keys.Keysyms[i] == sym {
			event := xproto.KeyPressEvent{
				Detail: x.minKey + xproto.Keycode(i/int(x.keys.KeysymsPerKeycode)),
				Root:   x.root, Event: x.window, SameScreen: true,
			}
			x.send(event.Bytes(), xproto.EventMaskKeyPress)
			return
		}
	}
	x.t.Fatalf("no key for symbol %#x", sym)
}

func (x uiTest) enter(px, py int, text string) {
	x.click(px, py)
	for _, ch := range text {
		x.key(xproto.Keysym(ch))
	}
	x.key(0xff0d)
}

func (x uiTest) capture(name string, geom *xproto.GetGeometryReply) {
	dir := os.Getenv("LAB10_GUI_CAPTURE_DIR")
	if dir == "" {
		return
	}
	time.Sleep(220 * time.Millisecond)
	img, err := xproto.GetImage(x.c, xproto.ImageFormatZPixmap, xproto.Drawable(x.window),
		0, 0, geom.Width, geom.Height, ^uint32(0)).Reply()
	if err != nil {
		x.t.Fatal(err)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, int(geom.Width), int(geom.Height)))
	for i := 0; i < len(rgba.Pix); i += 4 {
		rgba.Pix[i], rgba.Pix[i+1], rgba.Pix[i+2], rgba.Pix[i+3] =
			img.Data[i+2], img.Data[i+1], img.Data[i], 255
	}
	if err := scene.SavePNG(filepath.Join(dir, name), rgba); err != nil {
		x.t.Fatal(err)
	}
}

func (x uiTest) checkPNG(dir, name string, expected *image.RGBA) {
	x.t.Helper()
	path := filepath.Join(dir, name+".png")
	wantPath := filepath.Join(dir, name+"-expected.png")
	if err := scene.SavePNG(wantPath, expected); err != nil {
		x.t.Fatal(err)
	}
	want, err := os.ReadFile(wantPath)
	if err != nil {
		x.t.Fatal(err)
	}
	for until := time.Now().Add(5 * time.Second); time.Now().Before(until); {
		if got, err := os.ReadFile(path); err == nil {
			if !bytes.Equal(got, want) {
				x.t.Fatalf("%s from the GUI differs from the expected scene", name)
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	x.t.Fatal("PNG was not saved:", path)
}
