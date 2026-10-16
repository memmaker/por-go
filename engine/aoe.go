package engine

// Area of effect preview (extra): while the player aims a spell that hits
// everyone round the target square (o14_20C3's kinds 9..E), the combatants
// it would hit are marked with the cursor icon and counted in the prompt.
// The test is the cast's own: buildNear round the square, radius kind&7.
// ponytail: Lightning Bolt (kind 8, a line) gets no preview; boltPath
// would need a dry run.

// aoeMarks marks who the spell aimed at (x,y) would hit; the prompt says
// how many on each side. undo takes the marks off again.
func (e *Engine) aoeMarks(x, y int, prompt string) (string, func()) {
	if e.ext.aoe == 0 {
		return prompt, func() {}
	}
	saved := e.near.list
	e.near.list = nil
	e.buildNear(x, y, e.ext.aoe-1, 0xFF, 1)
	hit := e.near.list
	e.near.list = saved
	foes, friends := 0, 0
	for _, n := range hit {
		p := e.cpos[n.index]
		if c := e.combatant(n.index); c != nil && c.Side == e.selected().Side {
			friends++
		} else {
			foes++
		}
		if sx, sy := int(p.x)-e.cmap.viewX, int(p.y)-e.cmap.viewY; combatInView(sx, sy) {
			e.drawIcon(sx, sy, 0, 0, 0x19)
			e.drawOccupant(int(p.x), int(p.y))
		}
	}
	undo := func() {
		for _, n := range hit {
			p := e.cpos[n.index]
			if int(p.x) != x || int(p.y) != y || e.cmap.cursor == 0 {
				e.eraseCursor(int(p.x), int(p.y))
			}
		}
	}
	return "Hits " + itoa(foes) + " foe " + itoa(friends) + " ally ", undo
}
