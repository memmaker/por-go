package amiga

import (
	"encoding/binary"
	"testing"
)

// A 2-byte looped sample on channel 0 comes out left only, at full volume,
// and wraps back to its start.
func TestPaula(t *testing.T) {
	p := New(make([]byte, End-Base))
	p.frac = 1e9 // no player ticks
	p.wb(0x65000, 100)
	p.wb(0x65001, 0x9c) // -100
	p.wl(0xdff0a0, 0x65000)
	p.hww(0xdff0a4, 1)
	p.hww(0xdff0a6, pal/44100) // one sample byte per output frame
	p.hww(0xdff0a8, 64)
	p.hww(0xdff096, 0x8201)
	out := make([]byte, 12)
	p.Render(out, 44100)
	for i, want := range []int16{100 * 128, -100 * 128, 100 * 128} {
		l, r := int16(binary.LittleEndian.Uint16(out[4*i:])), binary.LittleEndian.Uint16(out[4*i+2:])
		if l != want || r != 0 {
			t.Fatalf("frame %d: %d %d, want %d 0", i, l, r, want)
		}
	}
}
