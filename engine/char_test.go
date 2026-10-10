package engine

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unsafe"
)

func TestCharRecordLayout(t *testing.T) {
	var r charRecord
	if n := binary.Size(&r); n != charRecSize {
		t.Fatalf("record is %#x bytes, want %#x", n, charRecSize)
	}
	// every field is bytes or little-endian words, so Go lays the struct out
	// without padding exactly where the DOS record has it
	for _, f := range []struct {
		name string
		off  uintptr
		want uintptr
	}{
		{"MaxHP", unsafe.Offsetof(r.MaxHP), 0x32},
		{"Control", unsafe.Offsetof(r.Control), 0x84},
		{"Levels", unsafe.Offsetof(r.Levels), 0x96},
		{"Exp", unsafe.Offsetof(r.Exp), 0xAC},
		{"ItemCount", unsafe.Offsetof(r.ItemCount), 0xC7},
		{"Hands", unsafe.Offsetof(r.Hands), 0x100},
		{"Status", unsafe.Offsetof(r.Status), 0x10C},
		{"HP", unsafe.Offsetof(r.HP), 0x11B},
	} {
		if f.off != f.want {
			t.Errorf("%s at %#x, want %#x", f.name, f.off, f.want)
		}
	}
	b := make([]byte, charRecSize)
	for i := range b {
		b[i] = byte(i * 7)
	}
	for _, o := range []int{0x7F, 0xC8, 0xCC, 0x104} { // pointers are dropped
		clear(b[o : o+4])
	}
	clear(b[0xCC:0x100])
	clear(b[0x104:0x10C])
	var c Character
	c.setRecord(b)
	if !bytes.Equal(c.record(), b) {
		t.Fatal("record does not round-trip")
	}
}

func TestItemAffectRecords(t *testing.T) {
	if n := binary.Size(Item{}); n != itemRecSize {
		t.Fatalf("item is %#x bytes, want %#x", n, itemRecSize)
	}
	if n := binary.Size(affectRecord{}); n != affectRecSize {
		t.Fatalf("affect is %d bytes, want %d", n, affectRecSize)
	}
	b := make([]byte, 2*itemRecSize)
	for i := range b {
		b[i] = byte(i)
	}
	clear(b[0x2A:0x2E]) // next pointers are dropped
	clear(b[itemRecSize+0x2A : itemRecSize+0x2E])
	items := readRecords[Item](b, itemRecSize)
	if len(items) != 2 || items[1].Price != 0x7A79 || !bytes.Equal(writeRecords(items), b) {
		t.Fatal("items do not round-trip")
	}
}
