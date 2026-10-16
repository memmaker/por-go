package ega

import "testing"

func TestFillMatchesSet(t *testing.T) {
	rects := [][4]int{{0, 0, 5, 5}, {-3, -2, 4, 3}, {310, 195, 330, 210}, {5, 5, 5, 9}, {9, 9, 2, 2}}
	for _, c := range []uint8{3, 16, 20, AmigaBase + 5} {
		for _, r := range rects {
			a, b := New(), New()
			a.Fill(r[0], r[1], r[2], r[3], c)
			for y := r[1]; y < r[3]; y++ {
				for x := r[0]; x < r[2]; x++ {
					b.Set(x, y, c)
				}
			}
			if a.Pix != b.Pix {
				t.Fatalf("Fill%v c=%d differs from Set", r, c)
			}
		}
	}
}
