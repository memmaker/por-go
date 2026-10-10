package engine

import "strings"

// ovr20: the View screen (character sheet, items, money, spell lists).

const (
	chPlayerFlag = 0x84 // > 0x7F: hired NPC
	chSex        = 0x9E
	chAlignment  = 0xA0
	chClass      = 0x2F
	chAge        = 0x30 // word
	chExp        = 0xAC // long
	chSpells     = 0x17 // 0x15 spell ids, +0x80 = still to be memorised
	chStudy      = 0x2C // hours of study the pending spells need (0/4/6)
)

// ClearScreen = 0709:0CAB.
func (e *Engine) ClearScreen() {
	if e.Trace != nil {
		e.Trace("0709:0CAB", "ClearScreen")
	}
	e.fillCells(0, 0, 0x27, 0x18)
}

// viewChar = DS:4716, the character on the sheet.
func (e *Engine) viewChar() *Character { return e.view.char }

type viewState struct {
	char  *Character // DS:4716
	start *Character // DS:471A
}

// CharacterSheet = ovr20:004A.
func (e *Engine) CharacterSheet() {
	c := e.selected()
	e.view.char = c
	e.ClearScreen()
	e.DrawBorder(1, 1, 0x26, 0x16, 0)
	e.displayName(c, 1, 1, false)
	if c.Raw[chPlayerFlag] > 0x7F {
		e.DisplayString(len(c.Name())+3, 1, 0xA, "(NPC)")
	}
	x := 1
	s := e.X.PString(0xFEC + 7*int(int8(c.Raw[chSex])))
	e.DisplayString(x, 3, 0xF, s)
	x += len(s) + 1
	s = e.X.PString(0xF03 + 0xA*int(int8(c.Raw[chRace])))
	e.DisplayString(x, 3, 0xF, s)
	x += len(s) + 1
	e.DisplayString(x, 3, 0xF, "Age "+itoa(int(wordAt(c.Raw[:], chAge))))
	e.DisplayString(1, 4, 0xF, e.X.PString(0xF53+0x11*int(int8(c.Raw[chAlignment]))))
	e.DisplayString(1, 5, 0xF, e.X.PString(0xD38+0x1B*int(int8(c.Raw[chClass]))))
	for i, l := range []string{"STR ", "INT ", "WIS ", "DEX ", "CON ", "CHA "} {
		e.DisplayString(1, i+7, 0xA, l)
		e.showStat(i, false)
	}
	e.showMoney()
	e.DisplayString(1, 0xF, 0xF, "Level")
	s = ""
	for i := 0; i <= 7; i++ {
		if l := int8(c.Raw[chLevel+i]); l > 0 {
			if s != "" {
				s += "/"
			}
			s += itoa(int(l))
		}
	}
	e.DisplayString(7, 0xF, 0xF, s)
	exp := uint32(wordAt(c.Raw[:], chExp)) | uint32(wordAt(c.Raw[:], chExp+2))<<16
	e.DisplayString(0x11, 0xF, 0xF, "Exp "+itoa(int(int32(exp))))
	e.showCombatStats()
	y := 0x14
	if w := c.Slots[0]; w != nil {
		e.DisplayString(1, y, 0xF, "Weapon")
		e.ItemName(c, w, 8, y, false, true)
	}
	y++
	if a := c.Slots[2]; a != nil {
		e.DisplayString(2, y, 0xF, "Armor")
		e.ItemName(c, a, 8, y, false, true)
	}
	y++
	e.DisplayString(1, y, 0xF, "Status")
	e.DisplayString(8, y, 0xA, e.statusName(c.Raw[chStatus]))
	e.sheetEffects(c)
	if e.flag49F5 != 0 && e.symDAX == "8X8D3" { // DS:5376 == 3
		e.SheetPortrait()
	}
}

// Extra: the sheet lists the spell effects on the character (the camp
// menu's Display, which is not offered) on the free row under HP. When they
// do not all fit, the View menu offers Magic with the whole list.
const effectsRow = 0x13

func (e *Engine) effectsLine(c *Character) (string, bool) {
	names := e.affectList(c)
	if len(names) == 0 {
		return "", true
	}
	s := strings.Join(names, ", ")
	if len(s) <= 0x26-9 {
		return s, true
	}
	n := 0
	for len(strings.Join(names[:n+1], ", "))+len(", +9 more") <= 0x26-9 {
		n++
	}
	if n == 0 {
		return itoa(len(names)) + " spell effects", false
	}
	return strings.Join(names[:n], ", ") + ", +" + itoa(len(names)-n) + " more", false
}

