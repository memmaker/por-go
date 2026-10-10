package dax

import "testing"

// One 8x1 frame two cells wide: plane 0 sets the left tile's first pixel,
// plane 3 the right tile's last; split must yield two 8-pixel tiles.
func TestAmigaImageSplit(t *testing.T) {
	b := []byte{0, 1, 0, 2, 0, 0, 0, 0, 0, 1, 0, 2, // h=1 w=2 count=1 plane=2
		0x80, 0, 0, 0, 0, 0, 0, 0x01} // planes 0..3, one 2-byte row each
	id := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	im := ParseImage(AmigaImage(b, true, &id), false, 0)
	if im.Count != 2 || im.Width != 1 || im.Frame(0)[0] != 1 || im.Frame(1)[7] != 8 {
		t.Fatalf("count %d width %d frames %v %v", im.Count, im.Width, im.Frame(0), im.Frame(1))
	}
}
