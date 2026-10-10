package amiga

import (
	"encoding/binary"
	"testing"
)

// A 2-byte looped sample on channel 0 comes out left only, at full volume:
// a fast square wave of 100/-100 averages 0, a constant one settles at 100*64*2 once through the filter.
func TestPaula(t *testing.T) {
	for _, c := range []struct {
		b    byte
		want int16
	}{{0x9c, 0}, {100, 12800}} {
		p := New(make([]byte, End-Base))
		p.frac = 1e9 // no player ticks
		p.wb(0x65000, 100)
		p.wb(0x65001, c.b)
		p.wl(0xdff0a0, 0x65000)
		p.hww(0xdff0a4, 1)
		p.hww(0xdff0a6, 124)
		p.hww(0xdff0a8, 64)
		p.hww(0xdff096, 0x8201)
		out := make([]byte, 4*1000)
		p.Render(out, 44100)
		l, r := 0, 0 // mean of the last 100 frames
		for i := len(out) - 400; i < len(out); i += 4 {
			l += int(int16(binary.LittleEndian.Uint16(out[i:])))
			r += int(int16(binary.LittleEndian.Uint16(out[i+2:])))
		}
		l, r = l/100, r/100
		if l < int(c.want)-200 || l > int(c.want)+200 || r != 0 {
			t.Fatalf("%x: %d %d, want %d 0", c.b, l, r, c.want)
		}
	}
}