func (e *Engine) effectsFit(c *Character) bool { _, ok := e.effectsLine(c); return ok }

func (e *Engine) sheetEffects(c *Character) {
	s, _ := e.effectsLine(c)
	if s == "" {
		return
	}
	e.DisplayString(1, effectsRow, 0xF, "Effects")
	e.DisplayString(9, effectsRow, 0xA, s)
}

// showCombatStats = ovr20:0608: AC, HP, THAC0, damage, weight, movement.
func (e *Engine) showCombatStats() {
	c := e.selected()
	e.ApplyItems(c)
	y := 0x11
	e.DisplayString(1, y, 0xF, "AC    ")
	e.displayAC(c, 4, y)
	e.DisplayString(1, y+1, 0xF, "HP    ")
	e.displayHP(c, 4, y+1, false)
	x := 8
	e.DisplayString(x+1, y, 0xF, "THAC0   ")
	e.DisplayString(x+7, y, 0xA, itoa(int(byte(0x3C-c.Raw[chTHAC0]))))
	d := itoa(int(c.Raw[0x115])) + "d" + itoa(int(c.Raw[0x117]))
	if b := int8(c.Raw[0x119]); b > 0 {
		d += "+" + itoa(int(b))
	} else if b < 0 {
		d += "-" + itoa(-int(b)) // abs in 16 bits: -128 shows 128
	}
	d = cut(d, 0x28)
	e.DisplayString(x, y+1, 0xF, "Damage        ")
	e.DisplayString(x+7, y+1, 0xA, d)
	x = 0x16
	e.DisplayString(x, y, 0xF, "Encumbrance      ")
	e.DisplayString(x+0xC, y, 0xA, itoa(int(wordAt(c.Raw[:], chWeight))))
	m := c.Raw[chMove]
	if e.FindAffect(c, 0x2A) != nil {
		m <<= 1
	}
	if e.FindAffect(c, 0x27) != nil {
		m >>= 1
	}
	e.DisplayString(x+3, y+1, 0xF, "Movement   ")
	e.DisplayString(x+0xC, y+1, 0xA, itoa(int(m)))
}

// showStat = ovr20:0901 (stat, highlight): one ability score, with the
// exceptional strength in brackets.
func (e *Engine) showStat(stat int, hi bool) {
	col := 0xA
	if hi {
		col = 0xD
	}
	c := e.selected()
	x := 5
	e.ClearCells(5, stat+7, 0xB, stat+7)
	v := c.Raw[chStr+stat]
	if v < 10 {
		x++
	}
	e.DisplayString(x, stat+7, col, itoa(int(v)))
	if stat == 0 && v == 0x12 && c.Raw[chStr00] > 0 {
		s := itoa(int(c.Raw[chStr00]))
		if c.Raw[chStr00] < 10 {
			s = "0" + s
		}
		if c.Raw[chStr00] == 100 {
			s = "00"
		}
		e.DisplayString(7, 7, col, "("+s+")")
	}
}

// ViewCharacter = ovr20:0AB3 (var quit): the View command. Returns quit,
// which ItemsMenu sets when an item was used in combat.
func (e *Engine) ViewCharacter() bool {
	if e.beginFlag == 5 {
		e.combatViewEnter() // ovr31 (seg 03E0) +2F
	}
	key := byte(' ')
	quit := false
	e.view.start = e.selected()
	direct := e.ext.toItems // F1-F6: start on the inventory (extras.go)
	e.ext.toItems = false
	if direct && len(e.selected().Items) > 0 {
		var leave bool
		if quit, leave = e.inventory(direct); leave {
			key = 'E'
		}
	}
	if key != 'E' && !quit {
		e.CharacterSheet()
	}
	for key != 0 && key != 'E' && !quit {
		c := e.viewChar()
		spells := false
		for i := 0; i <= 0x14; i++ {
			if c.Raw[chSpells+i] > 0 {
				spells = true
			}
		}
		menu := ""
		if len(c.Items) > 0 {
			menu += "Items "
		}
		if spells {
			menu += "Spells "
		}
		if !e.effectsFit(c) {
			menu += "Magic " // extra: all of the spell effects
		}
		menu = cut(menu+"Exit", 0x28)
		e.ext.viewing = true // F1..F6 switch members (extras.go)
		k, extra := e.DisplayInput("View:", menu, 0xD, 0xA, 0xF, false, false)
		e.ext.viewing = false
		key = k
		_ = extra
		switch key {
		case 'I':
			var leave bool
			if quit, leave = e.inventory(direct); leave {
				key = 'E'
			}
		case 'S':
			idx := -1
			e.SpellList(0, 0, &idx, &spells)
		case 'M':
			e.displayAffects("Magic", []*Character{c})
		}
		if !quit && (key == 'I' || key == 'S' || key == 'M') {
			e.CharacterSheet()
		}
	}
	if e.beginFlag == 5 {
		e.combatViewLeave() // ovr31 (seg 03E0) +2A
	}
	e.DrawMainScreen()
	return quit
}

