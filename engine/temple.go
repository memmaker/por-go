package engine

// The temple, ported from ovr05. ECL COMBAT with the temple flag (game
// +0x5C4) set runs Temple instead of a fight.

// templeNotNeeded = ovr05:0013 (message): "<name> is not ..." and the
// question whether to pay anyway.
func (e *Engine) templeNotNeeded(msg string) byte {
	e.CharMessage(e.selected(), msg, 0, false)
	k := e.YesNo(0xD, 0xA, 0xF, "cast cure anyway: ")
	e.ClearMessageArea()
	return k
}

// templePay = ovr05:00BF (cure, price): the price from the purse, or else
// from the pool. 'Y' when paid.
func (e *Engine) templePay(cure string, price uint16) byte {
	e.DisplayTextBlock(1, 0x11, 0x26, 0x16, 0xA, true,
		cut(cure+" will only cost "+itoa(int(price))+" gold pieces.", 0xFF))
	k := e.YesNo(0xD, 0xF, 0xA, "pay for cure ")
	if k == 'Y' {
		c := e.selected()
		if purse := e.charMoneyValue(c); int32(price) <= purse {
			e.setCharMoney(uint16(purse - int32(price)))
		} else if pool := e.poolValue(); int32(price) <= pool {
			e.setPoolMoney(uint16(pool - int32(price)))
		} else {
			e.BottomMessage("Not enough money.")
			k = 'N'
		}
	}
	if k == 'Y' {
		e.ClearMessageArea()
		e.CharMessage(e.selected(), "is cured.", 0, true)
	}
	return k
}

// templeCure asks "not needed" when needed is false, then for the price.
func (e *Engine) templeCure(needed bool, notMsg, cure string, price uint16) bool {
	k := byte('Y')
	if !needed {
		k = e.templeNotNeeded(notMsg)
	}
	return k == 'Y' && e.templePay(cure, price) == 'Y'
}

// cureBlindness = ovr05:0249.
func (e *Engine) cureBlindness() {
	c := e.selected()
	if e.templeCure(e.FindAffect(c, 0x21) != nil, "is not blind.", "Cure Blindness", 1000) {
		e.RemoveAffect(c, 0x21, nil)
	}
}

// diseases = DS:0114: the affects Cure Disease removes.
var diseases = [6]byte{0x1F, 0x22, 0x2B, 0x2C, 0x32, 0x39}

// cureDisease = ovr05:02ED.
func (e *Engine) cureDisease() {
	c := e.selected()
	ill := false
	for _, d := range diseases {
		if e.FindAffect(c, d) != nil {
			ill = true
		}
	}
	if e.templeCure(ill, "is not Diseased.", "Cure Disease", 1000) {
		e.rv.cure = 1
		for _, d := range diseases {
			e.RemoveAffect(c, d, nil)
		}
		e.rv.cure = 0
	}
}

// cureWounds = ovr05:0401 (kind): light, serious or critical.
func (e *Engine) cureWounds(kind int) {
	c := e.selected()
	switch kind {
	case 1:
		if e.templePay("Cure Light Wounds", 100) == 'Y' {
			e.Heal(c, e.Dice(1, 8), false)
		}
	case 2:
		if e.templePay("Cure Serious Wounds", 350) == 'Y' {
			e.Heal(c, e.Dice(2, 8)+1, false)
		}
	case 3:
		if e.templePay("Cure Critical Wounds", 600) == 'Y' {
			e.Heal(c, e.Dice(3, 8)+3, false)
		}
	}
}

// neutralizePoison = ovr05:0751.
func (e *Engine) neutralizePoison() {
	c := e.selected()
	if e.templeCure(e.FindAffect(c, 0x37) != nil, "is not poisoned.", "Neutralize Poison", 1000) {
		e.rv.cure = 1
		e.RemoveAffect(c, 0x37, nil)
		e.RemoveAffect(c, 0x16, nil)
		e.RemoveAffect(c, 0xF, nil)
		e.rv.cure = 0
	}
}

// raiseDead = ovr05:0510: back with 1 hp and a point of constitution
// less; the constitution bonus to the maximum hit points shrinks with it.
func (e *Engine) raiseDead() {
	c := e.selected()
	r := c.Raw[:]
	dead := r[chStatus] == 6 || r[chStatus] == 1
	if !e.templeCure(dead, "is not dead.", "Raise Dead", 5500) || !dead {
		return
	}
	e.rv.cure = 1
	e.RemoveAffect(c, 0x20, nil)
	e.RemoveAffect(c, 0x37, nil)
	e.rv.cure = 0
	r[chHP] = 1
	r[chStatus] = 0
	r[chOkay] = 1
	const con = 0x14
	if r[con] > 0 {
		r[con]--
	}
	var excess byte
	if r[chMaxHP] > r[0xB1] {
		excess = r[chMaxHP] - r[0xB1]
	}
	if r[con] < 0xE {
		return
	}
	var levels byte
	for i := 0; i <= 7; i++ {
		lvl := int8(r[0x96+i])
		if lvl <= 0 {
			continue
		}
		switch {
		case i == 2: // fighters get the exceptional bonus
			levels += byte(int(lvl) * (int(r[con]) - 0xE))
		case r[con] > 0xF:
			levels += byte(lvl) * 2
		default:
			levels += byte(lvl)
		}
	}
	if levels > 0 {
		excess = byte(int(excess) / int(levels))
	}
	if r[con] < 0x11 || int8(r[0x98]) > 0 {
		r[chMaxHP] -= excess
	}
}

