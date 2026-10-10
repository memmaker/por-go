package engine

// Derived character values, ported from ovr24.

const (
	chWis       = 0x12
	chDex       = 0x13
	chMaxLevel  = 0x73 // highest class level
	chHitDice   = 0x2D // from the DS:3C7C table
	chSpellBook = 0x32 // +n: spell n known (1..0x38)
	chThief     = 0x76 // +1..8: thief skills
	chAttacks   = 0xA1
	chSlots     = 0xB1 // +1..3 cleric, +4..6 magic-user spell slots
	chLevel     = 0x96 // 8 class levels: cleric druid fighter paladin ranger mage thief monk
)

// RecomputeCharacter = ovr24:0000: hit dice, highest level, attacks,
// spell slots, known cleric spells, saving throws and thief skills from
// the class levels.
func (e *Engine) RecomputeCharacter(c *Character) {
	c.Raw[chHitDice] = 0
	for i := 0; i <= 7; i++ {
		lvl := int(int8(c.Raw[chLevel+i]))
		if v := e.X.Bytes(0x3C7C+i*0xB+lvl, 1)[0]; v > c.Raw[chHitDice] {
			c.Raw[chHitDice] = v
		}
		if int(int8(c.Raw[chMaxLevel])) < lvl {
			c.Raw[chMaxLevel] = byte(lvl)
		}
		switch i {
		case 2: // fighter
			if lvl > 6 {
				c.Raw[chAttacks] = 3
			} else {
				c.Raw[chAttacks] = 2
			}
		case 0: // cleric
			if lvl > 1 {
				for j := 1; j <= 3; j++ {
					c.Raw[chSlots+j] = e.X.Bytes(0x40BC+lvl*3+j, 1)[0]
				}
				e.wisdomSlots(c)
				for k := 1; k <= 0x38; k++ {
					rec := e.X.Bytes(0x31FA+k*16, 2)
					if rec[0] == 0 && c.Raw[chSlots+int(int8(rec[1]))] > 0 {
						c.Raw[chSpellBook+k] = 1
					}
				}
			}
		case 5: // magic-user
			if lvl > 1 {
				for j := 1; j <= 3; j++ {
					c.Raw[chSlots+3+j] = e.X.Bytes(0x41D4+lvl*3+j, 1)[0]
				}
			}
		}
	}
	e.savingThrows(c)
	if int8(c.Raw[chLevel+6]) > 0 {
		e.thiefSkills(c)
	}
}

// wisdomSlots = ovr24:01AF: bonus cleric spells for wisdom 13..17.
func (e *Engine) wisdomSlots(c *Character) {
	w := c.Raw[chWis]
	for _, b := range []struct {
		min  byte
		slot int
	}{{0xC, 1}, {0xD, 1}, {0xE, 2}, {0xF, 2}, {0x10, 3}} {
		if w > b.min && c.Raw[chSlots+b.slot] > 0 {
			c.Raw[chSlots+b.slot]++
		}
	}
}

// savingThrows = ovr24:0255: the best of each class's table (DS:426E).
func (e *Engine) savingThrows(c *Character) {
	for s := 0; s <= 4; s++ {
		c.Raw[chSaves+s] = 0x14
		for cl := 0; cl <= 7; cl++ {
			lvl := int(int8(c.Raw[chLevel+cl]))
			if lvl <= 0 {
				continue
			}
			if v := e.X.Bytes(0x426E+cl*0x2D+lvl*5+s, 1)[0]; c.Raw[chSaves+s] > v {
				c.Raw[chSaves+s] = v
			}
		}
	}
}

// thiefSkills = ovr24:0329: level table DS:3CF3 plus race DS:3D3B, plus
// dexterity DS:3D4E for the first five skills.
func (e *Engine) thiefSkills(c *Character) {
	race := int(int8(c.Raw[chRace]))
	lvl := int(int8(c.Raw[chLevel+6]))
	for k := 1; k <= 8; k++ {
		r := int(int8(e.X.Bytes(0x3D3B+race*8+k, 1)[0]))
		t := int(e.X.Bytes(0x3CF3+lvl*8+k, 1)[0])
		if r < 0 && -r > t {
			c.Raw[chThief+k] = 0
			continue
		}
		c.Raw[chThief+k] = byte(t + r)
		if k < 6 {
			c.Raw[chThief+k] += byte(int8(e.X.Bytes(0x3D4E+int(c.Raw[chDex])*5+k, 1)[0]))
		}
	}
}

