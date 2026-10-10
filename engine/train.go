package engine

import "por/dax"

// ovr17 (second half): Modify, Train and Add on the party screen, the
// portrait and icon editors, and the hit point helpers they share.

// partyExp = es:[game+0x550]: the classes this training hall teaches
// (a mask that is ANDed with DS:5DE; the party menu only offers Train
// when it is set).
func (e *Engine) partyExp() int { return int(wordAt(e.game, 0x550)) }

// modifyDraw = ovr17:2033 (idx, highlight): the ability line idx of the
// selected character, or its hit points for idx 6.
func (e *Engine) modifyDraw(idx int, hi bool) {
	if idx < 6 {
		e.showStat(idx, hi) // ovr20:0901
	} else {
		e.displayHP(e.selected(), 4, 0x12, hi) // 00BA:0973
	}
}

// ModifyCharacter = ovr17:209E: change the abilities and hit points of a
// character that has no experience yet, within the race and class limits
// (DS:3DA3.., DS:3F81..). Left/right change the line, up/down move,
// Keep (K) accepts, Exit (E) puts the old values back.
func (e *Engine) ModifyCharacter() {
	c := e.selected()
	if int32(c.Exp) > 0 {
		e.DisplayMessageWait(0xE, c.Name()+" can't be modified.")
		return
	}
	e.CharacterSheet() // ovr20:004A
	var saved [6]byte
	copy(saved[:], c.Stats[:])
	savedStr00, savedHP := c.Str00, c.MaxHP
	tbl := func(off int) byte { return e.X.Bytes(off, 1)[0] }
	race, class, sex := int(c.Race), int(c.Class), int(c.Sex)
	minOff := [6]int{0x3DA3 + sex, 0x3DA9, 0x3DAB, 0x3DAD, 0x3DAF, 0x3DB1}
	maxOff := [6]int{0x3DA5 + sex, 0x3DAA, 0x3DAC, 0x3DAE, 0x3DB0, 0x3DB2}
	idx := 6
	e.modifyDraw(idx, false)
	idx = 0
	e.modifyDraw(idx, true)
	// hpToDraw redraws the hit point line, leaving the cursor on idx.
	hpToDraw := func() {
		c.HP = c.MaxHP
		save := idx
		idx = 6
		e.modifyDraw(idx, false)
		idx = save
	}
	// slots: when wisdom changes, a first level cleric slot is capped at 1.
	wisdom := func() {
		if c.SpellSlots[0] > 0 {
			c.SpellSlots[0] = 1
		}
		e.wisdomSlots(c) // ovr24:01AF
	}
	lower := func(i int) {
		c.Stats[i]--
		if i == 0 && c.Str00 > 0 {
			c.Str00--
			c.Stats[statStr]++
		} else if lo := tbl(minOff[i] + race*16); c.Stats[i] < lo || c.Stats[i] < tbl(0x3F81+class*6+i) {
			// Quirk: below the class minimum the race minimum is stored.
			c.Stats[i] = lo
		}
		switch i {
		case 2:
			wisdom()
		case 4:
			if m := e.maxHitPoints(c); m < c.MaxHP {
				c.MaxHP = m
			}
			hpToDraw()
		}
	}
	raise := func(i int) {
		c.Stats[i]++
		if hi := tbl(maxOff[i] + race*16); c.Stats[i] > hi {
			if i == 0 && c.Stats[statStr] > 0x12 && int8(c.Levels[clsFighter]) > 0 && tbl(0x3DA7+race*16+sex) > c.Str00 {
				c.Str00++ // exceptional strength for fighters
			}
			c.Stats[i] = hi
		}
		switch i {
		case 2:
			wisdom()
		case 4:
			if m := e.minHitPoints(c); m > c.MaxHP {
				c.MaxHP = m
			}
			hpToDraw()
		}
	}
	defer e.rawArrows("↑↓=score|← →=lower, raise|K, click=keep|Esc=cancel")()
	for {
		key, extra := e.DisplayInput("Modify character: ", "Keep Exit", 0xD, 0xA, 0xF, true, false)
		e.modifyDraw(idx, false)
		if !extra && (key == 'E' || key == 0) {
			copy(c.Stats[:], saved[:])
			c.Str00, c.MaxHP = savedStr00, savedHP
			c.HP = c.MaxHP
			e.ApplyItems(c) // 00BA:0BB8
			return
		}
		if extra {
			switch key {
			case 0x50, 0x0D:
				if idx++; idx > 6 {
					idx = 0
				}
			case 0x48:
				if idx--; idx < 0 {
					idx = 6
				}
			case 0x4B:
				if idx < 6 {
					lower(idx)
				} else {
					c.MaxHP--
					if m := e.minHitPoints(c); m > c.MaxHP {
						c.MaxHP = m
					}
					c.HP = c.MaxHP
				}
			case 0x4D:
				if idx < 6 {
					raise(idx)
				} else {
					c.MaxHP++
					if m := e.maxHitPoints(c); m < c.MaxHP {
						c.MaxHP = m
					}
					c.HP = c.MaxHP
				}
			}
		}
		e.ApplyItems(c)
		e.showCombatStats() // ovr20:0608
		e.modifyDraw(idx, true)
		if !extra && key == 'K' {
			c.Share = 1
			return
		}
	}
}

