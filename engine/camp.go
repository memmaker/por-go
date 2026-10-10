package engine

import "strconv"

// Encamp, ported from ovr16 (camp menu, magic: cast / memorize / scribe /
// display, alter: order / drop / speed / pics) and the rest of ovr21
// (resting, healing, study).

// campState holds the ovr16/ovr21 globals.
type campState struct {
	rest     [7]int     // DS:6DC2: rest time left, clock units (DS:6DC4 mins, 6DC6 tens, 6DC8 hours, 6DCA days)
	heal     int        // DS:6DD8: five-minute steps toward the daily healing
	study    [0x49]byte // DS:6DCF: five-minute steps until member i's next spell is learned
	interval int        // DS:49FB: steps since the last encounter roll
}

func (e *Engine) spellLevel(sp byte) byte { return e.X.Bytes(0x31FB+16*int(sp&0x7F), 1)[0] }
func (e *Engine) spellClass(sp byte) byte { return e.X.Bytes(0x31FA+16*int(sp&0x7F), 1)[0] }

// studyTime = ovr16:0000 (char): minutes of study for the spells waiting
// to be memorised (+0x80) or scribed; +0x2C gets the base hours (4, or 6
// for spells above 2nd level).
func (e *Engine) studyTime(c *Character) int {
	maxMem, maxScr, sumMem, sumScr := byte(0), byte(0), 0, 0
	for i := 0; i <= 0x14; i++ {
		if s := c.Memorized[i]; s > 0x7F {
			l := e.spellLevel(s)
			if l > maxMem {
				maxMem = l
			}
			sumMem += int(l)
		}
	}
	for _, it := range c.Items {
		if !e.isScroll(it) {
			continue
		}
		for k := 1; k <= 3; k++ {
			if s := it.Spells[k-1]; s > 0x7F {
				l := e.spellLevel(s)
				if l > maxScr {
					maxScr = l
				}
				sumScr += int(l)
			}
		}
	}
	base := 0
	if sumMem > 0 || sumScr > 0 {
		base = 4
	}
	if maxMem > 2 || maxScr > 2 {
		base = 6
	}
	c.Study = byte(base)
	return base*60 + sumMem*15 + sumScr*15
}

// clearMemorize = ovr16:0159 (char): forget the spells still to be memorised.
func clearMemorize(c *Character) {
	for i := 0; i <= 0x14; i++ {
		if c.Memorized[i] > 0x7F {
			c.Memorized[i] = 0
		}
	}
	c.Study = 0
}

// clearScribe = ovr16:019C (char): scroll spells are no longer being scribed.
func (e *Engine) clearScribe(c *Character) {
	for _, it := range c.Items {
		if e.isScroll(it) {
			for k := 1; k <= 3; k++ {
				it.Spells[k-1] &= 0x7F
			}
		}
	}
}

// clearStudy = ovr16:0214: nobody studies any more.
func (e *Engine) clearStudy() {
	for _, c := range e.party {
		clearMemorize(c)
		e.clearScribe(c)
	}
}

// slotsLeft = ovr16:025C (level, class): spells of the level the selected
// character may still memorise (+0xB1 + 3*class + level holds the total).
func (e *Engine) slotsLeft(level, class byte) int {
	c := e.selected()
	n := 0
	for i := 0; i <= 0x14; i++ {
		if s := c.Memorized[i]; s > 0 && e.spellLevel(s) == level && e.spellClass(s) == class {
			n++
		}
	}
	return int(byte(int(c.spellSlot(int(class), int(level))) - n))
}

// canStudy = ovr16:036C (mode 1 cast, 2 memorize, 3 scribe): area +0x1CA
// forbids casting; the dead (status 1) and the fallen cannot study.
func (e *Engine) canStudy(mode int) bool {
	msg := ""
	c := e.selected()
	switch {
	case mode == 1 && wordAt(e.area, 0x1CA) > 0:
		msg = "cannot cast spells in this area"
	case c.Status == 1 || c.Okay == 0:
		msg = "is in no condition to "
		switch mode {
		case 1:
			msg += "cast any spells"
		case 2:
			msg += "memorize spells"
		case 3:
			msg += "scribe any scrolls"
		}
	}
	if msg == "" {
		return true
	}
	e.CharMessage(c, msg, 0xA, true)
	return false
}

