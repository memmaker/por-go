package engine

// The combat main loop and the player's command menus (ovr09).

// Combat = ovr09:007A: set up the battle, then rounds until one side is
// down (or the party stops fighting).
func (e *Engine) Combat() {
	e.beginFlag = 5
	e.combatTargets = true // DS:6B84 = 0068:0066 (ovr14:20C3)
	e.combatSetup()
	done := e.rv.side[0] == 0 || e.rv.side[1] == 0
	for !done {
		e.countSides()
		for _, c := range e.party {
			e.o14_0000(c)
		}
		putWord(e.game, 0x596, 0)
		for c := e.nextActor(); c != nil; c = e.nextActor() {
			e.takeTurn(c)
		}
		done = e.endRound()
	}
	e.combatEnd()
}

// combatEnd = ovr09:0000: free the clouds, the map and the sprite.
func (e *Engine) combatEnd() {
	e.combatTargets = false // DS:6B84 = 0096:0034 (ovr23:0A91)
	e.clouds = nil
	e.cmap = nil
	e.spr = nil
	e.combatViewEnter()
}

// nextActor = ovr09:0131 (var char): the highest initiative left (combat
// block +3); ties go to the higher d100. nil when nobody is left.
func (e *Engine) nextActor() *Character {
	var best, bestRoll byte
	var who *Character
	for _, c := range e.party {
		r := byte(e.Dice(1, 100))
		ini := c.Combat.Delay
		if ini > best {
			bestRoll = r
		}
		if ini >= best && r >= bestRoll {
			bestRoll, best, who = r, ini, c
		}
	}
	if best == 0 {
		return nil
	}
	return who
}

// takeTurn = ovr09:01F6 (char).
func (e *Engine) takeTurn(c *Character) {
	ci := c.Combat
	ci.Attackers, ci.Flank, ci.Guarding = 0, 0, 0
	e.affectGroup(c, 7)
	if int8(ci.Delay) <= 0 {
		return // no cursor erase when the turn is skipped
	}
	if ci.Delay == 0x14 {
		ci.Delay = 0x13
	}
	e.selectChar(c)
	e.combatRedrawChar(c, 2, 1)
	e.ApplyItems(c)
	e.combatPanel(c)
	e.affectGroup(c, 0xF)
	if int8(ci.Delay) > 0 {
		if c.Auto != 0 {
			e.o10_000F(c)
		} else {
			e.playerTurn(c)
		}
	}
	e.eraseCursor(e.combatX(c), e.combatY(c))
}