// o17_3F31 = ovr17:3F31 (char): hit point bonus per class from
// constitution (DS:408E), plus one or two for a class 2 character with
// constitution above 16 and 17. Returned as a signed byte.
func (e *Engine) o17_3F31(c *Character) int {
	var sum byte
	for i := 0; i <= 7; i++ {
		if int8(c.Levels[i]) <= 0 {
			continue
		}
		sum += e.X.Bytes(0x408E+int(c.Stats[statCon]), 1)[0]
		if c.Stats[statCon] > 0x10 && c.Class == 2 {
			sum++
		}
		if c.Stats[statCon] > 0x11 && c.Class == 2 {
			sum++
		}
	}
	return int(int8(sum))
}

// hitPointsFrom = the tail of ovr17:3FB3/408C: sum plus the constitution
// bonus, averaged over the classes (count 0 would be a divide error in
// the original; 0 here).
func (e *Engine) hitPointsFrom(c *Character, sum byte, count int) byte {
	if count == 0 {
		return 0
	}
	adj := e.o17_3F31(c)
	if adj < 0 {
		if int(sum) > count+(-adj) {
			return byte((int(sum) + adj) / count)
		}
		return 1
	}
	return byte((int(sum) + adj) / count)
}

// minHitPoints = ovr17:3FB3 (char): the least hit points the levels allow.
func (e *Engine) minHitPoints(c *Character) byte {
	var sum byte
	count := 0
	for i := 0; i <= 7; i++ {
		if lv := int8(c.Levels[i]); lv > 0 {
			sum += byte(lv)
			count++
			if i == 4 || i == 7 {
				sum++
			}
		}
	}
	return e.hitPointsFrom(c, sum, count)
}

// maxHitPoints = ovr17:408C (char): the most hit points the levels allow
// (a full hit die per level, the die of each class).
func (e *Engine) maxHitPoints(c *Character) byte {
	var sum byte
	count := 0
	for i := 0; i <= 7; i++ {
		lv := int(int8(c.Levels[i]))
		if lv <= 0 {
			continue
		}
		count++
		switch i {
		case 0, 1:
			sum += byte(lv << 3)
		case 2, 3:
			sum += byte(lv * 10)
		case 4:
			sum += byte((lv + 1) << 3)
		case 5:
			sum += byte(lv << 2)
		case 6:
			sum += byte(lv * 6)
		case 7:
			sum += byte((lv + 1) << 2)
		}
	}
	return e.hitPointsFrom(e.selected(), sum, count) // the original reads the bonus of the selected character, not c
}

