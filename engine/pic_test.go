package engine

import (
	"testing"

	"por/dax"
	"por/ega"
)

func TestDrawPictureMatchesSet(t *testing.T) {
	im := &dax.Image{Width: 2, Height: 3, Count: 1, BPP: 48, Mask: 13}
	for i := 0; i < 48; i++ {
		im.Data = append(im.Data, []byte{1, 13, 16, 20, 0, 40, 7, 5}[i%8])
	}
	for _, pos := range [][2]int{{0, 0}, {-1, -1}, {39, 24}, {38, 23}, {5, 7}} {
		for mode := 0; mode < 2; mode++ {
			a, b := &Engine{S: ega.New()}, &Engine{S: ega.New()}
			a.DrawPictureMasked(pos[0], pos[1], im, 0, mode)
			for r := 0; r < 3; r++ {
				for c := 0; c < 16; c++ {
					if p := im.Data[r*16+c]; mode == 0 || int(p) != im.Mask {
						b.S.Set(pos[0]*8+c, pos[1]*8+r, p)
					}
				}
			}
			if a.S.Pix != b.S.Pix {
				t.Fatalf("pos %v mode %d differs from Set", pos, mode)
			}
		}
	}
}
