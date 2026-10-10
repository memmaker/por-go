package engine

import "testing"

func TestItemSpan(t *testing.T) {
	menu := "Parts color-1 color-2 Size Exit"
	start, end := menuItems(menu)
	want := []string{"", "Parts", "color-1", "color-2", "Size", "Exit"}
	for j := 1; j < len(start); j++ {
		if a, b := itemSpan(menu, start, end, j); menu[a-1:b] != want[j] {
			t.Errorf("item %d: %q, want %q", j, menu[a-1:b], want[j])
		}
	}
	// a word with two hotkeys keeps the original spans
	start, end = menuItems("OK No")
	if a, b := itemSpan("OK No", start, end, 1); a != 1 || b != 1 {
		t.Errorf("OK: %d..%d", a, b)
	}
}

func TestItemStats(t *testing.T) {
	e := &Engine{items: make([]byte, 64)}
	copy(e.items[16+9:], []byte{2, 4, 1}) // type 1: 2d4+1
	e.items[32+6] = 0x80 + 1              // type 2: shield 1
	e.items[48+6] = 0x80 + 57             // type 3: plate mail 7
	for typ, want := range []string{"", "(2d4+1) ", "<1> ", "<7> "} {
		if s := e.itemStats(byte(typ)); s != want {
			t.Errorf("type %d: %q, want %q", typ, s, want)
		}
	}
}
