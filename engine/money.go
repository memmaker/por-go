package engine

import "encoding/binary"

// The party's money. The original gave every character a purse of seven
// coin kinds plus a treasure pool (ovr22); here the party shares one purse
// and every coin is counted in gold pieces.

// purse is the party's money: gold pieces, and gems and jewelry to appraise.
type purse struct{ gold, gems, jewelry int }

func (p *purse) add(q purse) {
	p.gold += q.gold
	p.gems += q.gems
	p.jewelry += q.jewelry
}

func (p purse) empty() bool { return p == purse{} }

// coinCopper is the worth of copper, silver, electrum, gold and platinum
// in copper pieces (DS:0D24); 200 copper make a gold piece (DS:0D30).
var coinCopper = [5]int{1, 10, 100, 200, 1000}

// coinsToPurse counts coins of the seven kinds (the last two gems and
// jewelry) as gold, rounded to the nearest piece.
func coinsToPurse(c [7]uint16) purse {
	var t int
	for i, v := range coinCopper {
		t += int(c[i]) * v
	}
	return purse{gold: (t + 100) / 200, gems: int(c[5]), jewelry: int(c[6])}
}

// takeRecordCoins moves the coins in a character record (starting gold,
// money in an old save) into the party's purse.
func (e *Engine) takeRecordCoins(c *Character) {
	var coins [7]uint16
	for i := range coins {
		coins[i] = c.Coins[i]
		c.Coins[i] = 0
	}
	e.purse.add(coinsToPurse(coins))
}

// pay takes gold from the purse; false (and a message) when there is not enough.
func (e *Engine) pay(gold int) bool {
	if gold > e.purse.gold {
		e.BottomMessage("Not enough money.")
		return false
	}
	e.purse.gold -= gold
	return true
}

// purseCoin is what ECL scripts see in the selected character's coin words
// (DS:6BBB.. copper to platinum): the purse as platinum and gold.
func (e *Engine) purseCoin(i int) uint16 {
	switch i {
	case 3:
		return uint16(e.purse.gold % 5)
	case 4:
		return uint16(min(e.purse.gold/5, 0xFFFF))
	}
	return 0
}

// setPurseCoin is a script setting one of those coin words.
func (e *Engine) setPurseCoin(i int, v uint16) {
	var c [7]uint16
	for k := 0; k <= 4; k++ {
		c[k] = e.purseCoin(k)
	}
	c[i] = v
	e.purse = purse{gold: coinsToPurse(c).gold, gems: e.purse.gems, jewelry: e.purse.jewelry}
}

// appendPurse / readPurse keep the purse after the original save game data.
func appendPurse(b []byte, p purse) []byte {
	for _, v := range []int{p.gold, p.gems, p.jewelry} {
		b = binary.LittleEndian.AppendUint32(b, uint32(v))
	}
	return b
}

func readPurse(b []byte) purse {
	if len(b) < 12 {
		return purse{}
	}
	v := func(i int) int { return int(binary.LittleEndian.Uint32(b[4*i:])) }
	return purse{v(0), v(1), v(2)}
}

// showMoney = ovr20:050D: the party's purse on the character sheet.
func (e *Engine) showMoney() {
	e.ClearCells(0xC, 7, 0x1A, 0xE)
	y := 7
	for _, l := range []struct {
		name string
		n    int
	}{{"Gold", e.purse.gold}, {"Gems", e.purse.gems}, {"Jewelry", e.purse.jewelry}} {
		if l.n == 0 {
			continue
		}
		e.DisplayString(0x14-len(l.name), y, 0xA, l.name)
		e.DisplayString(0x15, y, 0xA, itoa(l.n))
		y++
	}
}

// shopMoney (extra) puts the party's gold under the shop's and temple's menus.
func (e *Engine) shopMoney() {
	e.ClearBox(1, 0x15, 0x26, 0x16)
	e.DisplayString(1, 0x15, 0xE, e.moneyLine())
}

func (e *Engine) moneyLine() string { return "Party gold " + itoa(e.purse.gold) }

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
