package engine

import "testing"

func TestSlotName(t *testing.T) {
	for in, want := range map[string]string{"before boss": "BEFORE BOSS", "../x/y": "XY", " a:b ": "AB", "": ""} {
		if got := slotName(in); got != want {
			t.Errorf("slotName(%q) = %q, want %q", in, got, want)
		}
	}
}
