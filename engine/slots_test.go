package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlotName(t *testing.T) {
	for in, want := range map[string]string{"before boss": "BEFORE BOSS", "../x/y": "XY", " a:b ": "AB", "": ""} {
		if got := slotName(in); got != want {
			t.Errorf("slotName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAdoptLettered(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"savgamB.dat", "CHRDATB1.SAV", "CHRDATB1.ITM", "CHRDATA1.SAV", "ROSTER.SAV"} {
		os.WriteFile(filepath.Join(dir, f), nil, 0o644)
	}
	e := &Engine{}
	e.Cfg.SaveDir = dir
	if got := e.slots(); len(got) != 1 || got[0].name != "GAME B" {
		t.Fatalf("slots = %v, want GAME B", got)
	}
	for _, f := range []string{"saves/GAME B/SAVGAMA.DAT", "saves/GAME B/CHRDATB1.SAV", "saves/GAME B/CHRDATB1.ITM", "savgamB.dat", "CHRDATB1.SAV", "CHRDATA1.SAV"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s missing", f)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "saves/GAME B/CHRDATA1.SAV")); err == nil {
		t.Error("another letter's party was copied")
	}
	os.RemoveAll(filepath.Join(dir, "saves/GAME B"))
	if got := e.slots(); len(got) != 0 {
		t.Errorf("a deleted slot came back: %v", got)
	}
}

// An F9 save leaves the area block as it was: saveSlot packs the wall slots
// into area+0x3F4.., and area+0x3FE is the view animation flag.
func TestSaveToKeepsArea(t *testing.T) {
	e := &Engine{area: make([]byte, 0x800), game: make([]byte, 0x800), aux: make([]byte, 0x400), ecl: make([]byte, 0x1E00), symDAX: "8X8D1"}
	e.Cfg.SaveDir = t.TempDir()
	e.mapPairB = [3]int{5, 6, 7}
	e.saveTo("X")
	if w := wordAt(e.area, 0x3FE); w != 0 {
		t.Errorf("area+0x3FE = %d after a save, want 0", w)
	}
	if _, err := os.Stat(filepath.Join(e.slotRoot(), "X", "SAVGAMA.DAT")); err != nil {
		t.Error("no save written")
	}
}
