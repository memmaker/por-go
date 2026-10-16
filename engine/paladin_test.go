package engine

import "testing"

// Paladin and ranger rules ported from CotAB.
func TestPaladinRanger(t *testing.T) {
	e := &Engine{}
	pal := &Character{}
	pal.Levels[clsPaladin] = 7
	e.RecomputeCharacter(pal)
	if pal.BaseAttack[0] != 3 || pal.Saves[0] != 8 || !fighterKind(pal) {
		t.Errorf("paladin 7: attacks %d saves %v", pal.BaseAttack[0], pal.Saves)
	}
	rng := &Character{}
	rng.Levels[clsRanger] = 7
	e.RecomputeCharacter(rng)
	if rng.BaseAttack[0] != 2 {
		t.Errorf("ranger 7: attacks %d", rng.BaseAttack[0])
	}
	if xpTable[clsPaladin][9] != -1 || xpTable[clsRanger][8] != -1 || xpTable[clsRanger][7] != 90001 {
		t.Error("level caps")
	}
	if raceClasses[7][0] != 6 || raceClasses[4][0] != 13 || classAlignments[4][0] != 3 {
		t.Error("class lists")
	}
	if !levelBlocked(4, 0x11, clsRanger, 7) || levelBlocked(7, 0x11, clsRanger, 7) {
		t.Error("half-elf ranger limit")
	}
}
