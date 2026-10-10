package dax

import "testing"

// One 8x1 frame two cells wide: plane 0 sets the left tile's first pixel,
// plane 3 the right tile's last; split must yield two 8-pixel tiles in
// palette slot 1 (colours 48..63).
func TestAmigaImageSplit(t *testing.T) {
	b := []byte{0, 1, 0, 2, 0, 0, 0, 0, 0, 1, 0, 2, // h=1 w=2 count=1 plane=2
		0x80, 0, 0, 0, 0, 0, 0, 0x01} // planes 0..3, one 2-byte row each
	im := ParseImage(AmigaImage(b, true, 1), false, 0)
	if im.Count != 2 || im.Width != 1 || im.Frame(0)[0] != 49 || im.Frame(1)[7] != 56 || im.Frame(1)[0] != 48 {
		t.Fatalf("count %d width %d frames %v %v", im.Count, im.Width, im.Frame(0), im.Frame(1))
	}
}

// Two frames: a sheet's planes run through all frames, so the second
// 2-byte group is frame 1's plane 0 and the fourth frame 1's plane 1.
func TestAmigaImageSheetPlanes(t *testing.T) {
	b := []byte{0, 1, 0, 2, 0, 0, 0, 0, 0, 2, 0, 2, // h=1 w=2 count=2 plane=2
		0, 0, 0x80, 0, 0, 0, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	im := ParseImage(AmigaImage(b, true, 2), false, 0)
	if im.Count != 4 || im.Frame(2)[0] != 64+3 || im.Frame(0)[0] != 64 {
		t.Fatalf("count %d frames %v %v", im.Count, im.Frame(0), im.Frame(2))
	}
}
