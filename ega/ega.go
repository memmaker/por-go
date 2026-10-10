// Package ega is the 320x200 16-colour frame buffer the game draws into,
// plus the keyboard queue, modelled on the BIOS services the DOS build used.
package ega

import (
	"image/color"
	"sync"
	"sync/atomic"
	"time"
)

const (
	W = 320
	H = 200
)

// Palette is the standard 16-colour EGA/CGA palette (RGB).
var Palette = [16][3]uint8{
	{0, 0, 0}, {0, 0, 173}, {0, 173, 0}, {0, 173, 173},
	{173, 0, 0}, {173, 0, 173}, {173, 82, 0}, {173, 173, 173},
	{82, 82, 82}, {82, 82, 255}, {82, 255, 82}, {82, 255, 255},
	{255, 82, 82}, {255, 82, 255}, {255, 255, 82}, {255, 255, 255},
}

// Screen is the shared frame buffer. The engine goroutine writes it; the
// renderer snapshots it each frame under the lock.
type Screen struct {
	mu    sync.Mutex
	Pix   [W * H]uint8 // colour index per pixel
	Pal   [16]uint8    // palette register -> colour (SetPalette remaps)
	Keys  chan uint16  // BIOS style: low byte ASCII, high byte scan code
	Dirty bool

	held  int          // Hold depth: while > 0 the window shows shown
	shown [W * H]uint8 // last released frame

	mouse int    // text cell under the pointer + 1, 0 outside the screen
	hint  string // key hint for the window's status line

	state atomic.Int32 // what the engine goroutine does: Working, Waiting, Pausing
}

// Engine states (SetState): the window draws a key's result once the engine
// is done with it (Settled).
const (
	Working = iota // handling input
	Waiting        // waiting for input
	Pausing        // a deliberate delay (Delay)
)

// SetState records what the engine does now (no screen: tests).
func (s *Screen) SetState(st int32) {
	if s != nil {
		s.state.Store(st)
	}
}

// Settled: the engine has taken the keys sent and finished with them, or
// pauses on purpose; the picture is worth drawing.
func (s *Screen) Settled() bool {
	st := s.state.Load()
	return st == Pausing || st == Waiting && len(s.Keys) == 0
}

// Idle: the engine waits for input and none is queued (held keys repeat
// only then, so they never pile up).
func (s *Screen) Idle() bool { return s.state.Load() == Waiting && len(s.Keys) == 0 }

// Hold freezes what the window shows until the matching Release, so a
// picture built in several passes (the 3D view) appears at once like the
// DOS back buffer flip (0464:1B52).
func (s *Screen) Hold() {
	s.mu.Lock()
	if s.held == 0 {
		s.shown = s.Pix
	}
	s.held++
	s.mu.Unlock()
}

// Release ends a Hold.
func (s *Screen) Release() {
	s.mu.Lock()
	if s.held > 0 {
		s.held--
	}
	s.mu.Unlock()
}

// Visible returns the pixels the window should show; call under Lock.
func (s *Screen) Visible() *[W * H]uint8 {
	if s.held > 0 {
		return &s.shown
	}
	return &s.Pix
}

// New creates a screen with the identity palette.
func New() *Screen {
	s := &Screen{Keys: make(chan uint16, 64)}
	for i := range s.Pal {
		s.Pal[i] = uint8(i)
	}
	return s
}

// Lock/Unlock guard Pix for the renderer.
func (s *Screen) Lock()   { s.mu.Lock() }
func (s *Screen) Unlock() { s.mu.Unlock() }

// Set writes one pixel; values >= 16 are ignored (transparent).
func (s *Screen) Set(x, y int, c uint8) {
	if c == 16 { // dax.Clear: a masked-out pixel drawn plain
		c = 0
	}
	if c < 16 && x >= 0 && x < W && y >= 0 && y < H {
		s.Pix[y*W+x] = c
	}
}

// Get reads one pixel.
func (s *Screen) Get(x, y int) uint8 {
	if x < 0 || x >= W || y < 0 || y >= H {
		return 0
	}
	return s.Pix[y*W+x]
}

// Fill paints a rectangle [x0,x1) x [y0,y1).
func (s *Screen) Fill(x0, y0, x1, y1 int, c uint8) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			s.Set(x, y, c)
		}
	}
}

// RGBA returns the displayed colour of palette register i.
func (s *Screen) RGBA(i uint8) color.RGBA {
	p := Palette[s.Pal[i&15]&15]
	return color.RGBA{p[0], p[1], p[2], 255}
}

// SetPalette remaps palette register index to EGA colour.
func (s *Screen) SetPalette(index, colour int) {
	s.Pal[index&15] = uint8(colour & 15)
}

// --- keyboard ---------------------------------------------------------------

// KeyPressed reports whether a key is waiting (INT 16h/01h).
func (s *Screen) KeyPressed() bool { return len(s.Keys) > 0 }

// ReadKey blocks for the next key (INT 16h/00h).
func (s *Screen) ReadKey() uint16 {
	s.state.Store(Waiting)
	k := <-s.Keys
	s.state.Store(Working)
	return k
}

// PushKey enqueues a key from the window layer.
func (s *Screen) PushKey(k uint16) {
	select {
	case s.Keys <- k:
	default:
	}
}

// --- mouse and hints -------------------------------------------------------

// SetMouse records the text cell (40x25) under the pointer; ok false when
// the pointer is off the game picture.
func (s *Screen) SetMouse(cx, cy int, ok bool) {
	s.mu.Lock()
	s.mouse = 0
	if ok && cx >= 0 && cx < W/8 && cy >= 0 && cy < H/8 {
		s.mouse = cy*(W/8) + cx + 1
	}
	s.mu.Unlock()
}

// Mouse returns the cell under the pointer.
func (s *Screen) Mouse() (cx, cy int, ok bool) {
	s.mu.Lock()
	m := s.mouse
	s.mu.Unlock()
	if m == 0 {
		return 0, 0, false
	}
	return (m - 1) % (W / 8), (m - 1) / (W / 8), true
}

// SetHint sets the key hint the window shows under the picture.
func (s *Screen) SetHint(h string) {
	s.mu.Lock()
	s.hint = h
	s.mu.Unlock()
}

// Hint returns the key hint.
func (s *Screen) Hint() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hint
}

// Delay sleeps; the engine calls this instead of busy-waiting on the timer.
func Delay(ms int) {
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}