// inventory runs ItemsMenu for ViewCharacter (extra:
// leave reports that the View page should close too: F1-F6 switched to
// another member, or Esc in an inventory F1-F6 opened, unless P came back
// to the View page).
func (e *Engine) inventory(direct bool) (quit, leave bool) {
	e.ext.toSheet, e.ext.switchTo = false, false
	quit = e.ItemsMenu()
	leave = !quit && (e.ext.switchTo || direct && !e.ext.toSheet)
	e.ext.toSheet, e.ext.switchTo = false, false
	return quit, leave
}

// canLoseItem = ovr20:0D70 (item): readied items must be unreadied first;
// a scroll someone is about to scribe from asks before it goes.
func (e *Engine) canLoseItem(it *Item) bool {
	ok := false
	switch {
	case it.Raw[itReadied] != 0:
		e.BottomMessage("Must be unreadied")
	case !e.isScroll(it):
		ok = true
	case it.Raw[0x3C] > 0x7F || it.Raw[0x3D] > 0x7F || it.Raw[0x3E] > 0x7F:
		c := e.selected()
		e.displayName(c, 1, 0x15, false)
		e.msgCol, e.msgRow = len(c.Name())+2, 0x15
		e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, false, " was going to scribe from that scroll")
		ok = e.YesNo(0xD, 0xA, 0xF, "is it Okay to lose it? ") == 'Y'
	default:
		ok = true
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
	return ok
}

// SheetPortrait = ovr29:04F9: the character's head and body in a box at
// the top right of the sheet.
func (e *Engine) SheetPortrait() {
	c := e.selected()
	if c.Raw[0xBB] == 0 || c.Raw[0xBC] == 0 {
		return
	}
	e.DrawBoxTitled(0x1C, 1, 0x26, 0xB, 0, 0, "")
	e.LoadPortrait(int(e.X.Bytes(0x286F+int(c.Raw[0xBB]), 1)[0]), int(e.X.Bytes(0x287D+int(c.Raw[0xBC]), 1)[0]))
	e.DrawPortrait(0x1C, 1, true)
}

