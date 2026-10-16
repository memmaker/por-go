// Command por runs Pool of Radiance on top of the original data files.
//
//	por [-data DIR] [-scale N]
//
// Gamepad (standard layout): D-pad or left stick move and turn, A picks the
// highlighted menu item, B is Escape, LB/RB move the menu highlight, LT/RT
// select the previous/next party member, X shows the party overview, Y
// toggles the minimap, Start is Enter.
//
// Mouse: pointing at a menu item highlights it, a click picks it (and list
// lines, party members, combat squares); the wheel moves the highlight and
// the right button is Escape. "?" lists the keys; F1-F6 open the inventory
// of party member 1-6.
//
// DIR must hold the DOS game files (TITLE.DAX, 8X8D1.DAX, GAME data ...);
// it defaults to $GOPLAY_DATA, else the parent of the executable's module
// directory, i.e. the folder this source tree sits in. With -import SRC
// (used by the goplay web page) the install found under SRC is copied into
// DIR with a COVER.PNG, and the program exits.
package main

import (
	"flag"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"por/ega"
	"por/engine"
)

type game struct {
	scr *ega.Screen
	eng *engine.Engine
	img *ebiten.Image
	buf []byte

	overlays []ega.Overlay                 // the engine's, copied each frame
	scans    map[image.Image]*ebiten.Image // their GPU copies

	stickDir  uint16 // left stick direction being held
	stickHeld int    // ticks it has been held

	dsf   float64 // device scale factor of the last Layout
	pic   rect    // where the picture is drawn, in device pixels
	wheel float64 // wheel movement not yet sent as a step
	sent  bool    // input went to the engine this tick
}

type rect struct{ x, y, w, h float64 }

// tps: Update runs 120 times a second, so a key waits at most ~8 ms before
// the engine sees it.
const tps = 120

func (g *game) Update() error {
	g.sent = false
	g.keys()
	for _, id := range ebiten.AppendGamepadIDs(nil) {
		g.gamepad(id)
	}
	g.mouse()
	if g.sent {
		g.settle()
	}
	return nil
}

// push sends a key to the engine.
func (g *game) push(k uint16) {
	g.scr.PushKey(k)
	g.sent = true
}

// settle gives the engine up to 8 ms to finish with the keys just sent, so
// the frame drawn next already shows their result (instead of one frame
// later). Gosched, not Sleep: in the browser the engine shares the thread.
func (g *game) settle() {
	t0 := time.Now()
	for !g.scr.Settled() && time.Since(t0) < 8*time.Millisecond {
		runtime.Gosched()
	}
}

// Held arrow keys repeat like the PC keyboard did (typematic), but only
// while the engine waits for input, so walking stops when the key is let go.
var repeatKeys = []ebiten.Key{
	ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight,
	ebiten.KeyKP8, ebiten.KeyKP2, ebiten.KeyKP4, ebiten.KeyKP6,
	ebiten.KeyPageUp, ebiten.KeyPageDown,
}

// keys passes the keyboard to the engine.
func (g *game) keys() {
	for _, k := range repeatKeys {
		if d := inpututil.KeyPressDuration(k); d > 1 && repeats(d) && g.scr.Idle() {
			code, _ := biosKey(k)
			g.push(code)
		}
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		if code, ok := biosKey(k); ok {
			g.push(code)
		} else if k == ebiten.KeyF && ebiten.IsKeyPressed(ebiten.KeyControl) && ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.push(engine.KeyFontPrev) // Ctrl-F next font, Ctrl-Shift-F the one before
		} else if k >= ebiten.KeyA && k <= ebiten.KeyZ && ebiten.IsKeyPressed(ebiten.KeyControl) {
			g.push(uint16(k-ebiten.KeyA) + 1) // Ctrl-letter: ASCII 1..26 (Ctrl-S sound toggle, Ctrl-C quit)
		}
	}
	// a Ctrl combination types no letter of its own (Ctrl+Alt is AltGr: it does)
	if ebiten.IsKeyPressed(ebiten.KeyControl) && !ebiten.IsKeyPressed(ebiten.KeyAlt) {
		return
	}
	for _, r := range ebiten.AppendInputChars(nil) {
		if r < 128 {
			g.push(uint16(r))
		}
	}
}

// mouse passes the pointer's text cell, clicks and wheel steps to the engine.
func (g *game) mouse() {
	mx, my := ebiten.CursorPosition()
	cx, cy, in := g.cell(float64(mx), float64(my))
	g.scr.SetMouse(cx, cy, in)
	if in && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.push(engine.MouseKey(engine.MouseLeft, cx, cy))
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.push(engine.MouseKey(engine.MouseRight, 0, 0))
	}
	_, dy := ebiten.Wheel()
	g.wheel += dy
	for ; g.wheel >= 1; g.wheel-- {
		g.push(engine.MouseKey(engine.MouseWheelUp, 0, 0))
	}
	for ; g.wheel <= -1; g.wheel++ {
		g.push(engine.MouseKey(engine.MouseWheelDown, 0, 0))
	}
}

