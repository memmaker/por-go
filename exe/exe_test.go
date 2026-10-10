package exe

import "testing"

func TestLoad(t *testing.T) {
	im, err := Load("/Users/felix/Projects/POR/START.EXE")
	if err != nil {
		t.Fatal(err)
	}
	if len(im.Mem) != 68592 {
		t.Fatalf("len %d", len(im.Mem))
	}
	if s := im.PString(0x5F0); s != "C D M T V A R L S B E J" {
		t.Fatalf("DS:75B = %q", s)
	}
	if s := im.PString(0x5F0); s != "C D M T V A R L S B E J" {
		t.Fatalf("DS:5F0 = %q", s)
	}
}