// ItemsMenu = ovr20:0EF8: the character's items with Ready, Use, Trade,
// Drop, Halve, Join, Sell and Id. Returns true when the View should end
// (an item was used in combat).
func (e *Engine) ItemsMenu() bool {
	c := e.selected()
	key := byte(' ')
	var it *Item
	index := 0
	redraw, full := true, true
	quit := false
	for key != 0 && key != 'E' && !quit && c.Raw[chItemCount] > 0 {
		count := c.Raw[chItemCount]
		if len(c.Items) == 0 {
			if c.Raw[chItemCount] != count {
				redraw = true
			}
			continue
		}
		hired := c.Raw[chPlayerFlag] >= 0x80 && c.Raw[chOkay] != 0 && c.Raw[chStatus] != 1
		menu := "Ready"
		if c.Raw[chOkay] != 0 && wordAt(e.area, 0x1CA) == 0 &&
			(e.beginFlag == 2 || e.beginFlag == 3 || e.beginFlag == 4 ||
				e.beginFlag == 5 && c.Combat != nil && c.Combat.Raw[2] != 0) {
			menu += " Use"
		}
		if !hired && e.beginFlag != 5 {
			menu += " Trade"
		}
		menu += " Drop"
		if c.Raw[chItemCount] < 0x10 {
			menu += " Halve"
		}
		menu += " Join"
		if !hired && e.beginFlag == 1 {
			menu += " Sell"
		}
		if e.beginFlag == 1 {
			menu += " Id"
		}
		menu = cut(menu, 0x28)
		list := make([]*ListNode, len(c.Items))
		for i, x := range c.Items {
			e.ItemName(c, x, 0, 0, true, false)
			list[i] = &ListNode{Text: x.Name()}
		}
		if full || e.itemsRedraw {
			e.ClearScreen()
			e.DrawBorder(1, 1, 0x26, 0x16, 0)
			e.displayName(c, 1, 1, true)
			e.DisplayString(len(c.Name())+4, 1, 0xA, "Items")
			e.DrawBoxTitled(1, 3, 0x26, 0x16, 0, 0, "")
			e.DisplayString(1, 3, 0xF, "Ready Item")
			redraw, full, e.itemsRedraw = true, false, false
		}
		var sel *ListNode
		e.ext.items = true // P and F1-F6 (extras.go)
		key = e.ListMenu("", menu, 0xD, 0xA, 0xF, 1, 5, 0x26, 0x16, list, true, &redraw, &index, &sel)
		e.ext.items = false
		it = nil
		for i, n := range list {
			if n == sel {
				it = c.Items[i]
			}
		}
		if it != nil {
			switch key {
			case 'R':
				e.readyItem(it)
			case 'U':
				if it.Raw[itReadied] == 0 {
					e.BottomMessage("Must be Readied")
					key = ' '
				} else if e.isScroll(it) || it.Raw[0x3D] > 0 && it.Raw[0x3E] < 0x80 {
					quit = e.useItem(it)
					if e.beginFlag != 5 {
						quit = false
					}
					if !quit {
						full = true
					}
				}
			case 'T':
				if e.canLoseItem(it) {
					e.tradeItem(it)
				} else {
					key = ' '
				}
				full = true
			case 'D':
				if e.canLoseItem(it) {
					e.ItemName(c, it, 0, 0, false, false)
					e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "Your "+it.Name()+"will be gone forever")
					if e.YesNo(0xD, 0xA, 0xF, "Drop It? ") == 'Y' {
						e.RemoveItem(e.selected(), it)
						redraw = true
					}
					e.ClearCells(1, 0x15, 0x26, 0x16)
				} else {
					key = ' '
				}
			case 'H':
				e.halveItem(it)
			case 'J':
				e.joinItems(it)
			case 'S':
				if e.canLoseItem(it) {
					e.sellItem(it)
				} else {
					key = ' '
				}
			case 'I':
				e.identifyItem(it, &redraw)
			}
		}
		e.ApplyItems(c) // 146F: also reached with nothing selected
		if c.Raw[chItemCount] != count {
			redraw = true
		}
	}
	return quit
}

// readyItem = ovr20:14D0 (item): ready or unready, with the reasons it
// cannot be readied and the item's own effect (+0x3E > 0x7F).
func (e *Engine) readyItem(it *Item) {
	effect := it.Raw[0x3E] > 0x7F
	c := e.selected()
	if it.Raw[itReadied] != 0 {
		if it.Raw[itMagic] != 0 {
			e.BottomMessage("It's Cursed")
			return
		}
		it.Raw[itReadied] = 0
		if effect {
			e.itemAffect(it, c, 1)
		}
		return
	}
	why := 0
	typ := it.Raw[itType]
	if int(e.baseItem(typ, 1))+int(c.Raw[chHands]) > 2 {
		why = 3
	}
	slot := int(e.baseItem(typ, 0))
	switch {
	case slot <= 8:
		if c.Slots[slot] != nil {
			why = 2
		}
	case slot == 9:
		if c.Slots[slotRing2] != nil {
			why = 2
		}
	}
	if typ == 0x49 && c.Slots[slotArrows] != nil {
		why, slot = 2, slotArrows
	}
	if typ == 0x1C && c.Slots[slotBolts] != nil {
		why, slot = 2, slotBolts
	}
	if e.baseItem(typ, 0xD)&c.Raw[0xB0] == 0 {
		why = 1
	}
	switch why {
	case 0:
		it.Raw[itReadied] = 1
		if effect {
			e.itemAffect(it, c, 0)
		}
	case 1:
		e.BottomMessage("Wrong Class")
	case 2:
		// slot 9 (ring 2 taken) names Slots[9], which may be empty: the
		// original then reads through a nil pointer.
		name := ""
		if o := c.Slots[slot]; o != nil {
			e.ItemName(c, o, 0, 0, false, false)
			name = o.Name()
		}
		e.BottomMessage("already using " + name)
	case 3:
		if !(e.beginFlag == 5 && c.Raw[0x10F] != 0) {
			e.BottomMessage("Your hands are full!")
		}
	}
}