// o17_4235 = ovr17:4235 (char, mask): hit dice rolled for the classes in
// mask (DS:5DE bits); a first level die is at least two thirds of the die.
func (e *Engine) o17_4235(c *Character, mask int) int {
	var total byte
	for i := 0; i <= 7; i++ {
		if int8(c.Levels[i]) <= 0 || e.X.Bytes(0x5DE+i, 1)[0]&byte(mask) == 0 {
			continue
		}
		n, sides := int(e.X.Bytes(0x8B6+i, 1)[0]), int(e.X.Bytes(0x8BE+i, 1)[0])
		r := e.Dice(n, sides) // ovr25:0DE3
		if c.Levels[i] == 1 && r < (sides<<1)/3 {
			r = (sides << 1) / 3
		}
		total += byte(r)
	}
	return int(total)
}

// TrainCharacter = ovr17:29AC: raise the classes whose experience
// (DS:409B table) allows it and which this hall teaches (game +0x550),
// for 1000 gp. Race level limits apply; a mage level gained offers a
// new spell for the book.
func (e *Engine) TrainCharacter() {
	c := e.selected()
	if c.Status != 0 && !e.debug {
		e.DisplayMessageWait(0xE, "we only train conscious people")
		return
	}
	if e.purse.gold < 1000 && !e.debug {
		e.DisplayMessageWait(0xE, "Training costs 1000 gp.")
		return
	}
	classMask := func(i int) byte { return e.X.Bytes(0x5DE+i, 1)[0] }
	var have, can byte
	var best int32
	race, str := c.Race, c.Stats[statStr]
	for i := 0; i <= 7; i++ {
		lv := c.Levels[i]
		if int8(lv) <= 0 {
			continue
		}
		have += classMask(i)
		// Level limits of the races.
		skip := false
		switch race {
		case 1:
			skip = i == 2 && (lv == 8 && str == 0x11 || lv == 7 && str < 0x11)
		case 2:
			skip = i == 2 && (lv == 7 || lv == 6 && str == 0x11 || lv == 5 && str < 0x11)
		case 3:
			skip = i == 2 && (lv == 6 || lv == 5 && str < 0x12)
		case 4:
			if i == 0 && lv == 5 {
				skip = true
			} else if i == 2 {
				skip = lv == 8 || lv == 7 && str == 0x11 || lv == 6 && str < 0x11
			}
		case 5:
			skip = i == 2 && (lv == 6 || lv == 5 && str == 0x11 || lv == 4 && str < 0x11)
		}
		if skip {
			continue
		}
		exp := int32(c.Exp)
		need := int32(e.X.Long(0x409B + 0x38*i + 4*int(int8(lv+1))))
		if need > exp && !e.debug || need <= 0 {
			continue
		}
		can += classMask(i)
		next := int32(e.X.Long(0x409B + 0x38*i + 4*int(int8(lv+2))))
		if next <= 0 || exp < next {
			continue
		}
		// Enough for two levels: the experience is held below the second.
		if next > best {
			best = next - 1
		}
	}
	if best > 0 {
		c.Exp = uint32(best)
	}
	hall := e.partyExp()
	if !e.debug {
		if byte(int(have)&hall) == 0 {
			e.DisplayMessageWait(0xE, "We don't train that class here")
			return
		}
		if byte(int(can)&hall) == 0 {
			e.DisplayMessageWait(0xE, "Not Enough Experience")
			return
		}
	}
	mask := can
	if !e.debug {
		mask = byte(int(can) & hall)
	}
	e.ClearCells(1, 1, 0x26, 0x16)
	y := 4
	e.displayName(c, 4, y, false) // 00BA:1816
	e.DisplayString(len(c.Name())+4, y, 0xA, " will become:")
	for i := 0; i <= 7; i++ {
		if int8(c.Levels[i]) > 0 && classMask(i)&mask > 0 {
			y++
			pre := "and a level "
			if y == 5 {
				pre = "    a level "
			}
			e.DisplayString(6, y, 0xA, pre+itoa(int(int8(c.Levels[i]))+1)+" "+e.X.PString(0xD38+i*0x1B))
		}
	}
	if e.YesNo(0xD, 0xA, 0xF, "Do you wish to train? ") != 'Y' {
		return
	}
	e.BottomMessage("Congratulations...")
	if !e.debug {
		e.purse.gold -= 1000
	}
	count := 0
	mage := int8(c.Levels[clsMage])
	for i := 0; i <= 7; i++ {
		if int8(c.Levels[i]) <= 0 {
			continue
		}
		count++
		if classMask(i)&mask > 0 {
			c.Levels[i]++
			if c.Drained > 0 {
				c.DrainedHP -= c.DrainedHP / c.Drained
				c.Drained--
			}
		}
	}
	e.RecomputeCharacter(c) // ovr24:0000
	if int8(c.Levels[clsMage]) > mage {
		idx := -1
		var spell byte
		any := true
		for {
			spell = e.SpellList(4, 4, &idx, &any) // ovr20:299C
			if spell > 0 || !any {
				break
			}
		}
		if spell > 0 { // (the original, with nothing to choose, set +0x32, the max HP, to 1)
			c.SpellBook[spell-1] = 1
		}
	}
	roll := e.o17_4235(c, int(mask))
	per := roll / count
	if per == 0 {
		per = 1
	}
	c.HPRoll += byte(per)
	per = (roll + e.o17_3F31(c)) / count
	if per < 1 {
		per = 1
	}
	damage := c.MaxHP - c.HP
	c.MaxHP += byte(per)
	c.HP = c.MaxHP - damage
}