// CastMenu = ovr16:04A1: cast memorised spells until none is picked.
func (e *Engine) CastMenu() {
	drew := false
	index := -1
	e.spellCaster = nil // DS:6DB5
	if !e.canStudy(1) {
		return
	}
	for {
		var any bool
		spell := e.SpellList(0, 1, &index, &any)
		switch {
		case spell != 0:
			drew = true
			e.DrawBorder(1, 0x11, 0x26, 0x16, 0)
			e.ClearCells(1, 0x11, 0x26, 0x16)
			var quit bool
			e.CastSpell(spell, 0, 1, &quit)
			index = -1
		case any:
			drew = true
		default:
			e.CharMessage(e.selected(), "has no spells memorized", 0xA, true)
		}
		if spell == 0 {
			break
		}
	}
	if drew {
		e.DrawMainScreen()
	}
}

// memorizeInfo = ovr16:05A4: "can memorize:" with the free slots per class
// and level in the message box; false when there are none at all.
func (e *Engine) memorizeInfo() bool {
	c := e.selected()
	var grid [2][4]string
	var has [2]bool
	any := false
	for cls := 0; cls <= 1; cls++ {
		for lvl := 1; lvl <= 3; lvl++ {
			s := strconv.Itoa(int(int8(e.slotsLeft(byte(lvl), byte(cls)))))
			if len(s) > 2 {
				s = s[:2]
			}
			grid[cls][lvl] = s
			if c.spellSlot(cls, lvl) == 0 {
				grid[cls][lvl] = " "
			} else {
				has[cls] = true
				any = true
			}
		}
	}
	if !any {
		return false
	}
	e.CharMessage(c, "can memorize:", 0xA, false)
	row := 3
	for cls := 0; cls <= 1; cls++ {
		if !has[cls] {
			continue
		}
		label := "    Cleric Spells:"
		if cls == 1 {
			label = "Magic-User Spells:"
		}
		e.DisplayString(1, row+0x11, 0xA, label)
		x := 0x14 // ovr16:0723: [bp-0x41]=0x13, passed +1
		for lvl := 1; lvl <= 3; lvl++ {
			e.DisplayString(x, row+0x11, 0xA, grid[cls][lvl])
			x += 3
		}
		row++
	}
	return true
}

// restMenu = ovr16:077A (var interrupted): the longest study time of the
// party becomes the proposed rest time, then rest (ovr21:0C9A).
func (e *Engine) restMenu() (interrupted bool) {
	most := 0
	for _, c := range e.party {
		if t := e.studyTime(c); t > most {
			most = t
		}
	}
	r := &e.camp.rest
	r[3] = most / 60
	r[2] = (most - 60*r[3]) / 10
	r[1] = most % 10
	interrupted = e.Rest()
	e.camp.rest = [7]int{}
	e.StatusLine()
	return
}

