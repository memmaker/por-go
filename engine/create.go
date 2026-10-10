package engine

// Character creation and dropping, ported from ovr17.

// charMgmtState holds ovr17 state (filled in by the ovr17 port).
type charMgmtState struct{}

// freeCharacter = ovr17:0000 (var c): disposes the combat block, the item
// and affect lists and the record, then nils the pointer. Go's collector
// does the freeing.
func (e *Engine) freeCharacter(c **Character) {
	if *c != nil {
		(*c).Combat, (*c).Items, (*c).Affects = nil, nil, nil
	}
	*c = nil
}

// selectTemp makes c the selected character (DS:5D92 = c) although it is not
// in the party; the returned func restores the old selection.
func (e *Engine) selectTemp(c *Character) func() {
	sel, n := e.partySel, len(e.party)
	e.party = append(e.party, c)
	e.partySel = n
	e.tempSel = c
	return func() {
		e.party = e.party[:n]
		e.partySel = sel
		e.tempSel = nil
	}
}

// CreateCharacter = ovr17:056C: pick race, gender, class and alignment, roll
// the abilities (best of six 3d6 each, then the race/class/age limits), hit
// points and starting spells; keep it, name it, pick the pictures, save it.
func (e *Engine) CreateCharacter() {
	c := &Character{}
	ds := func(off int) int { return int(e.X.Bytes(off, 1)[0]) }
	dsWord := func(off int) int {
		b := e.X.Bytes(off, 2)
		return int(b[0]) | int(b[1])<<8
	}
	for i := 1; i <= 6; i++ {
		t := ds(0x3CF5 + i)
		c.IconColour[i-1] = byte((t+8)<<4 + t)
	}
	c.BaseAC = 0x32
	c.HitDice = 0x28
	c.Status = 0
	c.Okay = 1
	c.PortHead = 1
	c.PortBody = 1
	c.Size = 1
	c.RandomID = byte(e.Random(0x100))

	// pick runs one of the four selection menus; false = left with Esc/Exit,
	// which abandons the whole creation.
	pick := func(list []*ListNode) (int, bool) {
		if q := e.ext.quick; q != nil {
			return q.choose(list)
		}
		index, redraw := 1, true
		var sel *ListNode
		for {
			k := e.ListMenu("", "", 0xD, 0xA, 0xF, 1, 2, 0x26, 0x16, list, true, &redraw, &index, &sel)
			if k == 0 {
				return 0, false
			}
			if k == 0xD {
				return index, true
			}
		}
	}

	// Race.
	list := newList(7)
	list[0].Flag, list[0].Text = 1, "Pick Race"
	raceLoop := 0 // the loop variable, left at 7 and used below (quirk)
	for raceLoop = 1; raceLoop <= 7; raceLoop++ {
		if raceLoop >= 1 && raceLoop <= 5 {
			list[raceLoop].Text = "  " + e.X.PString(0xF03+raceLoop*10)
		} else if raceLoop == 7 {
			list[raceLoop-1].Text = "  " + e.X.PString(0xF03+raceLoop*10)
		}
	}
	raceLoop = 7 // for-loop exit value
	idx, ok := pick(list)
	if !ok {
		return
	}
	if idx == 6 {
		idx++
	}
	c.Race = byte(idx)
	switch c.Race {
	case 5:
		c.IconSize = 1
		e.AddAffect(c, 0x5A, 0, 0xFF, 0)
		e.AddAffect(c, 0x61, 0, 0xFF, 0)
	case 1:
		c.IconSize = 1
		for _, t := range []byte{0x5A, 0x61, 0x1A, 0x2F} {
			e.AddAffect(c, t, 0, 0xFF, 0)
		}
	case 3:
		c.IconSize = 1
		for _, t := range []byte{0x61, 0x12, 0x2F, 0x30} {
			e.AddAffect(c, t, 0, 0xFF, 0)
		}
	case 2:
		c.IconSize = 2
		e.AddAffect(c, 0x6B, 0, 0xFF, 0)
	case 4:
		c.IconSize = 2
		e.AddAffect(c, 0x7C, 0, 0xFF, 0)
	default:
		c.IconSize = 2
	}

	// Gender.
	list = newList(3)
	list[0].Flag, list[0].Text = 1, "Pick Gender"
	for i := 0; i <= 1; i++ {
		list[i+1].Text = "  " + e.X.PString(0xFEC+i*7)
	}
	if idx, ok = pick(list); !ok {
		return
	}
	c.Sex = byte(idx - 1)

	// Class.
	race := int(int8(c.Race))
	cnt := ds(0x3E17 + race*12)
	list = newList(cnt + 1)
	list[0].Flag, list[0].Text = 1, "Pick Class"
	for k := 1; k <= cnt; k++ {
		list[k].Text = "  " + e.X.PString(0xD38+0x1B*int(int8(byte(ds(0x3E17+race*12+k)))))
	}
	if idx, ok = pick(list); !ok {
		return
	}
	c.Class = byte(ds(0x3E17 + race*12 + idx))
	c.MaxLevel = 1
	class := int(int8(c.Class))
	switch {
	case class >= 0 && class <= 7:
		c.Levels[class] = 1
	case class == 8:
		c.Levels[clsCleric], c.Levels[clsFighter] = 1, 1
	case class == 9:
		c.Levels[clsCleric], c.Levels[clsFighter], c.Levels[clsMage] = 1, 1, 1
	case class == 10:
		c.Levels[clsCleric], c.Levels[clsRanger] = 1, 1
	case class == 11:
		c.Levels[clsCleric], c.Levels[clsMage] = 1, 1
	case class == 12:
		c.Levels[clsCleric], c.Levels[clsThief] = 1, 1
	case class == 13:
		c.Levels[clsFighter], c.Levels[clsMage] = 1, 1
	case class == 14:
		c.Levels[clsFighter], c.Levels[clsThief] = 1, 1
	case class == 15:
		c.Levels[clsFighter], c.Levels[clsMage], c.Levels[clsThief] = 1, 1, 1
	case class == 16:
		c.Levels[clsMage], c.Levels[clsThief] = 1, 1
	}
	if int8(c.Levels[clsThief]) > 0 {
		e.thiefSkills(c) // ovr24:0329
	}
	c.ClassMask = 0
	c.HitDice = 0
	for k := 0; k <= 7; k++ {
		if int8(c.Levels[k]) > 0 {
			if t := byte(ds(0x3C7C + k*11 + int(int8(c.Levels[k])))); t > c.HitDice {
				c.HitDice = t
			}
			c.ClassMask += byte(ds(0x5D6 + k))
		}
	}
	e.savingThrows(c) // ovr24:0255

	// Alignment.
	cnt = ds(0x3FE7 + class*10)
	list = newList(cnt + 1)
	list[0].Flag, list[0].Text = 1, "Pick Alignment"
	for k := 1; k <= cnt; k++ {
		list[k].Text = "  " + e.X.PString(0xF53+0x11*int(int8(byte(ds(0x3FE7+class*10+k)))))
	}
	if idx, ok = pick(list); !ok {
		return
	}
	c.Alignment = byte(ds(0x3FE7 + class*10 + idx))

	// Starting age: single classes roll the dice, multi-classes use the
	// maximum. Class 10 (cleric/ranger) has no case, so its age stays 0.
	age := 0
	base := 0x3E5B + race*0x1C
	if class <= 7 {
		b := base + class*4
		age = dsWord(b) + int(byte(e.Dice(ds(b+2), ds(b+3))))
	} else {
		o := -1
		switch class {
		case 8, 9, 0xB, 0xC:
			o = base
		case 0xD, 0xF, 0x10:
			o = base + 0x14
		case 0xE:
			o = base + 8
		}
		if o >= 0 {
			age = dsWord(o) + ds(o+2)*ds(o+3)
		}
	}
	c.Age = uint16(age)

	undo := e.selectTemp(c)
	e.CharacterSheet() // ovr20:004A
	for {
		c.Str00 = 0
		for i := 0; i <= 5; i++ {
			st := &c.Stats[i]
			*st = 0
			for n := 1; n <= 6; n++ {
				roll := byte(e.Dice(3, 6))
				// raceLoop is 7 here, so none of these ever applies.
				switch raceLoop {
				case 1:
					if i == 4 {
						roll++
					} else if i == 5 {
						roll--
					}
				case 2:
					if i == 3 {
						roll++
					} else if i == 4 {
						roll--
					}
				case 5:
					if i == 0 {
						roll--
					} else if i == 3 {
						roll++
					}
				case 6:
					if i == 0 || i == 4 {
						roll++
					} else if i == 5 {
						roll -= 2
					}
				}
				if *st < roll {
					*st = roll
				}
			}
			// Age steps DS:3F31 + race*10 (4 words) against the age word.
			older := func(j int) bool { return dsWord(0x3F31+race*10+2*j) < int(c.Age) }
			gender := int(int8(c.Sex))
			switch i {
			case 0:
				if *st != 0 && older(0) {
					*st++
				}
				if *st != 0 && older(1) {
					*st--
				}
				if *st != 0 && older(2) {
					*st -= 2
				}
				if *st != 0 && older(3) {
					*st--
				}
			case 1:
				if older(1) {
					*st++
				}
				if older(3) {
					*st++
				}
			case 2:
				if !older(0) {
					*st--
				}
				if older(1) {
					*st++
				}
				if older(2) {
					*st++
				}
				if older(3) {
					*st++
				}
			case 3:
				if older(2) {
					*st -= 2
				}
				if older(3) {
					*st--
				}
			case 4:
				if older(1) {
					*st--
				}
				if older(2) {
					*st--
				}
				if older(3) {
					*st--
				}
			}
			// Race limits (DS:3DA3 + race*16; strength is by gender).
			lo, hi := 0x3DA9+2*(i-1)+race*16, 0x3DAA+2*(i-1)+race*16
			if i == 0 {
				lo, hi = 0x3DA3+race*16+gender, 0x3DA5+race*16+gender
			}
			if int(*st) < ds(lo) {
				*st = byte(ds(lo))
			}
			if int(*st) > ds(hi) {
				*st = byte(ds(hi))
			}
			// Class minimum (DS:3F81 + class*6).
			if m := ds(0x3F81 + class*6 + i); int(*st) < m {
				*st = byte(m)
			}
			if i == 0 {
				if *st == 0x12 && int8(c.Levels[clsFighter]) > 0 {
					c.Str00 = byte(e.Random(100) + 1)
					if m := byte(ds(0x3DA7 + race*16 + gender)); m < c.Str00 {
						c.Str00 = m
					}
				}
			}
			if i == 2 && *st < 0xD && class >= 8 && class <= 0xC { // set [8..12] at cs:0525
				*st = 0xD
			}
			e.showStat(i, false) // ovr20:0901
		}

		// Hit points: one die per class, averaged, with the constitution
		// adjustment.
		c.HP = c.MaxHP
		c.BaseAttack[0], c.BaseDice[0], c.BaseSides[0], c.StrBonus, c.BaseMove = 2, 1, 2, 1, 0xC
		var sum, count byte
		for k := 0; k <= 7; k++ {
			if int8(c.Levels[k]) <= 0 {
				continue
			}
			if k == 0 {
				c.SpellSlots[0] = 1
			} else if k == 5 {
				c.SpellSlots[3] = 1
			}
			sum += byte(e.Dice(ds(0x4263+k*2), ds(0x4264+k*2)))
			count++
			if k == 0 {
				e.wisdomSlots(c) // ovr24:01AF
				for s := 1; s <= 0x38; s++ {
					if rec := e.X.Bytes(0x31FA+s*16, 2); rec[0] == 0 && rec[1] == 1 {
						c.SpellBook[s-1] = 1
					}
				}
			} else if k == 5 {
				c.SpellBook[10], c.SpellBook[17], c.SpellBook[18], c.SpellBook[20] = 1, 1, 1, 1
			}
		}
		c.Coins[3] = uint16(int16(int(sum) * 10 / int(count)))
		c.HPRoll = byte(e.o17_4235(c, 0xFF))
		c.MaxHP = c.HPRoll
		adj := int(int8(e.o17_3F31(c)))
		if adj < 0 {
			if int(c.MaxHP) > -adj+int(count) {
				c.MaxHP = byte((int(c.MaxHP) + adj) / int(count))
			} else {
				c.MaxHP = 1
			}
		} else {
			c.MaxHP = byte((int(c.MaxHP) + adj) / int(count))
		}
		c.HP = c.MaxHP
		c.HPRoll = byte(int(c.HPRoll) / int(count))
		e.showCombatStats() // ovr20:0608
		if q := e.ext.quick; q != nil {
			if q.good(c) {
				break
			}
			continue
		}
		if e.YesNo(0xD, 0xA, 0xF, "Keep this character? ") == 'Y' {
			break
		}
	}
	e.CharacterSheet()
	undo()

	for q := e.ext.quick; q != nil; q = nil {
		c.SetName(q.name)
		c.PortHead, c.PortBody = q.head, q.body
	}
	for c.PName[0] == 0 {
		c.SetName(e.InputString("Character name: ", 0xD, 0xF)) // 0709:09E3
		if c.PName[0] != 0 {
			break
		}
	}
	undo = e.selectTemp(c)
	if e.ext.quick == nil {
		e.o17_33E7()
		e.o17_382D()
	}
	e.SaveCharacter(c, "") // ovr18:069E
	undo()
	e.freeCharacter(&c)
}

// DropCharacter = ovr17:1F37: asks twice, then the selected member leaves
// the party (a hired NPC bids farewell, anyone else is dumped).
func (e *Engine) DropCharacter() {
	if len(e.party) == 0 {
		return
	}
	c := e.selected()
	if e.YesNo(0xE, 0xA, 0xF, "Drop "+c.Name()+" forever? ") == 'Y' &&
		e.YesNo(0xE, 0xA, 0xF, "Are you sure? ") == 'Y' {
		if c.Okay == 0 {
			e.BottomMessage("You dump " + c.Name() + " out back.")
		} else {
			e.BottomMessage(c.Name() + " bids you farewell.")
		}
		e.RemoveSelected(false, true) // ovr17:3291
		return
	}
	e.BottomMessage(c.Name() + " breathes a sigh of relief.")
}
