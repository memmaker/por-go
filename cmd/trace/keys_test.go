package main

import "testing"

// Same vector as decomp/emu/keys.py's self-check.
func TestParseKeys(t *testing.T) {
	got, err := ParseKeys([]string{"enter", "a", "A", "0x4800", "f1"})
	want := []uint16{0x1C0D, 0x1E61, 0x1E41, 0x4800, 0x3B00}
	if err != nil || len(got) != len(want) {
		t.Fatal(got, err)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%d: %04X want %04X", i, got[i], want[i])
		}
	}
}