// removeCurse = ovr05:083A: cursed items or the curse affect; the work is
// the Remove Curse spell (ovr23:2540) on the selected character.
func (e *Engine) removeCurse() {
	c := e.selected()
	cursed := false
	for _, it := range c.Items {
		if it.Raw[itMagic] != 0 {
			cursed = true
			break
		}
	}
	if !cursed && e.FindAffect(c, 0x24) != nil {
		cursed = true
	}
	if e.templeCure(cursed, "is not cursed.", "Remove Curse", 3500) {
		e.spellTarget = c
		e.spellRemoveCurse()
	}
}

// stoneToFlesh = ovr05:0928.
func (e *Engine) stoneToFlesh() {
	c := e.selected()
	if e.templeCure(c.Raw[chStatus] == 7, "is not stoned.", "Stone to Flesh", 2000) && c.Raw[chStatus] == 7 {
		c.Raw[chStatus] = 0
		c.Raw[chOkay] = 1
		c.Raw[chHP] = 1
	}
}

// templeHeal = ovr05:09D7: the list of services (DS:011A).
func (e *Engine) templeHeal() {
	list := make([]*ListNode, 10)
	for i := range list {
		list[i] = &ListNode{Text: cut(e.X.PString(0xF1+0x29*(i+1)), 0x28)}
	}
	e.ClearBox(0, 0x18, 0x27, 0x18)
	redraw := true
	e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0xF, "")
	e.DrawBoxTitled(1, 1, 0x26, 1, 0, 0xF, "")
	index := 0
	for done := false; !done; {
		e.DisplayString(1, 1, 0xF, cut(e.selected().Name()+", how can we help you?", 0x28))
		e.DrawBoxTitled(1, 0x11, 0x26, 0x16, 0, 0xF, "")
		var sel *ListNode
		key := e.ListMenu("", "Heal Exit", 0xD, 0xA, 0xF, 2, 4, 0x26, 0xF, list, false, &redraw, &index, &sel)
		switch {
		case key == 'H' || key == 13:
			switch index {
			case 0:
				e.cureBlindness()
			case 1:
				e.cureDisease()
			case 2, 3, 4:
				e.cureWounds(index - 1)
			case 5:
				e.neutralizePoison()
			case 6:
				e.raiseDead()
			case 7:
				e.removeCurse()
			case 8:
				e.stoneToFlesh()
			case 9:
				done = true
			}
		case key == 0:
			done = true
		}
	}
	e.ClearBox(0, 0, 0x27, 0x18) // ovr05:0C18
	e.DrawMainScreen()
	e.DisplayPartyList()
}

// Temple = ovr05:0D09.
func (e *Engine) Temple() {
	e.beginFlag = 1
	e.DrawMainScreen()
	e.DisplayPartyList()
	e.money = [7]uint32{}
	e.poolHasMoney = false
	for leave := false; !leave; {
		money, _ := e.treasureFlags()
		menu := "Heal View Pool Appraise Exit"
		if money {
			menu = "Heal View Take Pool Share Appraise Exit"
		}
		e.shopMoney()
		key, extra := e.DisplayInput("", menu, 0xD, 0xA, 0xF, true, false)
		appraised := false
		switch key {
		case 'H':
			if !extra {
				e.templeHeal()
			}
		case 'V':
			e.ViewCharacter()
		case 'T':
			e.TakeMoneyMenu()
		case 'P':
			if !extra {
				e.PoolMoney()
			}
		case 'S':
			e.ShareMoney()
		case 'A':
			appraised = e.Appraise()
		case 'E':
			if money, _ := e.treasureFlags(); money {
				e.DisplayTextBlock(1, 0x11, 0x26, 0x16, 0xA, true,
					"As you leave a priest says, \"Excuse me but you have left some money here\" ")
				e.DisplayTextBlock(1, 0x11, 0x26, 0x16, 0xA, true, "Do you want to go back and retrieve your money?")
				if e.MenuInput(false, 0xF, 0xA, 0xD, "~Yes ~No", "") == 1 {
					leave = true
				} else {
					e.ClearBox(1, 0x11, 0x26, 0x16)
				}
			} else {
				leave = true
			}
		case 'G', 'O':
			e.partyKeypad(key)
		}
		if key == 'B' || key == 'T' || key == 'A' && appraised {
			e.DrawMainScreen()
		}
		e.DisplayPartyList()
	}
}
