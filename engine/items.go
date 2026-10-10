package engine

// Items: names, lists and the START helpers around them.

// Item record fields (0x3F bytes; the name is built into +0).
const (
	itName    = 0x00 // pascal string, max 0x28 (built by ItemName)
	itType    = 0x2E // base item type (index into DS:5582 records)
	itWords   = 0x2F // 3 word ids into DS:10A7 (0x15 bytes each)
	itPlus    = 0x32 // signed
	itPlus2   = 0x33
	itReadied = 0x34
	itHidden  = 0x35 // bit 3-i: word i not known yet
	itMagic   = 0x36
	itCount   = 0x39
)

// itemWord = DS:10A7 + 0x15*n.
func (e *Engine) itemWord(n byte) string { return e.X.PString(0x10A7 + 0x15*int(n)) }

func (it *Item) Name() string {
	n := int(it.Raw[itName])
	if n > 0x28 {
		n = 0x28
	}
	return string(it.Raw[1 : 1+n])
}

func (it *Item) setName(s string) {
	s = cut(s, 0x28)
	it.Raw[itName] = byte(len(s))
	copy(it.Raw[1:0x29], s)
}

// ItemName = 00BA:0442 (char, item, x, y, readied, display): builds the
// item's shown name into its name field ("Yes"/"No" readied column, "*"
// for magic when someone detects magic, count, the known words with the
// plural "s" on the right one) and optionally prints it.
func (e *Engine) ItemName(c *Character, it *Item, x, y int, readied, display bool) {
	r := &it.Raw
	name := ""
	if readied {
		if r[itReadied] != 0 {
			name = " Yes  "
		} else {
			name = " No   "
		}
	}
	detect := false
	for _, p := range e.party {
		if detect {
			break
		}
		if e.FindAffect(p, 5) != nil {
			detect = true
		}
	}
	if detect && (int8(r[itPlus]) > 0 || int8(r[itPlus2]) > 0 || r[itMagic] != 0) {
		name = cut(name+"* ", 0x28)
	}
	if r[itCount] > 0 {
		name = cut(name+itoa(int(r[itCount]))+" ", 0x28)
	}
	mask := 0
	for i := 1; i <= 3; i++ {
		if r[itType+i] != 0 && (r[itHidden]>>(3-i))&1 == 0 {
			mask += 1 << (i - 1)
		}
	}
	plural := false
	for i := 3; i >= 1; i-- {
		if (mask>>(i-1))&1 == 0 {
			continue
		}
		name = cut(name+e.itemWord(r[itType+i]), 0x28)
		s := " "
		if r[itCount] >= 2 && !plural {
			typ := r[itType]
			switch {
			case 1<<(i-1) == mask,
				i == 1 && mask > 4 && typ != 0x56,
				i == 2 && mask&1 == 0,
				i == 3 && typ == 0x56,
				(typ == 0x49 || typ == 0x1C) && r[itWords+2] != 0xB1:
				s = "s "
				plural = true
			}
		}
		name = cut(name+s, 0x28)
	}
	it.setName(name)
	if display {
		e.DisplayString(x, y, 0xA, it.Name())
	}
}

// RemoveItem = 00BA:1509 (char, item): unlinks the item and frees it.
func (e *Engine) RemoveItem(c *Character, it *Item) {
	if it == nil {
		e.MessageRow24b(0xE, "Nil Item pointer...")
		return
	}
	for i, x := range c.Items {
		if x == it {
			c.Items = append(c.Items[:i:i], c.Items[i+1:]...)
			return
		}
	}
	e.MessageRow24b(0xE, "Tried to Lose item & couldn't find it!")
}

// AddItemCopy = 00BA:15F7 (char, item): a copy of the item at the end of
// the character's list.
func (e *Engine) AddItemCopy(c *Character, it *Item) *Item {
	n := *it
	c.Items = append(c.Items, &n)
	return &n
}

// BottomMessage = 00BA:1693: the string on row 24 for the message delay.
func (e *Engine) BottomMessage(s string) {
	s = cut(s, 0x28)
	e.ClearCells(0, 24, 0x27, 24)
	e.DisplayString(0, 24, 0xA, s)
	e.MsgDelay()
	e.ClearCells(0, 24, 0x27, 24)
}
