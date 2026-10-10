package engine

import "strconv"

// ovr22 money: coins on characters (7 words at +0x88: copper, silver,
// electrum, gold, platinum, gems, jewelry) and the treasure pool DS:67F4.

const (
	coinGold     = 3
	coinPlatinum = 4
)

func (c *Character) coins(i int) uint16 { return wordAt(c.Raw[:], chCoins+2*i) }
func (c *Character) setCoins(i int, v uint16) {
	putWord(c.Raw[:], chCoins+2*i, v)
}

// maxWeight = ovr22:0000.
func (e *Engine) maxWeight(c *Character) uint16 { return uint16(strCarry(c) + 1500) }

// subWeight = ovr22:0020, addWeight = ovr22:003A.
func subWeight(c *Character, w uint16) { putWord(c.Raw[:], chWeight, wordAt(c.Raw[:], chWeight)-w) }
func addWeight(c *Character, w uint16) { putWord(c.Raw[:], chWeight, wordAt(c.Raw[:], chWeight)+w) }

// overweight = ovr22:0054 (char, amount, room): true when amount more
// weight does not fit; room is what still does.
func (e *Engine) overweight(c *Character, amount uint16, room *uint16) bool {
	if wordAt(c.Raw[:], chWeight)+amount > e.maxWeight(c) {
		*room = e.maxWeight(c) - wordAt(c.Raw[:], chWeight)
		return true
	}
	*room = 0
	return false
}

// coinValue = DS:0D24 (5 longs), coinUnit = DS:0D30.
func (e *Engine) coinValue(i int) int32 { return int32(e.X.Long(0xD24 + 4*i)) }

// poolValue = ovr22:00A8: the pool's copper..platinum in gold pieces.
func (e *Engine) poolValue() int32 {
	var t int32
	for i := 0; i <= 4; i++ {
		t += int32(e.money[i]) * e.coinValue(i)
	}
	return (t + 100) / int32(e.X.Long(0xD30))
}

// charMoneyValue = ovr20:28BE (char): the same for a character's purse.
func (e *Engine) charMoneyValue(c *Character) int32 {
	var t int32
	for i := 0; i <= 4; i++ {
		t += int32(c.coins(i)) * e.coinValue(i)
	}
	return (t + 100) / int32(e.X.Long(0xD30))
}

// setCharMoney = ovr22:0130 (gold): the selected character's coins
// become platinum and gold worth the amount.
func (e *Engine) setCharMoney(v uint16) {
	c := e.selected()
	for i := 0; i <= 4; i++ {
		c.setCoins(i, 0)
	}
	c.setCoins(coinPlatinum, v/5)
	c.setCoins(coinGold, v%5)
}

// setPoolMoney = ovr22:0185.
func (e *Engine) setPoolMoney(v uint16) {
	for i := 0; i <= 4; i++ {
		e.money[i] = 0
	}
	e.money[coinPlatinum] = uint32(v / 5)
	e.money[coinGold] = uint32(v % 5)
}

// addPlatinum = ovr22:0202 (amount).
func (e *Engine) addPlatinum(v uint16) {
	c := e.selected()
	var room uint16
	if e.overweight(c, v, &room) {
		e.BottomMessage("Overloaded.  Money will be put in Pool.")
		c.setCoins(coinPlatinum, c.coins(coinPlatinum)+room)
		addWeight(c, room)
		e.money[coinPlatinum] += uint32(v) - uint32(room)
	} else {
		c.setCoins(coinPlatinum, c.coins(coinPlatinum)+v)
		addWeight(c, v)
	}
}

