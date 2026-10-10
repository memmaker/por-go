package engine

import "strings"

// The shop, ported from ovr07. ECL COMBAT with the shop flag (game +0x6D8)
// set runs Shop instead of a fight; the goods are the treasure list.

// shopList = ovr07:0030 (var index, var list, var item, var key): the goods
// with their prices right-aligned in a 30-character field, in a "Shop" box.
func (e *Engine) shopList(index *int, item **Item) byte {
	list := make([]*ListNode, len(e.run.treasure))
	for i, it := range e.run.treasure {
		if wordAt(it.Raw[:], 0x3A) == 0 {
			putWord(it.Raw[:], 0x3A, 1)
		}
		price := itoa(int(wordAt(it.Raw[:], 0x3A)))
		if len(price) > 0x10 {
			price = price[:0x10]
		}
		line := []byte(strings.Repeat(" ", 0x1E))
		copy(line[0x1E-len(price):], price)
		copy(line, it.Name()) // (a name longer than 30 is cut off)
		list[i] = &ListNode{Text: string(line)}
		list[i].Dim = !e.canUseItem(e.selected(), it) // extra: what the buyer cannot use is grey and passed over
	}
	e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0xF, "Shop")
	e.DisplayString(2, 2, 0xE, e.moneyLine())
	redraw := true
	var sel *ListNode
	{ // extra: the cursor starts on something the buyer can use
		for k := 0; k < len(list) && list[*index%len(list)].Dim; k++ {
			*index = (*index + 1) % len(list)
		}
		*index %= len(list)
	}
	e.ext.buying = true
	key := e.ListMenu("Items: ", "Buy", 0xD, 0xA, 0xF, 1, 4, 0x26, 0x16, list, true, &redraw, index, &sel)
	e.ext.buying = false
	*item = nil
	for i, n := range list {
		if n == sel {
			*item = e.run.treasure[i]
		}
	}
	for _, it := range e.run.treasure {
		e.ItemName(nil, it, 0, 0, false, false)
	}
	return key
}

// buyItem = ovr07:0239 (item, var overloaded): a copy goes to the
// selected character unless it is too much to carry.
func (e *Engine) buyItem(it *Item) (overloaded bool) {
	c := e.selected()
	if e.overloaded(it, c) {
		e.BottomMessage("OverLoaded")
		return true
	}
	e.AddItemCopy(c, it)
	e.ApplyItems(c)
	return false
}

// shopBuy = ovr07:0361: pick goods until Exit, paid from the party's gold.
func (e *Engine) shopBuy() {
	// extra: the list opens at its top on the first item (the original,
	// ovr07:037C, starts on the second, with the top left where the last
	// list had it)
	index := 0
	e.listTop = 0
	for {
		var it *Item
		key := e.shopList(&index, &it)
		if key == keyRelist>>8 {
			continue // F1-F6 picked another buyer
		}
		if key != 'B' && key != 13 || it == nil {
			return
		}
		price := int(wordAt(it.Raw[:], 0x3A))
		if price > e.purse.gold {
			e.BottomMessage("Not enough Money.")
		} else if !e.buyItem(it) {
			e.purse.gold -= price
		}
	}
}

// Shop = ovr07:053E.
func (e *Engine) Shop() {
	e.beginFlag = 1
	e.DrawMainScreen()
	e.DisplayPartyList()
	for _, it := range e.run.treasure {
		e.ItemName(nil, it, 0, 0, false, false)
	}
	for leave := false; !leave; {
		e.shopMoney()
		key, _ := e.DisplayInput("", "Buy View Appraise Exit", 0xD, 0xA, 0xF, true, false)
		appraised := false
		switch key {
		case 'B':
			e.shopBuy()
		case 'V':
			e.ViewCharacter()
		case 'A':
			appraised = e.Appraise()
		case 'E':
			leave = true
		case 'G', 'O':
			e.partyKeypad(key)
		}
		if key == 'B' || key == 'A' && appraised {
			e.DrawMainScreen()
		}
		e.DisplayPartyList()
	}
}

// canUseItem (extra): the item's class mask lets c ready it (the Wrong
// Class test of readyItem).
func (e *Engine) canUseItem(c *Character, it *Item) bool {
	return e.baseItem(it.Raw[itType], 0xD)&c.Raw[0xB0] != 0
}