// selectChar makes c the selected member (DS:5D92).
func (e *Engine) selectChar(c *Character) {
	for i, p := range e.party {
		if p == c {
			e.partySel = i
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// playerTurn = ovr09:031A (char): a spell being cast goes off, otherwise
// the combat menu until the turn is used.
func (e *Engine) playerTurn(c *Character) {
	if c.Okay == 0 {
		e.clearCombatInfo(c)
		return
	}
	ci := c.Combat
	if ci.Spell > 0 {
		s := ci.Spell
		ci.Spell = 0
		var quit bool
		e.CastSpell(s, 0, 1, &quit)
		e.clearCombatInfo(c)
		return
	}
	done := false
	for !done {
		switch e.combatMenu(c) {
		case 'M':
			done = e.moveAttack(c)
		case 'V':
			done = e.ViewCharacter() // ovr20:0AB3 (var done): ItemsMenu sets it
			e.o14_0D23(c)
			if !done {
				e.combatUseHook()
			}
		case 'A':
			var aim aimTarget
			var ok bool
			e.o14_3558(c, 0xFF, 1, 0, &ok, &aim)
			done = ok
		case 'U':
			done = e.ItemsMenu() // ovr20:0EF8 (var done)
			e.o14_0D23(c)
			if !done {
				e.combatUseHook()
			}
		case 'C':
			var ok bool
			e.o14_2414(0, 0, &ok)
			done = ok
		case 'T':
			e.o14_116A(c)
			done = e.clearCombatInfo(c)
		case 'Q':
			e.quickOne(c)
			e.ClearRow24()
			e.ClearKeyboard()
			e.Delay(200)
			done = true
			e.o10_000F(c)
		case 'D':
			done = e.otherMenu(c)
		case '2':
			e.v6E30 ^= 1
			if e.v6E30 != 0 {
				e.BottomMessage("Magic On")
			} else {
				e.BottomMessage("Magic Off")
			}
		case ' ':
			for _, p := range e.party {
				if p.Control < 0x80 {
					p.Auto = 0
				}
			}
		case 0x10:
			ci.Delay = 0x14
			for _, p := range e.party {
				e.quickOne(p)
			}
			e.ClearRow24()
			e.Delay(200)
			done = true
		case '-':
			if e.o14_3B93() {
				e.combatRedrawChar(c, 3, 0)
				done = true
			} else {
				e.BottomMessage("That doesn't work")
			}
		}
		if !done {
			e.combatRedrawChar(c, 2, 1)
			e.combatPanel(c)
		}
	}
}

// combatMenu = ovr09:0666 (char, var key): the main combat menu.
func (e *Engine) combatMenu(c *Character) byte {
	e.affectGroup(c, 0xF)
	ci := c.Combat
	m := ""
	if ci.Moves > 0 {
		m += "Move "
	}
	m += "View Aim "
	if c.ItemCount > 0 {
		m += "Use "
	}
	spells := false
	for i := 0; i <= 0x14; i++ {
		if c.Memorized[i] > 0 {
			spells = true
		}
	}
	if spells && ci.CanCast != 0 && wordAt(e.area, 0x1CA) == 0 {
		m += "Cast "
	}
	if int8(c.Levels[clsCleric]) > 0 && ci.TurnUsed == 0 {
		m += "Turn "
	}
	m += "Quick Done"
	for {
		key, extra := e.DisplayInput("", m, 0xD, 0xA, 0xF, true, false)
		if extra && !(key == 0x10 || key == 0x13 || key == '-' || key == '2') { // set cs:0626
			key = 0
		}
		switch key { // set cs:0646
		case 0x10, 0x13, ' ', '-', '2', 'A', 'C', 'D', 'M', 'Q', 'T', 'U', 'V':
			return key
		}
	}
}

// endRound = ovr09:0880 (var done): the clock, the dying, the end check.
func (e *Engine) endRound() (done bool) {
	e.AdvanceClock(1, 1)
	e.v6DE3++
	e.o14_2879()
	for _, c := range e.party {
		e.affectGroup(c, 0x13)
		e.stinkingCloud(c)
		if c.Status == 5 {
			c.Combat.Dying++
			if c.Combat.Dying > 9 {
				c.Status = 6
			}
		}
	}
	if e.bandage(false) {
		e.BottomMessage("Your Teammate is Dying")
	}
	e.countSides()
	e.combatCentre(e.cmap.viewX+3, e.cmap.viewY+3, 0xFF, 8)
	if e.rv.side[0] == 0 || e.rv.side[1] == 0 {
		done = true
	}
	if e.rv.side[0] > 0 && e.rv.side[1] == 0 && e.v49F4 == 0 {
		if e.YesNo(0xD, 0xA, 0xF, "Continue Battle:") == 'Y' {
			done = false
		}
	}
	return
}

// moveAttack = ovr09:09DF (char, var done): step around with the keypad;
// stepping into someone attacks, off the map flees. Escape undoes the
// moves of this menu.
func (e *Engine) moveAttack(c *Character) (done bool) {
	ci := c.Combat
	moves, face := ci.Moves, ci.Facing
	x, y := e.combatX(c), e.combatY(c)
	key := byte(0x20)
	done = false
	defer e.rawArrows("keypad, arrows, click a square=step, attack|Enter, click self=done|Esc=undo the moves")()
	mark := e.regionMark()
	defer e.regionCut(mark)
	{ // extra (nav.go): a click on a square next to the character steps there
		e.addRegion(region{x0: 1, y0: 1, x1: 21, y1: 21, click: func(cx, cy int) uint16 {
			sx, sy, in := e.combatSquare(cx, cy)
			dx, dy := sx-e.combatX(c), sy-e.combatY(c)
			if !in || dx < -1 || dx > 1 || dy < -1 || dy > 1 {
				return 0
			}
			if dx == 0 && dy == 0 {
				return 13
			}
			dxs, dys := e.X.Bytes(0x2736, 8), e.X.Bytes(0x273F, 8)
			for d := 0; d < 8; d++ {
				if int(int8(dxs[d])) == dx && int(int8(dys[d])) == dy {
					return stepKeys[d]
				}
			}
			return 0
		}})
	}
	for ci.Moves > 1 && key != 0 && key != 0xD {
		e.ClearCells(0x18, 0x18, 0x27, 0x18)
		key, _ = e.DisplayInput("Move/Attack, Move Left = "+itoa(int(ci.Moves>>1))+" ", "", 0xA, 0xA, 0xF, true, false)
		d := 8
		switch key {
		case 0:
			ci.Moves = moves
			e.combatClearIcon(0, 0, e.combatIndex(c))
			done = !e.combatSquareFree(c, x, y, false)
			e.combatCentre(e.combatX(c), e.combatY(c), 0, 8)
			ci.Facing = face
		case 'H':
			d = 0
		case 'I':
			d = 1
		case 'M':
			d = 2
		case 'Q':
			d = 3
		case 'P':
			d = 4
		case 'O':
			d = 5
		case 'K':
			d = 6
		case 'G':
			d = 7
		}
		if d >= 8 {
			continue
		}
		e.combatTurn(c, byte(d), 0, false)
		who, g := e.groundAt(c, d)
		switch {
		case who > 0:
			done = e.attackTarget(c, e.combatant(who), done)
		case g == 0:
			switch e.YesNo(0xD, 0xA, 0xF, "Flee:") {
			case 'Y':
				done = e.o14_0C6A(c)
			case 'N':
				done = false
			}
		case e.groundInfo(g)[0] > ci.Moves:
			e.BottomMessage("Blocked")
		default:
			e.o14_08B2(c, d)
			if c.Okay == 0 {
				done = e.clearCombatInfo(c)
				continue
			}
			e.o14_0712(c, d)
			if c.Okay == 0 {
				done = e.clearCombatInfo(c)
			}
			e.stinkingCloud(c)
			if e.hasAffect286B(c) {
				done = e.clearCombatInfo(c)
			}
		}
	}
	if ci.Moves < 2 {
		ci.Moves = 0
	}
	return
}

// rangedWeapon = 00BA:2E03 (char): the readied weapon shoots (base item
// +0xC above 1).
func (e *Engine) rangedWeapon(c *Character) bool {
	w := c.Slots[0]
	return w != nil && e.baseItem(w.Type, 0xC) > 1
}

// throwable = 00BA:2E46 (char): a ranged weapon that also works in melee
// (base item flags 0x14).
func (e *Engine) throwable(c *Character) bool {
	return e.rangedWeapon(c) && e.baseItem(c.Slots[0].Type, 0xE)&0x14 == 0x14
}

// ammoFor = 00BA:2E8B (char, var item): the item a ranged attack uses up
// (the weapon itself, the arrows in slot 11 or the bolts in slot 12); true
// when there is one or the weapon needs none (flags 0x0A).
func (e *Engine) ammoFor(c *Character, it **Item) bool {
	w := c.Slots[0]
	*it = nil
	var f byte
	if w != nil {
		f = e.baseItem(w.Type, 0xE)
		if f&0x10 != 0 {
			*it = w
		}
		if f&8 != 0 {
			if f&1 != 0 {
				*it = c.Slots[11]
			}
			if f&0x80 != 0 {
				*it = c.Slots[12]
			}
		}
	}
	return *it != nil || f == 0xA
}

// attackTarget = ovr09:0D45 (char, target, var done).
// done passes through when no attack is made.
func (e *Engine) attackTarget(c, t *Character, done bool) bool {
	if e.rangedWeapon(c) && !e.throwable(c) {
		e.BottomMessage("Not with that weapon")
		return done
	}
	if !e.o14_2998(c, t) {
		return done
	}
	c.Combat.Target = t
	if e.o14_0E82(c, t) {
		return true
	}
	e.o14_17F6(c, t)
	return e.o14_1884(c, t, 0, nil)
}

// otherMenu = ovr09:0E45 (char, var done): Done's submenu.
func (e *Engine) otherMenu(c *Character) (done bool) {
	done = false
	m := ""
	if !e.rangedWeapon(c) || e.throwable(c) {
		m += "Guard "
	}
	m += "Delay Quit "
	if e.bandage(false) {
		m += "Bandage "
	}
	m += "Speed Exit"
	for key := byte(0x20); key != 0 && key != 'E' && !done; {
		key, _ = e.DisplayInput("", m, 0xD, 0xA, 0xF, false, false)
		switch key {
		case 'G':
			done = e.guard(c)
		case 'D':
			c.Combat.Delay = 1
			done = true
		case 'Q':
			done = e.clearCombatInfo(c)
		case 'B':
			done = e.bandage(true)
			done = e.clearCombatInfo(c)
		case 'S':
			e.speedMenu()
		}
	}
	return
}

// guard = 00BA:2686 (char): end the turn on guard.
func (e *Engine) guard(c *Character) bool {
	r := e.clearCombatInfo(c)
	c.Combat.Guarding = 1
	e.BottomMessage("Guarding")
	return r
}

// bandage = ovr09:0FFF (doIt): is a party member dying? With doIt the
// first one is bandaged.
func (e *Engine) bandage(doIt bool) bool {
	found := false
	for _, c := range e.party {
		if c.Combat.Extra == 0 && c.Side == 0 && c.Status == 5 {
			found = true
			if doIt {
				c.Status = 4
				c.Combat.Dying = 0
				e.CharMessage(c, "is bandaged", 0xA, true)
				doIt = false
			}
		}
	}
	return found
}

// speedMenu = ovr09:10F6: the message delay.
func (e *Engine) speedMenu() {
	for key := byte(0x20); key != 0 && key != 'E'; {
		m := " "
		if e.msgDelay < 9 {
			m += "Slower "
		}
		if e.msgDelay > 0 {
			m += "Faster "
		}
		m += "Exit"
		key, _ = e.DisplayInput("GameSpeed ("+itoa(int(e.msgDelay))+") :", m, 0xD, 0xA, 0xF, false, false)
		switch key {
		case 'S':
			e.msgDelay++
		case 'F':
			e.msgDelay--
		}
	}
}

// quickOne = ovr09:122B (char): the computer fights for it; a target on
// its own side is dropped.
func (e *Engine) quickOne(c *Character) {
	c.Auto = 1
	if t := c.Combat.Target; t != nil && t.Side == c.Side {
		c.Combat.Target = nil
	}
}

// combatPanel = 00BA:09E3 (char): the combatant's name, hit points, AC,
// readied weapon and status on the right of the combat screen.
func (e *Engine) combatPanel(c *Character) {
	e.ClearCells(0x17, 1, 0x26, 0x15)
	row := 1
	e.CharMessage(c, " ", row, false)
	row++
	e.DisplayString(0x17, row+1, 0xA, "Hitpoints")
	e.displayHP(c, 0x21, row+1, false)
	row += 2
	e.DisplayString(0x17, row+1, 0xA, "AC")
	e.displayAC(c, 0x1A, row+1)
	e.msgRow = row + 1
	if w := c.Slots[0]; w != nil {
		row += 2
		e.ItemName(c, w, 0, 0, false, false)
		e.DisplayTextBlock(0x17, row+1, 0x26, row+3, 0xA, true, w.Name())
	}
	row = e.msgRow + 1
	if c.Okay == 0 {
		e.DisplayString(0x17, row+1, 0xF, e.X.PString(0x1047+0xD*int(c.Status)))
	} else if e.hasAffect286B(c) {
		e.DisplayString(0x17, row+1, 0xF, "(Helpless)")
	}
}

// combatUseHook = 00BA:2B88: redraw the combat screen: the map and panel
// frames, rows 23-24 cleared, the map centred on the view.
func (e *Engine) combatUseHook() {
	e.combatViewLeave() // ovr32:00E3: the combat palette
	e.DrawBoxTitled(1, 1, 0x15, 0x15, 0, 0, "")
	e.DrawBoxTitled(0x17, 1, 0x26, 0x15, 0, 0, "")
	e.ClearCells(0, 0x17, 0x27, 0x18)
	e.combatCentre(e.cmap.viewX+3, e.cmap.viewY+3, 0xFF, 8)
}