// Readied-item slots at char+0xCC (13 far pointers). Slot numbers come
// from the base item record; 9 and 10 are the two rings, 11 arrows
// (type 0x49), 12 quarrels (type 0x1C).
const (
	slotWeapon  = 0
	slotRing1   = 9
	slotRing2   = 10
	slotArrows  = 11
	slotBolts   = 12
	chItemCount = 0xC7
	chHands     = 0x100
	chWeight    = 0x102 // word
	chMove      = 0x11C
	chBaseMove  = 0x72
	chBaseAC    = 0xA9
	chAC        = 0x111
	chRearAC    = 0x112
	chCoins     = 0x88 // 7 words
	chStrBonus  = 0xAA // non-zero: strength modifies hit and damage
)

// baseItem returns byte k of the DS:5582 base record of an item type.
func (e *Engine) baseItem(typ byte, k int) byte {
	if o := int(typ)*16 + k; o < len(e.items) {
		return e.items[o]
	}
	return 0
}

// readiedWeight = DS:4846.
var readiedWeight uint16

// ApplyItems = 00BA:0BB8: everything that follows from the items carried
// and readied: slots, weight, hands, attack and damage, armour class,
// movement and saving throw bonus.
func (e *Engine) ApplyItems(c *Character) {
	r := &c.Raw
	c.Slots = [13]*Item{}
	bag := false
	r[chItemCount] = 0
	r[chHands] = 0
	putWord(r[:], chWeight, 0)
	readiedWeight = 0
	for _, it := range c.Items {
		r[chItemCount]++
		w := wordAt(it.Raw[:], 0x37)
		if it.Raw[itCount] > 0 {
			w *= uint16(it.Raw[itCount])
		}
		putWord(r[:], chWeight, wordAt(r[:], chWeight)+w)
		if it.Raw[itReadied] == 0 {
			continue
		}
		readiedWeight += w
		typ := it.Raw[itType]
		switch s := e.baseItem(typ, 0); {
		case s <= 8:
			c.Slots[s] = it
		case s == 9:
			if c.Slots[slotRing1] == nil {
				c.Slots[slotRing1] = it
			} else if c.Slots[slotRing2] == nil {
				c.Slots[slotRing2] = it
			}
		}
		if typ == 0x49 {
			c.Slots[slotArrows] = it
		}
		if typ == 0x1C {
			c.Slots[slotBolts] = it
		}
		if it.Raw[itWords] == 0xBA {
			bag = true
		}
		r[chHands] += e.baseItem(typ, 1)
	}
	for i := 1; i <= 2; i++ {
		r[0x114+i] = r[0xA2+i]
		r[0x116+i] = r[0xA4+i]
		r[0x118+i] = r[0xA6+i]
	}
	var ac [5]int8
	ringOnly := false
	r[chSaveBonus] = 0
	r[chAC] = r[chBaseAC]
	r[chMove] = r[chBaseMove]
	r[chTHAC0] = r[chHitDice]
	ac[0] = dexAC(r[chDex])
	if c.Slots[slotWeapon] == nil {
		r[chTHAC0] += byte(strHit(c))
		r[0x119] += byte(strDamage(c))
	}
	e.weaponStats(c)
	for _, it := range c.Items {
		if it.Raw[itReadied] == 0 {
			continue
		}
		e.armourMove(it, c)
		e.itemAC(c, it, &ac, &ringOnly)
	}
	if ringOnly {
		ac[3] = 0
	}
	if bag {
		if wordAt(r[:], chWeight) < 5000 {
			putWord(r[:], chWeight, 0)
		} else {
			putWord(r[:], chWeight, wordAt(r[:], chWeight)-5000)
		}
		if wordAt(r[:], chWeight) < readiedWeight {
			putWord(r[:], chWeight, readiedWeight)
		}
	}
	e.encumbrance(c)
	if int(ac[4]) < int(r[chAC]) {
		ac[4] = int8(r[chAC])
	}
	r[chAC] = 0
	for i := 0; i <= 4; i++ {
		r[chAC] += byte(ac[i])
	}
	r[chRearAC] = byte(int(ac[4]) + int(ac[2]) + int(ac[3]) - 2)
	if int8(r[chLevel+2]) > 0 && int8(r[chRace]) > 0 {
		r[0x6B] = r[chLevel+2]
	} else {
		r[0x6B] = 1
	}
}