// itemAffect calls ovr25:0000 with the item standing in for the affect
// record, as the original does.
func (e *Engine) itemAffect(it *Item, c *Character, flag byte) {
	a := &Affect{Item: it}
	copy(a.Raw[:], it.Raw[:])
	e.callAffect(it.Raw[0x3E], c, a, flag)
}

// tradeItem = ovr20:1734 (item).
func (e *Engine) tradeItem(it *Item) {
	to := e.view.start
	e.DrawMainScreen()
	to = e.SelectCharacter("Trade with Whom?", true, to)
	if to == nil {
		return
	}
	e.view.start = to
	if e.overloaded(it, to) {
		e.BottomMessage("Overloaded")
		return
	}
	e.AddItemCopy(to, it)
	e.RemoveItem(e.selected(), it)
	e.ApplyItems(to)
}

// halveItem = ovr20:17ED (item): half the stack becomes a new unreadied
// item right after it.
func (e *Engine) halveItem(it *Item) {
	half := it.Raw[itCount] >> 1
	if half == 0 {
		e.BottomMessage("Can't halve that")
		return
	}
	n := *it
	n.Raw[itCount] = half
	n.Raw[itReadied] = 0
	it.Raw[itCount] -= half
	c := e.selected()
	for i, x := range c.Items {
		if x == it {
			c.Items = append(c.Items[:i+1], append([]*Item{&n}, c.Items[i+1:]...)...)
			return
		}
	}
}

// joinItems = ovr20:189D (item): every matching stack joins it, up to 255
// a stack. (The original compares +0x3C..+0x3E of each item with itself.)
func (e *Engine) joinItems(it *Item) {
	total := int(it.Raw[itCount])
	c := e.selected()
	for _, p := range append([]*Item(nil), c.Items...) {
		if p == it || p.Raw[itCount] == 0 {
			continue
		}
		same := true
		for _, k := range []int{0x2F, 0x30, 0x31, itType, itPlus, itPlus2, itMagic} {
			if p.Raw[k] != it.Raw[k] {
				same = false
			}
		}
		if !same || wordAt(p.Raw[:], 0x37) != wordAt(it.Raw[:], 0x37) || p.Raw[0x3C] >= 2 {
			continue
		}
		if int(p.Raw[itCount])+total <= 0xFF {
			total += int(p.Raw[itCount])
			e.RemoveItem(c, p)
		} else {
			it.Raw[itCount] = 0xFF
			p.Raw[itCount] -= byte(0xFF - total)
			total = int(p.Raw[itCount])
			it = p
		}
	}
	it.Raw[itCount] = byte(total)
}

// useItem = ovr20:1A8A (item): cast the scroll's spell or the item's own
// (+0x3D, charges +0x3C). Returns the spell's "turn used" result.
func (e *Engine) useItem(it *Item) bool {
	idx := -1
	e.usingItem = false
	var spell byte
	quit := false
	if e.isScroll(it) {
		e.usedScroll = it
		var flag bool
		spell = e.SpellList(2, 1, &idx, &flag)
	} else if it.Raw[0x3D] > 0 && it.Raw[0x3E] < 0x80 {
		e.usingItem = true
		spell = it.Raw[0x3D]
	}
	if spell == 0 {
		return false
	}
	c := e.selected()
	if e.beginFlag == 5 && c.Raw[0x10F] == 0 {
		e.combatUseHook() // 00BA:2B88
	}
	if e.usingItem {
		if spell > 0x38 {
			spell -= 0x17
		}
		e.CharMessage(c, "uses an item", 0xA, false)
		if e.beginFlag == 5 {
			e.DisplayString(0, 0x17, 0xA, "Item:")
			e.ItemName(c, it, 5, 0x17, false, true)
		} else {
			e.ItemName(c, it, 1, 0x16, false, true)
		}
		e.MsgDelay()
		e.ClearMessageArea()
	}
	e.usingItem = true
	e.CastSpell(spell, c.Raw[0x10F], 0, &quit)
	e.usingItem = false
	if e.beginFlag == 5 && e.spellRec(spell, 0xB) != 0 {
		quit = e.clearCombatInfo(c)
	}
	if quit {
		if e.isScroll(it) {
			e.scrollUsed(c, it, spell)
		} else if it.Raw[0x3C] > 0 {
			if it.Raw[itCount] > 1 {
				it.Raw[itCount]--
			} else {
				it.Raw[0x3C]--
				if it.Raw[0x3C] == 0 {
					e.RemoveItem(c, it)
				}
			}
		}
	}
	return quit
}