// inputAmount = ovr22:0294 (max, prompt, colour): digits on row 24 up to
// max (more shows max); Enter takes it, Esc gives 0.
func (e *Engine) inputAmount(max uint16, prompt string, colour int) uint16 {
	prompt = cut(prompt, 0x28)
	e.DisplaySpaces(0, 0x18, 0, 0x28)
	e.DisplayString(0, 0x18, colour, prompt)
	x0 := len(prompt)
	x := x0
	maxS := strconv.Itoa(int(max))
	s := ""
	var k byte
	for k != 0xD && k != 0x1B {
		k = e.ReadKeyByte()
		switch {
		case k >= '0' && k <= '9':
			s = cut(s+string(k), 6)
			v, _ := strconv.Atoi(s)
			if int(max) >= v {
				x++
			} else {
				s = cut(maxS, 6)
				x = x0 + len(s)
			}
			e.DisplayString(x0, 0x18, 0xF, s)
		case k == 8 && len(s) > 0:
			// original bug kept: blanks the cell at the cursor, not the char before it
			s = s[:len(s)-1]
			e.DisplaySpaces(x, 0x18, 0, 1)
			x--
		}
	}
	e.DisplaySpaces(0, 0x18, 0, 0x28)
	if k == 0x1B {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return uint16(v)
}

// giveMoney = ovr22:044C (from, to, amount, coin). The original adds the
// weight back to the giver instead of the receiver; ApplyItems sets both
// right again.
func (e *Engine) giveMoney(from, to *Character, amount uint16, coin int) {
	if wordAt(to.Raw[:], chWeight)+amount > e.maxWeight(to) {
		e.BottomMessage("Overloaded")
		return
	}
	from.setCoins(coin, from.coins(coin)-amount)
	subWeight(from, amount)
	to.setCoins(coin, to.coins(coin)+amount)
	addWeight(from, amount)
}

// sharer (ovr22:05E0 counts them): members whose share counts (+0x84 is 0 or 0xB3).
func sharer(c *Character) bool { return c.Raw[chPlayerFlag] == 0 || c.Raw[chPlayerFlag] == 0xB3 }

// PoolMoney = ovr22:0508: everyone's coins into the pool.
func (e *Engine) PoolMoney() {
	e.poolHasMoney = true
	for _, c := range e.party {
		if !sharer(c) {
			continue
		}
		for i := 0; i <= 6; i++ {
			e.money[i] += uint32(c.coins(i))
			subWeight(c, c.coins(i))
		}
		for i := 0; i <= 6; i++ {
			c.setCoins(i, 0)
		}
	}
}

// ShareMoney = ovr22:063A: the pool split evenly, the remainder handed
// out one coin at a time, then whatever still fits.
func (e *Engine) ShareMoney() {
	n := 0
	for _, c := range e.party {
		if sharer(c) {
			n++
		}
	}
	var share, rest [7]uint16
	for i := 0; i <= 6; i++ {
		if int32(e.money[i]) > 0 && n > 0 {
			share[i] = uint16(int32(e.money[i]) / int32(n))
			rest[i] = uint16(int32(e.money[i]) % int32(n))
		}
	}
	var room uint16
	for _, c := range e.party {
		if c.Raw[chPlayerFlag] >= 0x80 {
			continue
		}
		for i := 6; i >= 0; i-- {
			if !e.overweight(c, share[i], &room) {
				c.setCoins(i, c.coins(i)+share[i])
				addWeight(c, share[i])
				if rest[i] > 0 && !e.overweight(c, 1, &room) {
					c.setCoins(i, c.coins(i)+1)
					addWeight(c, 1)
					rest[i]--
				}
			} else {
				c.setCoins(i, c.coins(i)+room)
				rest[i] = rest[i] + share[i] - room
				addWeight(c, room)
			}
		}
	}
	for i := 6; i >= 0; i-- {
		for _, c := range e.party {
			if rest[i] == 0 {
				break
			}
			space := e.maxWeight(c) - wordAt(c.Raw[:], chWeight)
			if space == 0 {
				continue
			}
			if rest[i] > space {
				c.setCoins(i, c.coins(i)+space)
				addWeight(c, space)
				rest[i] -= space
			} else {
				c.setCoins(i, c.coins(i)+rest[i])
				addWeight(c, rest[i])
				rest[i] = 0
			}
		}
	}
	e.poolHasMoney = false
	for i := 0; i <= 6; i++ {
		e.money[i] = uint32(rest[i])
		if e.money[i] != 0 {
			e.poolHasMoney = true
		}
	}
}

// dropMoney = ovr22:0A01 (char, amount, coin): in a shop or the pool
// screen the coins go to the pool, otherwise they are gone.
func (e *Engine) dropMoney(c *Character, amount uint16, coin int) {
	c.setCoins(coin, c.coins(coin)-amount)
	subWeight(c, amount)
	if e.beginFlag == 6 || e.beginFlag == 1 {
		e.money[coin] += uint32(amount)
	}
}

// takeMoney = ovr22:0A88 (char, amount, coin): from the pool.
func (e *Engine) takeMoney(c *Character, amount uint16, coin int) {
	var room uint16
	if e.overweight(c, amount, &room) {
		e.BottomMessage("Overloaded")
		return
	}
	if int32(amount) > int32(e.money[coin]) {
		amount = uint16(e.money[coin])
	}
	e.money[coin] -= uint32(amount)
	c.setCoins(coin, c.coins(coin)+amount)
	addWeight(c, amount)
}

// coinFromText = ovr22:0B93 (text, name): the coin a list line names.
func coinFromText(text string) (int, string) {
	i := 0
	for i < len(text) && text[i] == ' ' {
		i++
	}
	if i >= len(text) {
		return 0, ""
	}
	switch text[i] {
	case 'G':
		if i+1 < len(text) && upcase(text[i+1]) == 'E' {
			return 5, "Gems "
		}
		return 3, "Gold "
	case 'P':
		return 4, "Platinum "
	case 'E':
		return 2, "electrum "
	case 'S':
		return 1, "silver "
	case 'C':
		return 0, "Copper "
	case 'J':
		return 6, "Jewelry "
	}
	return 0, "" // unset in the original
}

func upcase(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

// moneyName = DS:0FFA + 0xB*i.
func (e *Engine) moneyName(i int) string { return e.X.PString(0xFFA + 0xB*i) }

// TakeMoneyMenu = ovr22:0CEB: pick coins out of the pool.
func (e *Engine) TakeMoneyMenu() {
	e.DrawBoxTitled(1, 1, 0x26, 0x16, 0, 0xA, "")
	index := 0
	for done := false; !done; {
		redraw := true
		var list []*ListNode
		for i := 6; i >= 0; i-- {
			if int32(e.money[i]) > 0 {
				s := cut(e.moneyName(i)+" "+strconv.Itoa(int(int32(e.money[i]))), 0x28)
				list = append([]*ListNode{{Text: s}}, list...)
			}
		}
		var sel *ListNode
		k := e.ListMenu("Select type of coin", "", 0xD, 0xA, 0xF, 2, 2, 0xF, 8, list, true, &redraw, &index, &sel)
		if sel == nil || k == 0 {
			break
		}
		coin, name := coinFromText(sel.Text)
		amount := e.inputAmount(uint16(e.money[coin]), cut("How much "+name+"will you take? ", 0xFF), 0xA)
		e.takeMoney(e.selected(), amount, coin)
		e.poolHasMoney = false
		done = true
		for i := 0; i <= 6; i++ {
			if int32(e.money[i]) > 0 {
				e.poolHasMoney = true
				done = false
			}
		}
	}
}

// treasureFlags = ovr22:0F7E: is there money / are there items to take.
func (e *Engine) treasureFlags() (money, items bool) {
	for i := 0; i <= 6; i++ {
		if e.money[i] != 0 {
			money = true
		}
	}
	return money, len(e.run.treasure) > 0
}

// randomItemClass = ovr22:0FD2: d20 1-14 → 1, 15-19 → 2, 20 → 3.
func (e *Engine) randomItemClass() int {
	switch r := e.Dice(1, 20); {
	case r >= 1 && r <= 14:
		return 1
	case r >= 15 && r <= 19:
		return 2
	}
	return 3
}
