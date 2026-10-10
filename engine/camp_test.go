package engine

import "testing"

// The scripts' "hour < 14" sky check follows day 6:00-20:00 instead.
func TestSkyHour(t *testing.T) {
	e := &Engine{area: make([]byte, 0x800), ecl: make([]byte, 0x10000)}
	// opCompare [49C9] 14
	for i, b := range []byte{0x03, 0x01, 0xC9, 0x49, 0x00, 14} {
		e.setEclByte(0x9900+i, b)
	}
	for _, c := range []struct {
		hour uint16
		day  bool
	}{{1, false}, {6, true}, {13, true}, {15, true}, {19, true}, {20, false}} {
		putWord(e.area, 0x192, c.hour)
		e.eclPC = 0x9900
		e.opCompare()
		if lt := e.es.flags[2]; lt != c.day {
			t.Errorf("hour %d: day sky %v, want %v", c.hour, lt, c.day)
		}
	}
}