// cell is the text cell (40x25) of the picture under device pixel (x, y).
func (g *game) cell(x, y float64) (int, int, bool) {
	p := g.pic
	if p.w <= 0 || x < p.x || y < p.y || x >= p.x+p.w || y >= p.y+p.h {
		return 0, 0, false
	}
	px := int((x - p.x) * ega.W / p.w)
	py := int((y - p.y) * ega.H / p.h)
	return px / 8, py / 8, true
}

// padButtons maps standard-layout gamepad buttons to BIOS key codes (and
// the engine's virtual gamepad keys).
var padButtons = []struct {
	b      ebiten.StandardGamepadButton
	key    uint16
	repeat bool
}{
	{ebiten.StandardGamepadButtonLeftTop, engine.KeyPadUp, true},
	{ebiten.StandardGamepadButtonLeftBottom, engine.KeyPadDown, true},
	{ebiten.StandardGamepadButtonLeftLeft, engine.KeyPadLeft, true},
	{ebiten.StandardGamepadButtonLeftRight, engine.KeyPadRight, true},
	{ebiten.StandardGamepadButtonRightBottom, engine.KeyPadSelect, false},
	{ebiten.StandardGamepadButtonRightRight, 27, false},
	{ebiten.StandardGamepadButtonRightLeft, engine.KeyPadOverview, false},
	{ebiten.StandardGamepadButtonRightTop, engine.KeyPadMinimap, false},
	{ebiten.StandardGamepadButtonFrontTopLeft, ',', true},
	{ebiten.StandardGamepadButtonFrontTopRight, '.', true},
	{ebiten.StandardGamepadButtonFrontBottomLeft, 0x4700, true},
	{ebiten.StandardGamepadButtonFrontBottomRight, 0x4F00, true},
	{ebiten.StandardGamepadButtonCenterRight, 13, false},
	{ebiten.StandardGamepadButtonCenterLeft, 27, false},
}

// Held buttons repeat after padDelay ticks (1/tps s), every padRate ticks.
const padDelay, padRate = tps / 3, tps * 3 / 20

func repeats(d int) bool { return d == 1 || d >= padDelay && (d-padDelay)%padRate == 0 }

func (g *game) gamepad(id ebiten.GamepadID) {
	if !ebiten.IsStandardGamepadLayoutAvailable(id) {
		return
	}
	for _, p := range padButtons {
		d := inpututil.StandardGamepadButtonPressDuration(id, p.b)
		if d == 1 || p.repeat && d > 1 && repeats(d) {
			g.push(p.key)
		}
	}
	// Left stick as a D-pad.
	x := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
	y := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
	dir := uint16(0)
	switch {
	case y < -0.6:
		dir = engine.KeyPadUp
	case y > 0.6:
		dir = engine.KeyPadDown
	case x < -0.6:
		dir = engine.KeyPadLeft
	case x > 0.6:
		dir = engine.KeyPadRight
	}
	if dir != g.stickDir {
		g.stickDir, g.stickHeld = dir, 0
	}
	if dir != 0 {
		g.stickHeld++
		if repeats(g.stickHeld) {
			g.push(dir)
		}
	}
}

func (g *game) Draw(dst *ebiten.Image) {
	g.scr.Lock()
	for i, c := range g.scr.Visible() {
		rgb := g.scr.RGBA(c)
		g.buf[i*4], g.buf[i*4+1], g.buf[i*4+2], g.buf[i*4+3] = rgb.R, rgb.G, rgb.B, 255
	}
	note, at := g.eng.Note()
	if time.Since(at) < 2*time.Second {
		g.drawNote(note)
	}
	g.overlays = append(g.overlays[:0], g.scr.Overlays...)
	g.scr.Unlock()
	g.img.WritePixels(g.buf)
	dst.Fill(color.Black)

	// The picture: as large as fits, 4:3 like a CRT.
	b := dst.Bounds()
	W, H := float64(b.Dx()), float64(b.Dy())
	scale := math.Min(W/4, H/3)
	pw, ph := scale*4, scale*3
	g.pic = rect{(W - pw) / 2, (H - ph) / 2, pw, ph}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(pw/ega.W, ph/ega.H)
	op.GeoM.Translate(g.pic.x, g.pic.y)
	dst.DrawImage(g.img, op)

	// Overlays (journal scans) at the window's own resolution.
	for _, o := range g.overlays {
		im := g.scans[o.Img]
		if im == nil {
			if g.scans == nil {
				g.scans = map[image.Image]*ebiten.Image{}
			}
			im = ebiten.NewImageFromImage(o.Img)
			g.scans[o.Img] = im
		}
		b := o.Part()
		im = im.SubImage(b).(*ebiten.Image) // drawn from its own top left
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(float64(o.At.Dx())*pw/ega.W/float64(b.Dx()), float64(o.At.Dy())*ph/ega.H/float64(b.Dy()))
		op.GeoM.Translate(g.pic.x+float64(o.At.Min.X)*pw/ega.W, g.pic.y+float64(o.At.Min.Y)*ph/ega.H)
		dst.DrawImage(im, op)
	}
}