// AddCharacter = ovr17:306D: pick saved characters from the roster list
// and add them to the party (at most 6 characters and 8 with hired NPCs).
// Chosen entries get a "* " in the list.
func (e *Engine) AddCharacter() {
	e.ClearCells(1, 1, 0x26, 0x16)
	names := e.BuildCharList() // ovr18:0010
	if len(names) == 0 {
		return
	}
	list := make([]*ListNode, len(names))
	for i, n := range names {
		list[i] = &ListNode{Text: n}
	}
	index := 0
	redraw := true
	var sel *ListNode
	count := 0 // uninitialised until the first add in the original
	for {
		// The show-exit argument is an uninitialised local; false here.
		key := e.ListMenu("Add a character: ", "Add Exit", 0xD, 0xA, 0xF, 1, 2, 0x26, 0x16, list, false, &redraw, &index, &sel)
		if (key == 0xD || key == 'A') && sel != nil && (sel.Text == "" || sel.Text[0] != '*') {
			c := &Character{}
			if e.LoadCharacter(sel.Text, c, false) { // ovr18:0AEC
				sel.Text = cut("* "+sel.Text, 0x28)
				count = 0
				if len(e.party) == 0 {
					putWord(e.game, gsPartyCount, 0)
					e.AddToParty(c) // ovr18:1446
					e.BuildCharIcon(c, true)
				} else {
					var dup *Character
					for _, m := range e.party {
						if m.Name() == c.Name() && m.RandomID == c.RandomID {
							dup = m
							break
						}
						if m.Control < 0x80 {
							count++
						}
					}
					npc := c.Control >= 0x80
					if dup == nil && (!npc && count < 6 || npc && wordAt(e.game, gsPartyCount) < 8) {
						e.AddToParty(c)
						e.BuildCharIcon(c, true)
						if !npc {
							count++
						}
					}
				}
			}
		}
		if key == 'E' || key == 0 || count > 5 || wordAt(e.game, gsPartyCount) > 7 {
			break
		}
	}
	for i, n := range list {
		names[i] = n.Text
	}
	e.WriteCharList(names) // ovr18:01B8
}

// o17_33E7 = ovr17:33E7: choose the portrait: Head (1..14) and Body
// (1..12) step through the pictures, Keep accepts. There is no way out
// without Keep (Esc just asks again).
func (e *Engine) o17_33E7() {
	c := e.selected()
	head, body := c.PortHead, c.PortBody
	for {
		e.SheetPortrait() // ovr29:04F9
		key, extra := e.DisplayInput("", "Head Body Keep", 0, 0xF, 0xA, false, false)
		if !extra {
			switch key {
			case 'H':
				if c.PortHead < 0xE {
					c.PortHead++
				} else {
					c.PortHead = 1
				}
			case 'B':
				if c.PortBody < 0xC {
					c.PortBody++
				} else {
					c.PortBody = 1
				}
			case 'K':
				head, body = c.PortHead, c.PortBody
			}
		}
		if key == 'K' {
			break
		}
	}
	c.PortHead, c.PortBody = head, body
	e.ClearRow24()
}