// weaponStats = 00BA:0000: to-hit and damage of the readied weapon.
func (e *Engine) weaponStats(c *Character) {
	w := c.Slots[slotWeapon]
	if w == nil {
		return
	}
	r := &c.Raw
	typ := w.Raw[itType]
	flags := e.baseItem(typ, 0xE)
	r[chTHAC0] = r[chHitDice]
	if flags&2 != 0 { // missile
		r[chTHAC0] += byte(dexMissile(r[chDex]))
	}
	r[0x119] = e.baseItem(typ, 0xB)
	if flags&4 != 0 { // strength applies
		r[chTHAC0] += byte(strHit(c))
		r[0x119] += byte(strDamage(c))
	}
	plus := int8(w.Raw[itPlus])
	if flags > 0x7F && c.Slots[slotBolts] != nil {
		plus += int8(c.Slots[slotBolts].Raw[itPlus])
	}
	if flags&1 != 0 && c.Slots[slotArrows] != nil {
		plus += int8(c.Slots[slotArrows].Raw[itPlus])
	}
	r[0x119] += byte(plus)
	if r[chRace] == 2 && (w.Raw[itType] > 0x28 && w.Raw[itType] < 0x2D || w.Raw[itType] == 0x25 || w.Raw[itType] == 0x24) {
		plus++ // elves with bows and swords
	}
	r[chTHAC0] += byte(plus)
	r[0x115] = e.baseItem(typ, 9)
	r[0x117] = e.baseItem(typ, 0xA)
}

// armourMove = 00BA:01FB (char, item): body armour sets the movement by
// its weight; magical armour moves 3 faster up to 9.
func (e *Engine) armourMove(it *Item, c *Character) {
	if e.baseItem(it.Raw[itType], 0) != 2 {
		return
	}
	switch w := wordAt(it.Raw[:], 0x37); {
	case w <= 0x96:
		c.Raw[chMove] = c.Raw[chBaseMove]
	case w <= 0x18F:
		c.Raw[chMove] = 9
	default:
		c.Raw[chMove] = 6
	}
	if it.Raw[itPlus] != 0 && c.Raw[chMove] <= 9 {
		c.Raw[chMove] += 3
	}
}

// itemAC = 00BA:0282 (char, item, ac, ringOnly): ac[1] shield, ac[2]
// protective items, ac[3] best ring, ac[4] best armour; magical body
// armour cancels rings.
func (e *Engine) itemAC(c *Character, it *Item, ac *[5]int8, ringOnly *bool) {
	typ := it.Raw[itType]
	v := e.baseItem(typ, 6)
	if v <= 0x7F {
		return
	}
	v &= 0x7F
	slot := e.baseItem(typ, 0)
	plus := int8(it.Raw[itPlus])
	switch {
	case slot == 1:
		ac[1] = int8(v) + plus
	case v == 0:
		if slot == 9 {
			if plus > ac[3] {
				ac[3] = plus
			}
		} else {
			ac[2] += plus
		}
		c.Raw[chSaveBonus] += it.Raw[itPlus2]
	case int(v)+int(plus) > int(ac[4]):
		ac[4] = int8(v) + plus
		if plus > 0 && slot == 2 {
			*ringOnly = true
		}
	}
}

// encumbrance = 00BA:03A0: movement by the weight over the allowance.
func (e *Engine) encumbrance(c *Character) {
	over := int(int16(wordAt(c.Raw[:], chWeight) - uint16(strCarry(c))))
	if over < 0 {
		over = 0
	}
	m := byte(3)
	switch {
	case over <= 0x200:
		m = c.Raw[chMove]
	case over <= 0x300:
		m = 9
	case over <= 0x400:
		m = 6
	}
	if m < c.Raw[chMove] {
		c.Raw[chMove] = m
	}
}

// dexAC = 00BA:10E6.
func dexAC(d byte) int8 {
	switch {
	case d >= 1 && d <= 3:
		return -4
	case d >= 4 && d <= 6:
		return int8(d) - 7
	case d >= 15 && d <= 18:
		return int8(d) - 14
	case d >= 19 && d <= 20:
		return 4
	case d >= 21 && d <= 23:
		return 5
	case d >= 24 && d <= 25:
		return 6
	}
	return 0
}

