package engine

import (
	"testing"

	"por/exe"
)

func TestLineAndMissile(t *testing.T) {
	x, err := exe.Load("../../START.EXE")
	if err != nil {
		t.Skip(err)
	}
	e := &Engine{X: x, cmap: &combatMap{}}
	l := newLine(0, 0, 6, -3)
	var dirs []byte
	for e.step(l) {
		dirs = append(dirs, l.dir)
	}
	if l.x != 6 || l.y != -3 || len(dirs) != 6 || l.dir != 8 || dirs[0] != 1 || dirs[1] != 2 {
		t.Fatalf("line ended at %d,%d dirs %v last %d", l.x, l.y, dirs, l.dir)
	}
	e.flyMissile(1, 1, 9, 4, 2, 0) // must finish with the map stubs
}