// sortSpells = ovr16:0824: the selected character's spell slots in spell
// order (empty slots first).
func (e *Engine) sortSpells() {
	c := e.selected()
	s := c.Memorized[:]
	for i := 0; i <= 0x13; i++ {
		for j := i; j <= 0x14; j++ {
			if s[i]&0x7F > s[j]&0x7F {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// memorizeMenu = ovr16:090A.
func (e *Engine) memorizeMenu() {
	if !e.canStudy(2) {
		return
	}
	done := false
	index := -1
	var any bool
	e.SpellList(5, 0, &index, &any)
	drew := true
	if any {
		if e.YesNo(0xE, 0xA, 0xF, "Memorize These Spells? ") == 'N' {
			clearMemorize(e.selected())
		} else {
			done = true
		}
	} else {
		drew = false
	}
	index = -1
	for !done {
		done = !e.memorizeInfo()
		if done {
			e.CharMessage(e.selected(), "cannot memorize any spells", 0xA, true)
			continue
		}
		spell := e.SpellList(1, 2, &index, &any)
		drew = true
		if spell == 0 {
			done = true
			continue
		}
		if e.slotsLeft(e.spellLevel(spell), e.spellClass(spell)) > 0 { // unsigned test (jbe) in the original
			c := e.selected()
			k := 0
			for c.Memorized[k] != 0 {
				k++
			}
			c.Memorized[k] = spell + 0x80
			e.sortSpells()
		}
	}
	if index != -1 {
		index = -1
		e.SpellList(5, 0, &index, &any)
		if any && e.YesNo(0xE, 0xA, 0xF, "Memorize these spells? ") == 'N' {
			clearMemorize(e.selected())
		}
	}
	if drew {
		e.DrawMainScreen()
	}
}

// scribeMenu = ovr16:0B5B.
func (e *Engine) scribeMenu() {
	if !e.canStudy(3) {
		return
	}
	done := false
	index := -1
	var any bool
	e.SpellList(6, 0, &index, &any)
	drew := true
	if any {
		if e.YesNo(0xE, 0xA, 0xF, "Scribe These Spells? ") == 'N' {
			e.clearScribe(e.selected())
		} else {
			done = true
		}
	} else {
		drew = false
	}
	index = -1
	for !done {
		spell := e.SpellList(3, 3, &index, &any)
		c := e.selected()
		if spell == 0 {
			done = true
			if !any {
				e.CharMessage(c, "has no copyable scrolls", 0xA, true)
			} else {
				drew = true
			}
			continue
		}
		drew = true
		if c.SpellBook[int(spell)-1] != 0 {
			e.BottomMessage("You already know that spell")
			continue
		}
		found := false
		for _, it := range c.Items {
			if found {
				break
			}
			if !e.isScroll(it) {
				continue
			}
			for k := 1; k <= 3; k++ {
				if s := it.Spells[k-1]; s > 0x7F && s&0x7F == spell {
					e.BottomMessage("You are already scibing that spell")
					found = true
				}
			}
		}
		if found {
			continue
		}
		if c.spellSlot(int(e.spellClass(spell)), int(e.spellLevel(spell))) == 0 {
			e.BottomMessage("You can not scribe that spell.")
			continue
		}
		// (any item's spell bytes count here, not only scrolls)
		for _, it := range c.Items {
			if found {
				break
			}
			for k := 1; k <= 3 && !found; k++ {
				if it.Spells[k-1] == spell {
					it.Spells[k-1] |= 0x80
					found = true
				}
			}
		}
	}
	if index != -1 {
		index = -1
		e.SpellList(6, 0, &index, &any)
		if any && e.YesNo(0xE, 0xA, 0xF, "Scribe these spells? ") == 'N' {
			e.clearScribe(e.selected())
		}
	}
	if drew {
		e.DrawMainScreen()
	}
}

// listedAffect = the affect types ovr16:0FBC names after the spell that
// gives them (cs:0x1050 checks).
func listedAffect(t byte) bool {
	switch {
	case t == 1, t == 2, t == 5, t >= 8 && t <= 0xE, t == 0x10, t == 0x11,
		t >= 0x13 && t <= 0x19, t == 0x1C, t == 0x1D, t >= 0x20 && t <= 0x22,
		t >= 0x24 && t <= 0x27, t == 0x29, t == 0x2A, t == 0x2D, t == 0x2E,
		t == 0x31, t >= 0x33 && t <= 0x35:
		return true
	}
	return false
}

// affectNames = ovr16:0FBC's named affects that no spell list entry covers.
var affectNames = map[byte]string{
	0x04: "Studying Manual of Bodily Health",
	0x07: "Training with Manual of Bodily Health",
	0x1B: "Feather Fall",
	0x1F: "Helpless",
	0x23: "Prayer",
	0x2C: "Cause Disease",
	0x32: "Dreaded Mummy Disease",
	0x36: "Repulsed",
	0x37: "Poisoned",
	0x3B: "Regenerating",
	0x3D: "Fire Resistance",
	0x47: "Invisible",
	0x48: "Camouflaged",
	0x59: "Displaced",
}

// affectList = the names ovr16:0FBC lists for c: its spell effects.
func (e *Engine) affectList(c *Character) []string {
	var names []string
	for _, a := range c.Affects {
		t := a.Type
		name, known := "", true
		if listedAffect(t) {
			for s := 1; s <= 0x38 && name == ""; s++ {
				if e.X.Bytes(0x3204+16*s, 1)[0] == t {
					name = e.spellName(s)
				}
			}
			if name == "" {
				name = "Funky--" + itoa(int(t))
			}
		} else if name, known = affectNames[t]; !known {
			continue
		}
		names = append(names, name)
	}
	return names
}

// displayAffects = ovr16:0FBC: the members with the spell effects on them,
// under a box titled title. The list appends are the nested ovr16:0E37
// (text, flag). The caller redraws the screen.
func (e *Engine) displayAffects(title string, members []*Character) {
	list := []*ListNode{{Text: " "}}
	for _, c := range members {
		list = append(list, &ListNode{Text: c.Name(), Flag: 1})
		names := e.affectList(c)
		for _, name := range names {
			list = append(list, &ListNode{Text: cut(" "+name, 0x28)})
		}
		if len(names) == 0 {
			list = append(list, &ListNode{Text: " <No Spell Effects>"})
		}
		list = append(list, &ListNode{Text: " "})
	}
	redraw := true
	index := 0
	var sel *ListNode
	e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0xF, title)
	e.ListMenu("", "", 0xB, 0xA, 0xF, 1, 4, 0x26, 0x16, list, true, &redraw, &index, &sel)
}

// magicMenu = ovr16:13F1 (var interrupted).
func (e *Engine) magicMenu(interrupted *bool) {
	for key := byte(' '); !*interrupted && key != 0 && key != 'E'; {
		var extra bool
		menu := "Cast Memorize Scribe Rest Exit" // the original's Display: the View page shows the spell effects instead
		key, extra = e.DisplayInput("", menu, 0xD, 0xA, 0xF, true, true)
		if extra {
			e.partyKeypad(key)
			e.DisplayPartyList()
			continue
		}
		switch key {
		case 'C':
			e.CastMenu()
		case 'M':
			e.memorizeMenu()
		case 'S':
			e.scribeMenu()
		case 'R':
			*interrupted = e.restMenu()
		}
	}
}

// movePrev = ovr16:14A9: the selected member swaps places with the one
// before; the first goes to the end.
func (e *Engine) movePrev() {
	i := e.partySel
	if i == 0 {
		c := e.party[0]
		e.party = append(e.party[1:], c)
		e.partySel = len(e.party) - 1
		return
	}
	e.party[i-1], e.party[i] = e.party[i], e.party[i-1]
	e.partySel = i - 1
}

// moveNext = ovr16:1601: swaps with the one after; the last goes first.
func (e *Engine) moveNext() {
	i := e.partySel
	if i == len(e.party)-1 {
		c := e.party[i]
		e.party = append([]*Character{c}, e.party[:i]...)
		e.partySel = 0
		return
	}
	e.party[i+1], e.party[i] = e.party[i], e.party[i+1]
	e.partySel = i + 1
}

// orderMenu = ovr16:1780: Select a member, then Place it with Home/End.
func (e *Engine) orderMenu() {
	placing := false
	for key := byte(' '); key != 0 && key != 'E'; {
		menu := "Select Exit"
		if placing {
			menu = "Place Exit"
		}
		var extra bool
		key, extra = e.DisplayInput("Party Order: ", menu, 0xD, 0xA, 0xF, true, true)
		if extra {
			if !placing {
				e.partyKeypad(key)
			} else if key == 'G' {
				e.movePrev()
			} else if key == 'O' {
				e.moveNext()
			}
			e.DisplayPartyList()
			continue
		}
		if key == 13 || key == 'P' || key == 'S' {
			placing = !placing
			if placing {
				e.CharMessage(e.selected(), "has been selected", 0xA, false)
			} else {
				e.ClearMessageArea()
			}
		}
	}
}

// dropMenu = ovr16:18E5: drop the selected member (the last one quits).
func (e *Engine) dropMenu() {
	c := e.selected()
	if len(e.party) == 1 {
		if e.YesNo(0xE, 0xA, 0xF, "quit TO DOS: ") == 'Y' {
			e.RemoveSelected(false, true)
			e.Exit()
		}
		return
	}
	e.CharMessage(c, "will be gone", 0xA, false)
	if e.YesNo(0xE, 0xA, 0xF, "Drop from party? ") != 'Y' {
		e.CharMessage(c, "Breathes A sigh of relief", 0xA, true)
		return
	}
	if c.Okay != 0 {
		e.CharMessage(c, "bids you farewell", 0xA, true)
	} else {
		e.CharMessage(c, "is dumped in a ditch", 0xA, true)
	}
	e.RemoveSelected(false, true)
	e.ClearCells(0x11, 1, 0x26, 0xB)
	e.DisplayPartyList()
}

// campSpeedMenu = ovr16:1A97: the message delay (DS:49E2), 0 fastest.
func (e *Engine) campSpeedMenu() {
	for key := byte(' '); key != 0 && key != 'E'; {
		e.DisplayString(1, 0x12, 0xA, "Game Speed = "+itoa(int(e.msgDelay))+" (0=fastest 9=slowest)")
		menu := ""
		if e.msgDelay > 0 {
			menu += " Faster"
		}
		if e.msgDelay < 9 {
			menu += " Slower"
		}
		menu += " Exit"
		var extra bool
		key, extra = e.DisplayInput("Game Speed:", menu, 0xD, 0xA, 0xF, true, true)
		if extra {
			if key == 0x50 && e.msgDelay > 0 {
				e.msgDelay--
			} else if key == 0x48 && e.msgDelay < 9 {
				e.msgDelay++
			}
		} else if key == 'F' {
			e.msgDelay--
		} else if key == 'S' {
			e.msgDelay++
		}
	}
	e.ClearMessageArea()
}

// alterMenu = ovr16:1C59.
func (e *Engine) alterMenu() {
	for key := byte(' '); key != 0 && key != 'E'; {
		var extra bool
		key, extra = e.DisplayInput("Alter: ", "Order Drop Speed Icon Pics Exit", 0xD, 0xA, 0xF, true, true)
		if extra {
			e.partyKeypad(key)
			e.DisplayPartyList()
			continue
		}
		switch key {
		case 'O':
			e.orderMenu()
		case 'D':
			e.dropMenu()
		case 'S':
			e.campSpeedMenu()
		case 'I':
			e.o17_382D()
			e.DrawMainScreen()
		case 'P':
			for k := byte(' '); k != 0 && k != 'E'; {
				menu := "Monsters off  "
				if e.flag49F6 != 0 {
					menu = "Monsters on  "
				}
				if e.flag49F5 != 0 {
					menu += "Portraits on  "
				} else {
					menu += "Portraits off  "
				}
				k, _ = e.DisplayInput("", menu+"Exit", 0xD, 0xA, 0xF, false, true)
				if k == 'M' {
					e.flag49F6 ^= 1
				} else if k == 'P' {
					e.flag49F5 ^= 1
				}
			}
		}
	}
}

// CampMenu = ovr16:1E6F: true when resting was interrupted (the caller
// then runs the encounter).
func (e *Engine) CampMenu() bool {
	saved := e.beginFlag
	e.beginFlag = 2
	e.camp.heal = 0
	e.camp.rest = [7]int{}
	e.DrawMainScreen()
	e.ClearCells(1, 0x11, 0x26, 0x16)
	e.DisplayString(1, 0x12, 0xA, "The party makes camp...")
	e.clearStudy()
	interrupted := false
	for key := byte(' '); !interrupted && key != 0 && key != 'E'; {
		var extra bool
		key, extra = e.DisplayInput("Camp: ", "Save View Magic Rest Alter Exit", 0xD, 0xA, 0xF, true, true)
		if extra {
			e.partyKeypad(key)
			e.DisplayPartyList()
			continue
		}
		switch key {
		case 'S':
			e.SaveGame()
			if e.YesNo(0xE, 0xA, 0xF, "Quit TO DOS ") == 'Y' {
				e.Exit()
			}
		case 'V':
			e.ViewCharacter()
		case 'M':
			e.magicMenu(&interrupted)
		case 'R':
			interrupted = e.restMenu()
		case 'A':
			e.alterMenu()
		}
	}
	e.clearStudy()
	e.spellCaster = nil
	e.beginFlag = saved
	e.StatusLine()
	e.ClearMessageArea()
	e.ClearRow24()
	return interrupted
}

// ---- ovr21: resting ----

// subRest = ovr21:047A (count, unit): take count units off the rest time,
// borrowing from the larger units; all of it when that runs out.
func (e *Engine) subRest(count, unit int) {
	r := &e.camp.rest
	if r[4] == 0 && r[3] == 0 && r[2] == 0 && r[1] == 0 {
		return
	}
	for count > r[unit] {
		k := unit + 1
		for r[k] == 0 && k < 5 {
			k++
		}
		if k == 5 {
			*r = [7]int{}
			count = 0
			continue
		}
		for j := k; j >= unit+1; j-- {
			r[j]--
			r[j-1] += clockLimits[j-1]
		}
	}
	r[unit] -= count
	e.normalizeRest()
}

// normalizeRest = ovr21:03A3: carries, months fold into days, at most 99.
func (e *Engine) normalizeRest() {
	r := &e.camp.rest
	for i := 0; i < 5; i++ {
		if r[i] >= clockLimits[i] {
			r[i+1] += r[i] / clockLimits[i]
			r[i] %= clockLimits[i]
		}
	}
	if r[5] > 0 {
		r[4] += clockLimits[4] * r[5]
		r[5] = 0
		if r[4] > 0x63 {
			r[4] = 0x63
		}
	}
}

// twoDigits = ovr21:0565.
func twoDigits(n int) string {
	s := itoa(n)
	if n < 10 {
		s = "0" + s
	}
	return s[:2]
}

// showRest = ovr21:05F0 (unit): "Rest Time: dd:hh:mm", the unit being
// changed in white.
func (e *Engine) showRest(unit int) {
	var col [7]int
	for i := range col {
		col[i] = 0xA
	}
	col[unit] = 0xF
	r := &e.camp.rest
	e.DisplayString(1, 0x11, 0xA, "Rest Time:")
	x := 0xB
	for i := 4; i >= 3; i-- {
		e.DisplayString(x+1, 0x11, col[i], twoDigits(r[i]))
		e.DisplayString(x+3, 0x11, 0xA, ":")
		x += 3
	}
	e.DisplayString(x+1, 0x11, col[2], twoDigits(10*r[2]+r[1]))
}

// restSetup = ovr21:0721: set the rest time; true to start resting.
func (e *Engine) restSetup() bool {
	start := false
	unit := 2
	r := &e.camp.rest
	defer e.rawArrows("↑↓=more, less|← →=days, hours, mins|letter, click=choose|Enter=rest|Esc=back")()
	for key := byte(0); ; {
		e.showRest(unit)
		var extra bool
		key, extra = e.DisplayInput("", "Rest   daYs Hours Mins   Inc Dec   Exit", 0xD, 0xA, 0xF, true, false)
		if extra {
			switch key {
			case 0x48:
				key = 'I'
			case 0x50:
				key = 'D'
			case 0x4B:
				if unit++; unit > 4 {
					unit = 2
				}
				key = 'X'
			case 0x4D:
				if unit--; unit < 2 {
					unit = 4
				}
				key = 'X'
			default:
				key = 'X'
			}
		}
		if key == 13 {
			key = 'R'
		}
		switch key {
		case 'R':
			start = true
		case 'Y':
			unit = 4
		case 'H':
			unit = 3
		case 'M':
			unit = 2
		case 'I':
			if unit == 2 {
				r[1] += 5
			} else {
				r[unit]++
			}
			e.normalizeRest()
		case 'D':
			if unit == 2 {
				e.subRest(5, 1)
			} else {
				e.subRest(1, unit)
			}
			e.normalizeRest()
		}
		if key == 0 || key == 'E' || key == 'R' {
			return start
		}
	}
}

// healTick = ovr21:0871: every 24 hours of rest everyone heals 1 HP.
func (e *Engine) healTick() {
	e.camp.heal++
	if e.camp.heal < 0x120 {
		return
	}
	healed := false
	for _, c := range e.party {
		if e.Heal(c, 1, false) {
			healed = true
		}
	}
	e.showRest(0)
	e.DisplayString(1, 0x13, 0xA, "The Whole Party Is Healed")
	if healed {
		e.DisplayPartyList()
	}
	e.MsgDelay()
	e.ClearMessageArea()
	e.camp.heal = 0
}

// memorizeOne = ovr21:0925 (char, var check): with check clear the first
// waiting spell is memorised (and check set); returns the level of the
// next waiting spell, 0 when none is left.
func (e *Engine) memorizeOne(c *Character, check *bool) int {
	found := 0
	for i := 0; i <= 0x14 && found == 0; i++ {
		s := c.Memorized[i]
		if s <= 0x7F {
			continue
		}
		if *check {
			found = int(e.spellLevel(s))
			continue
		}
		c.Memorized[i] -= 0x80
		e.showRest(0)
		e.o23_331A(c, "has memorized", int(c.Memorized[i]))
		*check = true
	}
	return found
}

// scribeOne = ovr21:09E7 (char, var check): memorizeOne for scrolls being
// scribed; the spell goes into the book and off the scroll.
func (e *Engine) scribeOne(c *Character, check *bool) int {
	found := 0
	for _, it := range append([]*Item(nil), c.Items...) {
		if found != 0 {
			break
		}
		if !e.isScroll(it) {
			continue
		}
		for k := 1; k <= 3 && found == 0; k++ {
			s := it.Spells[k-1]
			if s <= 0x80 {
				continue
			}
			if *check {
				found = int(e.spellLevel(s))
				continue
			}
			sp := s & 0x7F
			c.SpellBook[int(sp)-1] = 1
			e.scrollUsed(c, it, sp)
			e.showRest(0)
			e.o23_331A(c, "has scribed", int(sp))
			*check = true
		}
	}
	return found
}

// studyStep = the shared tail of ovr21:0B0A / 0BB2: learn what is due and
// time the next spell (3 steps per spell level).
func (e *Engine) studyStep(i int, c *Character, check bool) {
	lvl := e.scribeOne(c, &check)
	if lvl == 0 {
		lvl = e.memorizeOne(c, &check)
	}
	e.camp.study[i] = byte(lvl * 3)
}

// memTick = ovr21:0B0A: every five minutes.
func (e *Engine) memTick() {
	for i, c := range e.party {
		if i+1 >= len(e.camp.study) {
			break
		}
		e.camp.study[i+1]--
		if e.camp.study[i+1] == 0 && c.Study == 0 {
			e.studyStep(i+1, c, false)
		}
	}
}

// hourTick = ovr21:0BB2 (var count): every hour the study hours run down.
func (e *Engine) hourTick(count *int) {
	*count++
	if *count < 12 {
		return
	}
	*count = 0
	for i, c := range e.party {
		if i+1 >= len(e.camp.study) {
			break
		}
		if int8(c.Study) > 0 {
			c.Study--
			if c.Study == 0 {
				e.studyStep(i+1, c, true)
			}
		}
	}
}

// Rest = ovr21:0C9A: rest in five-minute steps until the time is up, a key
// stops it, or a monster interrupts (game +0x5A4 steps between rolls,
// +0x5A6 percent chance). Returns true when interrupted.
func (e *Engine) Rest() bool {
	interrupted := false
	for i := range e.camp.study {
		e.camp.study[i] = 0
	}
	for i := 1; i < len(e.run.effTick); i++ {
		e.run.effTick[i] = 1
	}
	hours, shown := 0, 0
	e.ClearBox(1, 0x11, 0x26, 0x16)
	e.showRest(0)
	e.resting = 1
	done := !e.restSetup()
	r := &e.camp.rest
	for !done {
		if r[4] == 0 && r[3] == 0 && r[2] == 0 && r[1] == 0 {
			break
		}
		if e.keyWaiting() {
			e.showRest(0)
			if e.YesNo(0xD, 0xA, 0xF, "Stop Resting? ") == 'Y' {
				done = true
			} else {
				e.ClearRow24()
			}
		}
		if done {
			continue
		}
		e.subRest(5, 1)
		if shown++; shown >= 5 {
			e.showRest(0)
			shown = 0
		}
		e.AdvanceClock(5, 1)
		e.healTick()
		e.memTick()
		e.hourTick(&hours)
		if every := wordAt(e.game, 0x5A4); every > 0 {
			e.camp.interval++
			if e.camp.interval >= int(every) {
				e.camp.interval = 0
				if e.Dice(1, 100) <= int(wordAt(e.game, 0x5A6)) {
					e.ClearMessageArea()
					e.showRest(0)
					e.DisplayString(1, 0x13, 0xF, "The Party is rudely interrupted!")
					done = true
					interrupted = true
					e.MsgDelay()
				}
			}
		}
		e.idle() // let the window breathe; the DOS loop ran flat out
	}
	e.ClearBox(1, 0x11, 0x26, 0x16)
	e.resting = 0
	return interrupted
}
