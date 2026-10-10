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