// spellRec = DS:31FA + 16*spell.
func (e *Engine) spellRec(spell byte, k int) byte {
	return e.X.Bytes(0x31FA+int(spell)*16+k, 1)[0]
}

// sellItem = ovr20:1CEE (item): half the value (a twentieth per piece
// of a stack), paid into the party's gold.
func (e *Engine) sellItem(it *Item) {
	v := wordAt(it.Raw[:], 0x3A) >> 1
	if it.Raw[itCount] > 1 {
		// (the arrows/quarrels branch of the original can never be taken)
		v = uint16(uint32(uint16(uint32(it.Raw[itCount])*uint32(v))) / 0x14)
	}
	e.ItemName(e.viewChar(), it, 0, 0, false, false)
	e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true,
		"I'll give you "+itoa(int(v))+" gold pieces for your "+it.Name())
	if e.YesNo(0xD, 0xA, 0xF, "Is It a Deal? ") == 'Y' {
		e.BottomMessage("Sold!")
		e.RemoveItem(e.selected(), it)
		e.purse.gold += int(v)
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
}

// identifyItem = ovr20:1F52 (item, redraw): 200 gp shows the hidden words.
func (e *Engine) identifyItem(it *Item, redraw *bool) {
	e.ItemName(e.viewChar(), it, 0, 0, false, false)
	e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "For 200 gold pieces I'll identify your "+it.Name())
	if e.YesNo(0xD, 0xA, 0xF, "Is It a Deal? ") == 'Y' && e.pay(200) {
		if it.Raw[itHidden] == 0 {
			e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "I can't tell anything new about your "+it.Name())
		} else {
			it.Raw[itHidden] = 0
			e.ItemName(e.viewChar(), it, 0, 0, false, false)
			e.DisplayTextBlock(1, 0x15, 0x26, 0x16, 0xE, true, "It looks like some sort of "+it.Name())
			*redraw = true
		}
		e.MsgDelay()
	}
	e.ClearCells(1, 0x15, 0x26, 0x16)
}

// overloaded = ovr20:275B (item, char): no room for one more item.
func (e *Engine) overloaded(it *Item, c *Character) bool {
	e.ApplyItems(c)
	over := c.Raw[chItemCount] > 0xF
	w := wordAt(it.Raw[:], 0x37)
	if it.Raw[itCount] > 0 {
		w *= uint16(it.Raw[itCount])
	}
	if int32(wordAt(c.Raw[:], chWeight)+w) > int32(strCarry(c))+1500 {
		over = true
	}
	return over
}

// SpellList = ovr20:299C (kind, mode, index, any): the list of spells of
// one kind (0 memory, 1 book, 2 scroll, 3 scrolls, 4 choose, 5 to be
// memorised, 6 to be scribed); returns the chosen spell or 0.
func (e *Engine) SpellList(kind, mode int, index *int, any *bool) byte {
	titles := []string{"in Memory", "in Spell Book", "on Scroll", "on Scrolls", "to choose from", "to be memorized", "to be scribed"}
	title := ""
	if kind >= 0 && kind < len(titles) {
		title = titles[kind]
	}
	*any = e.buildSpellList(kind)
	if !*any {
		return 0
	}
	if *index < 0 {
		e.DrawBoxTitled(1, 1, 0x26, 1, 0, 0, "")
		c := e.selected()
		e.displayName(c, 1, 1, true)
		e.DisplayString(len(c.Name())+4, 1, 0xA, "Spells "+title)
	}
	return e.spellMenu(mode, index)
}
