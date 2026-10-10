package engine

import "strconv"

// Character is the 0x11D-byte DOS character record, kept raw so the .SAV
// files round-trip byte for byte. Offsets are from the disassembly.
type Character struct {
	Raw     [0x11D]byte
	Items   []*Item     // list at +0xC8 (0x3F-byte nodes)
	Affects []*Affect   // list at +0x7F (9-byte nodes)
	Combat  *CombatInfo // block at +0x108 while fighting
	Slots   [13]*Item   // readied items, +0xCC (set by ApplyItems)
}

const (
	chName   = 0x000 // pascal string, max 15
	chMaxHP  = 0x032
	chStatus = 0x10C // index into DS:1047 names
	chOkay   = 0x10D // non-zero = active (status display skipped)
	chNPC    = 0x10E // 1 = name drawn yellow
	chAC60   = 0x111 // 60 - armour class
	chHP     = 0x11B
)

func (c *Character) Name() string {
	n := int(c.Raw[chName])
	if n > 15 {
		n = 15
	}
	return string(c.Raw[chName+1 : chName+1+n])
}

// SetName stores a pascal string (max 15).
func (c *Character) SetName(s string) {
	if len(s) > 15 {
		s = s[:15]
	}
	c.Raw[chName] = byte(len(s))
	copy(c.Raw[chName+1:chName+16], s)
	for i := len(s); i < 15; i++ {
		c.Raw[chName+1+i] = 0
	}
}

// IntToStr = 00BA:1037.
func itoa(n int) string { return strconv.Itoa(n) }

// displayAC = 00BA:08EF (char, x, y): |ac60-60|, with "-" when ac60 > 60, colour 0xA.
func (e *Engine) displayAC(c *Character, x, y int) {
	v := int(c.Raw[chAC60])
	s := itoa(abs(v - 60))
	if v > 60 {
		s = "-" + s
	}
	e.DisplayString(x, y, 0xA, s)
}

// displayHP = 00BA:0973 (char, x, y, flag): yellow when below max, white
// otherwise, magenta when flag set.
func (e *Engine) displayHP(c *Character, x, y int, flag bool) {
	col := 0xA
	if c.Raw[chHP] < c.Raw[chMaxHP] {
		col = 0xE
	}
	if flag {
		col = 0xD
	}
	e.DisplayString(x, y, col, itoa(int(c.Raw[chHP])))
}

// displayName = 00BA:1816 (char, x, y, possessive): red when not okay,
// yellow for NPCs, cyan otherwise; optional "'s" suffix.
func (e *Engine) displayName(c *Character, x, y int, possessive bool) {
	col := 0xC
	if c.Raw[chOkay] != 0 {
		col = 0xB
		if c.Raw[chNPC] == 1 {
			col = 0xE
		}
	}
	e.DisplayString(x, y, col, c.Name())
	if possessive {
		e.DisplayString(x+len(c.Name()), y, col, "'s")
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