// dexMissile = 00BA:1168.
func dexMissile(d byte) int8 {
	switch {
	case d <= 2:
		return -4
	case d <= 5:
		return int8(d) - 6
	case d >= 16 && d <= 18:
		return int8(d) - 15
	case d >= 19 && d <= 20:
		return 3
	case d >= 21 && d <= 23:
		return 4
	case d >= 24 && d <= 25:
		return 5
	}
	return 0
}

// strIndex = 00BA:11EA: strength as one scale (18/xx = 18..23, 19+ = 24+).
func strIndex(c *Character) byte {
	s, x := c.Raw[chStr], c.Raw[chStr00]
	switch {
	case s <= 0x11:
		return s
	case s == 0x12:
		switch {
		case x == 0:
			return 0x12
		case x <= 0x32:
			return 0x13
		case x <= 0x4B:
			return 0x14
		case x <= 0x5A:
			return 0x15
		case x <= 0x63:
			return 0x16
		case x == 0x64:
			return 0x17
		}
	case s <= 0x19:
		return s + 5
	}
	return 0 // the original returns an unset local here
}

// strHit = 00BA:1282.
func strHit(c *Character) int8 {
	if c.Raw[chStrBonus] == 0 {
		return 0
	}
	switch i := strIndex(c); {
	case i >= 1 && i <= 3:
		return -3
	case i >= 4 && i <= 5:
		return -2
	case i >= 6 && i <= 7:
		return -1
	case i >= 0x11 && i <= 0x13:
		return 1
	case i >= 0x14 && i <= 0x16:
		return 2
	case i >= 0x17 && i <= 0x19:
		return 3
	case i >= 0x1A && i <= 0x1B:
		return 4
	case i >= 0x1C && i <= 0x1E:
		return int8(i) - 0x17
	}
	return 0
}

// strDamage = 00BA:1325.
func strDamage(c *Character) int8 {
	if c.Raw[chStrBonus] == 0 {
		return 0
	}
	switch i := strIndex(c); {
	case i >= 1 && i <= 2:
		return -2
	case i >= 3 && i <= 5:
		return -1
	case i == 0x10:
		return 1
	case i >= 0x11 && i <= 0x13:
		return int8(i) - 0x10
	case i >= 0x14 && i <= 0x1D:
		return int8(i) - 0x11
	case i == 0x1E:
		return 0xE
	}
	return 0
}

// strCarry = 00BA:13AB: weight allowance change.
func strCarry(c *Character) int16 {
	switch i := int16(strIndex(c)); {
	case i >= 1 && i <= 3:
		return -350
	case i >= 4 && i <= 5:
		return -250
	case i >= 6 && i <= 7:
		return -150
	case i >= 0xC && i <= 0xD:
		return 100
	case i >= 0xE && i <= 0xF:
		return 200
	case i == 0x10:
		return 350
	case i >= 0x11 && i <= 0x15:
		return (i-0x11)*250 + 500
	case i >= 0x16 && i <= 0x1A:
		return (i-0x16)*1000 + 2000
	case i == 0x1B:
		return 7500
	case i >= 0x1C && i <= 0x1E:
		return (i-0x1C)*3000 + 9000
	}
	return 0
}

// hasAffect286B = 00BA:0B73: any of the four affects in DS:286C.
func (e *Engine) hasAffect286B(c *Character) bool {
	found := false
	for i := 1; i <= 4; i++ {
		if e.FindAffect(c, e.X.Bytes(0x286B+i, 1)[0]) != nil {
			found = true
		}
	}
	return found
}

// atRaceLimit = 00BA:2F57 (char, class): the demihuman level limit is
// reached (fighter limits depend on strength 17/18; race 4 clerics stop at 5).
func atRaceLimit(c *Character, class int) bool {
	lvl, str := int8(c.Raw[chLevel+class]), c.Raw[chStr]
	if lvl <= 0 {
		return false
	}
	fighter := class == 2
	switch c.Raw[chRace] {
	case 1:
		return fighter && (lvl == 8 && str == 0x11 || lvl == 7 && str < 0x11)
	case 2:
		return fighter && (lvl == 7 || lvl == 6 && str == 0x11 || lvl == 5 && str < 0x11)
	case 3:
		return fighter && (lvl == 6 || lvl == 5 && str < 0x12)
	case 4:
		if class == 0 {
			return lvl == 5
		}
		return fighter && (lvl == 8 || lvl == 7 && str == 0x11 || lvl == 6 && str < 0x11)
	case 5:
		return fighter && (lvl == 6 || lvl == 5 && str == 0x11 || lvl == 4 && str < 0x11)
	}
	return false
}