// iconPane = ovr17:34DF (x, y) nested in the icon editor: the two poses
// of the icon in slot 0xC at icon square (x,y), on a cleared 12 cell
// panel (on EGA over the slot 0x19 background icons).
func (e *Engine) iconPane(x, y int) {
	if e.Cfg.Graphics != 'E' && e.Cfg.Graphics != 'T' { // DS:4A1E == 0 (1 EGA, 2 Tandy)
		e.FillRect(x*3, y*0x18, 0xC, 0x18, 3)
	} else {
		e.FillRect(x*3, y*0x18, 0xC, 0x18, 0)
		e.drawIcon(x, y, 0, 0, 0x19)
		e.drawIcon(x+3, y, 0, 1, 0x19)
	}
	e.drawIcon(x, y, 0, 0, 0xC)
	e.drawIcon(x+3, y, 0, 1, 0xC)
	e.Flip() // 0464:1B52
}

// iconCopy = ovr17:35BE (src, dst, recolour) nested in the icon editor:
// copy both poses of icon slot src into slot dst (the original also
// copies the mask planes, which the port keeps in the image); with
// recolour the selected character's six colour pairs (+0xC1..) replace
// the DS:3CF5 colours in the copy.
func (e *Engine) iconCopy(src, dst byte, recolour bool) {
	for p := 0; p < 2; p++ {
		if im := e.icons[src][p]; im != nil {
			e.icons[dst][p] = im.Clone()
		}
	}
	if !recolour {
		return
	}
	c := e.selected()
	var to [16]byte
	for i := range to {
		to[i] = byte(i)
	}
	pos := e.X.Bytes(0x3CF5, 7)
	for i := 1; i <= 6; i++ {
		v := c.IconColour[i-1]
		to[pos[i]] = v & 0xF
		to[pos[i]+8] = v >> 4
	}
	for p := 0; p < 2; p++ {
		recolourIcon(e.icons[dst][p], to)
	}
}

// recolourIcon = 0464:1E5C (identity, to, 0, 0, icon): colour i becomes
// to[i] for i = 0..15 in turn, so a pixel already moved to a later colour
// moves again (3 -> 15 -> to[15]). Mask pixels are left alone.
func recolourIcon(im *dax.Image, to [16]byte) {
	if im == nil {
		return
	}
	for i := 0; i < 16; i++ {
		if to[i] == byte(i) {
			continue
		}
		for k, v := range im.Data {
			if int(v) == i && i != im.Mask {
				im.Data[k] = to[i]
			}
		}
	}
}