// drawNote writes s into the top of the picture, white on blue in the
// current font, over what the game drew (the screen stays untouched).
func (g *game) drawNote(s string) {
	s = " " + s + " "
	if len(s) > 40 {
		s = s[:40]
	}
	x0 := (40 - len(s)) * 4
	for i := 0; i < len(s); i++ {
		gl := g.eng.Glyph(s[i])
		for r := 0; r < 8; r++ {
			for b := 0; b < 8; b++ {
				c := g.scr.RGBA(1)
				if gl[r]&(0x80>>b) != 0 {
					c = g.scr.RGBA(15)
				}
				p := ((r+4)*ega.W + x0 + i*8 + b) * 4
				g.buf[p], g.buf[p+1], g.buf[p+2] = c.R, c.G, c.B
			}
		}
	}
}

// Layout works in device pixels, so the picture fills the window exactly.
func (g *game) Layout(w, h int) (int, int) {
	g.dsf = ebiten.Monitor().DeviceScaleFactor()
	return int(float64(w) * g.dsf), int(float64(h) * g.dsf)
}

// biosKey maps keys the DOS game read as non-ASCII (scan code in the high
// byte, zero low byte) exactly like INT 16h returned them.
func biosKey(k ebiten.Key) (uint16, bool) {
	switch k {
	case ebiten.KeyEnter, ebiten.KeyKPEnter:
		return 13, true
	case ebiten.KeyEscape:
		return 27, true
	case ebiten.KeyBackspace:
		return 8, true
	case ebiten.KeyTab:
		return 9, true
	case ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5, ebiten.KeyF6:
		return engine.KeyF1 + uint16(k-ebiten.KeyF1)<<8, true
	case ebiten.KeyF7:
		return engine.KeyF7, true
	case ebiten.KeyArrowUp, ebiten.KeyKP8:
		return 0x4800, true
	case ebiten.KeyArrowDown, ebiten.KeyKP2:
		return 0x5000, true
	case ebiten.KeyArrowLeft, ebiten.KeyKP4:
		return 0x4B00, true
	case ebiten.KeyArrowRight, ebiten.KeyKP6:
		return 0x4D00, true
	case ebiten.KeyHome, ebiten.KeyKP7:
		return 0x4700, true
	case ebiten.KeyEnd, ebiten.KeyKP1:
		return 0x4F00, true
	case ebiten.KeyPageUp, ebiten.KeyKP9:
		return 0x4900, true
	case ebiten.KeyPageDown, ebiten.KeyKP3:
		return 0x5100, true
	case ebiten.KeyKP5:
		return 0x4C00, true
	}
	return 0, false
}

func main() {
	dir := flag.String("data", os.Getenv("GOPLAY_DATA"), "directory with the original game files")
	scale := flag.Int("scale", 3, "window scale")
	imp := flag.String("import", "", "copy the game found under this directory into -data, then exit (goplay)")
	imp2 := flag.String("import-disk2", "", "convert the Amiga graphics from the .adf disks under this directory into -data/DISK2, then exit (goplay)")
	flag.Parse()
	if *imp2 != "" {
		if *dir == "" {
			log.Fatal("-import-disk2 needs -data")
		}
		if err := importDisk2(*imp2, *dir); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *imp != "" {
		if *dir == "" {
			log.Fatal("-import needs -data")
		}
		if err := importGame(*imp, *dir); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *dir == "" {
		exe, _ := os.Executable()
		*dir = filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(*dir, "TITLE.DAX")); err != nil {
			*dir = filepath.Join("..")
		}
	}
	scr := ega.New()
	eng, err := engine.New(*dir, scr)
	if err != nil {
		log.Fatal(err)
	}
	actx := audio.NewContext(44100)
	if pl, err := actx.NewPlayer(eng.SoundReader()); err == nil {
		// at least the 2048 frames (46 ms) a browser pulls at once, or the
		// rest of each pull plays as silence (goplay)
		pl.SetBufferSize(50 * time.Millisecond)
		pl.Play()
	}
	eng.Cfg.Param = flag.Arg(0)
	go eng.Run()

	g := &game{scr: scr, eng: eng, img: ebiten.NewImage(ega.W, ega.H), buf: make([]byte, ega.W*ega.H*4), dsf: 1}
	ebiten.SetWindowSize(ega.W*(*scale), ega.H*(*scale)*6/5) // 4:3 like a CRT
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Pool of Radiance")
	ebiten.SetTPS(tps)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