// o17_382D = ovr17:382D: the combat icon editor (called from the
// character creation and the camp). States (menus DS:7BF + 0x29*state):
// 1 Parts/color-1/color-2/Size/Exit, 2 Head/Weapon, 3 the part to colour,
// 4 Size, 5 the Next/Prev/Keep/Exit stepping for the choice made before.
// The old icon is shown above the new one; Is this icon ok? ends it.
func (e *Engine) o17_382D() {
	c := e.selected()
	e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0, "")
	e.combatViewLeave() // ovr32:00E3
	menu3 := e.X.PString(0x83A)
	var state, parent, part int // parent and part are uninitialised until set
	var sub, key byte
	var face bool
	for {
		e.BuildCharIcon(c, false) // ovr18:0482
		state = 1
		var colours [6]byte
		copy(colours[:], c.IconColour[:])
		order := c.Order
		c.Order = 0xC // the working copy is built into slot 0xC
		e.BuildCharIcon(c, false)
		c.Order = order
		head, weapon, size := c.IconHead, c.IconBody, c.IconSize
		e.iconCopy(order, 0xC, true)
		e.iconPane(1, 2)
		e.DisplayString(8, 6, 0xF, "old")
		e.DisplayString(3, 0xA, 0xF, "ready   action")
		e.DisplayString(8, 0xC, 0xF, "new")
		e.DisplayString(3, 0x10, 0xF, "ready   action")
		for {
			e.iconPane(1, 4)
			sizeStr := ""
			if state == 4 {
				sizeStr = "Large"
				if c.IconSize == 2 {
					sizeStr = "Small" // as in the original
				}
			}
			m := e.X.PString(0x7BF + state*0x29)
			if state == 3 {
				m = menu3
			}
			k, extra := e.DisplayInput("", sizeStr+m, 0xD, 0xA, 0xF, false, false)
			key = k
			if !extra {
				done := key == 0 || key == 'E'
				switch state {
				case 1:
					parent = 1
					switch key {
					case 'P':
						state = 2
					case '1', '2':
						state, face = 3, key == '2'
						label := "Hair"
						if face {
							label = "Face"
						}
						// Delete(s, 13, 4); Insert(label, s, 13) on DS:83A.
						s := menu3[:12] + menu3[16:]
						menu3 = cut(s[:12]+label+s[12:], 0x28)
					case 'S':
						state = 4
					}
				case 2:
					parent = 2
					if done {
						state = 1
					} else {
						sub, state = key, 5
					}
				case 3:
					parent = 3
					part = 1
					switch key {
					case 'W':
						part = 6
					case 'B':
						part = 1
					case 'H', 'F':
						part = 4
					case 'S':
						part = 5
					case 'A':
						part = 2
					case 'L':
						part = 3
					}
					if done {
						state = 1
					} else {
						state = 5
					}
				case 4:
					switch {
					case key == 'L':
						c.IconSize = 2
						e.BuildCharIcon(c, false)
					case key == 'S':
						c.IconSize = 1
						e.BuildCharIcon(c, false)
					case key == 'K':
						size, state, key = c.IconSize, 1, ' '
					case done:
						c.IconSize = size
						state, key = 1, ' '
					}
					e.BuildCharIcon(c, false)
				case 5:
					switch parent {
					case 2:
						// Step head (0..13) or weapon (0..31) pictures.
						pic, saved, top := &c.IconHead, &head, byte(0xD)
						if sub == 'W' {
							pic, saved, top = &c.IconBody, &weapon, 0x1F
						}
						if sub == 'H' || sub == 'W' {
							switch {
							case key == 'P':
								if *pic > 0 {
									*pic--
								} else {
									*pic = top
								}
							case key == 'N':
								if *pic < top {
									*pic++
								} else {
									*pic = 0
								}
							case key == 'K':
								*saved, state, key = *pic, parent, ' '
							case done:
								*pic = *saved
								state, key = parent, ' '
							}
							e.BuildCharIcon(c, false)
						}
					case 3:
						lo, hi := c.IconColour[part-1]&0xF, c.IconColour[part-1]>>4
						switch {
						case key == 'N', key == 'P':
							d := byte(1)
							if key == 'P' {
								d = 0xF // (n-1) and 15
							}
							if face {
								hi = (hi + d) & 0xF
							} else {
								lo = (lo + d) & 0xF
							}
							c.IconColour[part-1] = hi<<4 + lo
						case key == 'K':
							copy(colours[:], c.IconColour[:])
							state, key = parent, ' '
						case done:
							copy(c.IconColour[:], colours[:])
							state, key = parent, ' '
						}
					}
				}
			}
			e.iconCopy(c.Order, 0xC, true)
			if parent == 1 && (key == 0 || key == 'E') {
				break
			}
		}
		c.IconHead, c.IconBody, c.IconSize = head, weapon, size
		copy(c.IconColour[:], colours[:])
		e.iconCopy(c.Order, 0xC, true)
		e.iconCopy(0xC, c.Order, false)
		e.ClearRow24()
		e.freeIcon(0xC) // ovr33:014E
		if e.viewMode > 1 {
			e.LoadIconPic("ICON", 0, 0xB)
			e.LoadIconPic("ICON", 1, 0xC)
		}
		if e.YesNo(0xD, 0xA, 0xF, "Is this icon ok? ") == 'Y' {
			e.Music(3) // Amiga 0x2F33E
			break
		}
	}
	e.combatViewEnter() // ovr32:0105
}
