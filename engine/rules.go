package engine

// Rules: affects, to-hit rolls, saving throws, damage and healing, ported
// from ovr25 and the START helpers it leans on (00BA:16EC ... 2637).

// CombatInfo is the 0x16-byte block at char+0x108 (allocated by ovr11).
type CombatInfo struct {
	Spell      byte       // +0x00 spell being cast (lost when hit)
	CanCast    byte       // +0x01
	CanUse     byte       // +0x02 may use an item
	Delay      byte       // +0x03 signed initiative; 0x14 = delayed
	AttackSlot byte       // +0x04 attack slot in use (1, 2)
	Sweeps     byte       // +0x05 sweeps left against weak foes
	Moves      byte       // +0x06 half squares left
	Guarding   byte       // +0x07
	Attacked   byte       // +0x08 has attacked this turn
	Facing     byte       // +0x09 0..7
	Target     *Character // +0x0A
	Dying      byte       // +0x0E rounds of bleeding left
	Attackers  byte       // +0x0F times attacked this round
	Turned     byte       // +0x10 turned by a cleric
	TurnUsed   byte       // +0x11 has tried to turn undead
	Flank      byte       // +0x12 accumulated turn from attackers
	Extra      byte       // +0x13 not one of the saved party
	Fleeing    byte       // +0x14
	Pattern    byte       // +0x15 AI move pattern, row of DS:02AE
}

// Rule globals (DS:6816..682A).
type ruleVars struct {
	save     byte    // DS:6816 saving throw total
	affVal   byte    // DS:6817 affect being added (0 = blocked)
	damage   byte    // DS:6818
	dmgType  byte    // DS:6819 1 fire 2 cold 4 electricity 8 magic 0x10 acid
	diceN    byte    // DS:681C
	hitRoll  int8    // DS:6822
	saveType byte    // DS:682A
	cure     byte    // DS:681F set while the temple removes affects
	side     [2]int8 // DS:6814 standing members per side
}

// AffectHandler is one entry of the DS:6828 table (filled by ovr13:31A2):
// handler(char, affect, flag); flag 1 = the affect is being removed.
type AffectHandler func(e *Engine, c *Character, a *Affect, flag byte)

var affectHandlers [0x8C]AffectHandler

// callAffect = ovr25:0000 (type, char, affect, flag).
func (e *Engine) callAffect(typ byte, c *Character, a *Affect, flag byte) {
	if int(typ) < len(affectHandlers) && affectHandlers[typ] != nil {
		affectHandlers[typ](e, c, a, flag)
	}
}

// FindAffect = 00BA:2199 (out, type, char): first affect of the type.
func (e *Engine) FindAffect(c *Character, typ byte) *Affect {
	for _, a := range c.Affects {
		if a.Type == typ {
			return a
		}
	}
	return nil
}

// RemoveAffect = ovr25:0026 (char, type, affect): affect nil means the first
// one of the type. Its handler runs first when the node asks for it.
func (e *Engine) RemoveAffect(c *Character, typ byte, a *Affect) {
	if a == nil {
		a = e.FindAffect(c, typ)
	}
	if a == nil {
		return
	}
	if a.Call != 0 {
		e.callAffect(typ, c, a, 1)
	}
	for i, x := range c.Affects {
		if x == a {
			c.Affects = append(c.Affects[:i], c.Affects[i+1:]...)
			return
		}
	}
}

// RemoveEffect is the old name of RemoveAffect used by the clock code.
func (e *Engine) RemoveEffect(c *Character, a *Affect) { e.RemoveAffect(c, a.Type, a) }

// AddAffect = ovr25:0E52 (char, type, time, data, call): append a node.
func (e *Engine) AddAffect(c *Character, typ byte, time int, data, call byte) {
	a := &Affect{affectRecord: affectRecord{Type: typ, Time: uint16(time), Data: data, Call: call}}
	c.Affects = append(c.Affects, a)
}

// auraAffects = ovr25 cs:0140: affects that also work on everyone near the
// one who has them (prayer, chant ...).
func auraAffect(t byte) bool { return t == 0x15 || t == 0x2D || t == 0x2E || t == 0x31 }

// checkAffect = ovr25:0160 (char, type): run the handler (flag 0) when the
// character has the affect, or for an aura affect when anyone in range
// (anyone at all outside combat) has it.
func (e *Engine) checkAffect(c *Character, typ byte) {
	a := e.FindAffect(c, typ)
	found := a != nil
	if !found && auraAffect(typ) {
		saved := e.nearList // DS:671B, 0xD8 bytes
		for _, p := range e.party {
			if found {
				break
			}
			pa := e.FindAffect(p, typ)
			if pa == nil {
				continue
			}
			a = pa
			if e.beginFlag != 5 {
				found = true
				continue
			}
			r := 1
			if typ == 0x31 {
				r = 6
			}
			e.nearCombatants(p, e.combatX(p), e.combatY(p), r)
			me := e.combatIndex(c)
			for _, n := range e.near.list {
				if int(n.index) == me {
					found = true
				}
			}
		}
		e.nearList = saved
	}
	if found {
		e.callAffect(typ, c, a, 0)
	}
}

// affectGroups = the lists of ovr25:02FA: which affects take part in each
// kind of check (the group number is the caller's).
var affectGroups = [][]byte{
	0x00: {0x7E},
	0x01: {0x3F, 0x25, 0x19, 0x47},
	0x02: {0x55, 0x56, 0x57, 0x44, 0x4F, 0x50, 0x4C, 0x49, 0x4D},
	0x03: {0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x4F, 0x55, 0x56, 0x57},
	0x04: {0x1D, 0x03, 0x06},
	0x05: {0x1C, 0x29, 0x68, 0x78, 0x65, 0x73, 0x74, 0x77, 0x7B, 0x60, 0x5E, 0x3C, 0x7A, 0x75},
	0x06: {0x71, 0x3D, 0x7A, 0x3C, 0x5B, 0x0A, 0x14, 0x69, 0x6A, 0x70, 0x72, 0x76, 0x11, 0x5D, 0x65, 0x1C},
	0x07: {0x33, 0x34, 0x35, 0x1F},
	0x08: {0x63, 0x51, 0x52, 0x59, 0x48, 0x38},
	0x09: {0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F, 0x70, 0x7C, 0x7D},
	0x0A: {0x01, 0x02, 0x21, 0x24, 0x31, 0x03, 0x06, 0x12, 0x1A},
	0x0B: {0x21, 0x11, 0x08, 0x09, 0x2D, 0x2E, 0x1E},
	0x0C: {0x08, 0x09, 0x0A, 0x11, 0x14, 0x21, 0x24, 0x2D, 0x2E, 0x31, 0x3D, 0x6F, 0x7D, 0x5A, 0x61},
	0x0D: {0x63, 0x64, 0x67, 0x4B, 0x4A},
	0x0E: {0x53, 0x54, 0x58, 0x79},
	0x0F: {0x15, 0x1E, 0x4A, 0x4B, 0x0B},
	0x10: {0x19, 0x47, 0x25, 0x2F, 0x30, 0x59},
	0x11: {0x01, 0x02, 0x0B},
	0x12: {0x27, 0x2A, 0x3A},
	0x13: {0x62, 0x17, 0x48, 0x38, 0x0B},
}

// Affect check groups by use.
const (
	grpStatus    = 0x00
	grpDamage    = 0x06
	grpAddAffect = 0x09
	grpAttacker  = 0x0A
	grpSave      = 0x0C
	grpDown      = 0x0D
	grpDefender  = 0x10
)

// affectGroup = ovr25:02FA (char, group).
func (e *Engine) affectGroup(c *Character, group int) {
	if group < len(affectGroups) {
		for _, t := range affectGroups[group] {
			e.checkAffect(c, t)
		}
	}
}

// stinkingCloud = ovr25:0AE7 (char): a character standing in a stinking
// cloud coughs (save) or is overcome by nausea.
func (e *Engine) stinkingCloud(c *Character) {
	if c.Okay == 0 {
		return
	}
	_, ground := e.groundAt(c, 8)
	if ground != 0x1E {
		return
	}
	for _, t := range []byte{0x1E, 0x1F, 0x20, 0x6F, 0x7D} {
		if e.FindAffect(c, t) != nil {
			return
		}
	}
	saved := e.SavingThrow(c, 0, 0)
	var t byte
	if saved {
		t = 0x1E
		e.AddAffectMsg(c, t, 1, 0xFF, 0, 0, saved, "starts to cough")
	} else {
		t = 0x1F
		e.AddAffectMsg(c, t, e.Dice(1, 4)+1, 0xFF, 0, 0, saved, "chokes and gags from nausea")
	}
	// ovr25 passes the out pointer of its last FindAffect (0x7D), which is nil.
	e.callAffect(t, c, nil, 0)
}

// RollHit = ovr25:0C4F (char, bonus): d20 (a natural 20 counts 100) plus
// bonus must beat the defender's armour value at +0x111.
func (e *Engine) RollHit(c *Character, bonus int) bool {
	e.rv.hitRoll = int8(e.Dice(1, 20))
	if e.rv.hitRoll <= 1 {
		return false
	}
	if e.rv.hitRoll == 20 {
		e.rv.hitRoll = 100
	}
	e.affectGroup(c, grpDefender)
	return e.rv.hitRoll >= 0 && int(e.rv.hitRoll)+int(byte(bonus)) > int(c.AC[0])
}

// AttackHits = ovr25:0CB7 (attacker, defender, need): attacking ends
// invisibility; d20 + THAC0 + the side's modifier must reach need.
func (e *Engine) AttackHits(att, def *Character, need int) bool {
	e.removeInvisibility(att)
	e.rv.hitRoll = int8(e.Dice(1, 20))
	if e.rv.hitRoll <= 1 {
		return false
	}
	if e.rv.hitRoll == 20 {
		e.rv.hitRoll = 100
	}
	e.affectGroup(att, grpAttacker)
	e.affectGroup(def, grpDefender)
	adj := int(int8(e.game[0x6E2]))
	if att.Side != 0 {
		adj = int(int8(e.game[0x6E0]))
	}
	return e.rv.hitRoll >= 0 && int(e.rv.hitRoll)+int(att.THAC0)+adj >= int(byte(need))
}

// SavingThrow = ovr25:0D63 (char, type, bonus): 1 always fails, 20 always
// saves, otherwise d20 + bonus + the character's save bonus must reach the
// target for the type (0 paralysis ... 4 spells).
func (e *Engine) SavingThrow(c *Character, typ, bonus int) bool {
	e.rv.save = byte(e.Dice(1, 20))
	switch e.rv.save {
	case 1:
		return false
	case 20:
		return true
	}
	e.rv.save = byte(int(e.rv.save) + int(int8(c.SaveBonus)) + int(int8(bonus)))
	e.rv.saveType = byte(typ)
	e.affectGroup(c, grpSave)
	return c.Saves[int(int8(typ))] <= e.rv.save
}

// RollDice = ovr25:0E2E (n, sides): Dice that also remembers n (DS:681C).
func (e *Engine) RollDice(n, sides int) int {
	e.rv.diceN = byte(n)
	return e.Dice(n, sides)
}

// KillCharacter = ovr25:0F06 (char, status, message).
func (e *Engine) KillCharacter(c *Character, status byte, msg string) {
	msg = cut(msg, 0x28)
	if c.Okay == 0 {
		return
	}
	idx := e.combatIndex(c)
	e.combatRedrawChar(c, 3, 0)
	e.CharMessage(c, msg, 0xA, true)
	c.Okay = 0
	c.Status = status
	if c.Status != 3 {
		c.HP = 0
	}
	e.combatClearIcon(0, 0, idx)
	e.Flip()
	e.cpos[idx].size = 0 // DS:5F2A + 4*idx
	e.combatRedrawMap()
	if e.clearCombatInfo(c) {
		e.removeCombatAffects(c)
	}
}

// removeInvisibility = ovr25:0FD2.
func (e *Engine) removeInvisibility(c *Character) {
	for a := e.FindAffect(c, 0x19); a != nil; a = e.FindAffect(c, 0x19) {
		e.RemoveAffect(c, 0x19, a)
	}
}

// combatAffects = DS:0C14: affects that end with the fight (or a death).
var combatAffects = [16]byte{0x07, 0x0B, 0x1E, 0x1F, 0x20, 0x33, 0x34, 0x35, 0x36, 0x3A, 0x3B, 0x5F, 0x62, 0x89, 0x4A, 0x4B}

// removeCombatAffects = ovr25:100A.
func (e *Engine) removeCombatAffects(c *Character) {
	for _, t := range combatAffects {
		e.RemoveAffect(c, t, nil)
	}
}

// removeAffects4A = ovr25:1041 (DS:0C24: 0x4A, 0x4B).
func (e *Engine) removeAffects4A(c *Character) {
	for _, t := range []byte{0x4A, 0x4B} {
		e.RemoveAffect(c, t, nil)
	}
}

// CureAffect = ovr25:1081 (char, type): "is Cured" and the affect goes.
func (e *Engine) CureAffect(c *Character, typ byte) bool {
	a := e.FindAffect(c, typ)
	if a == nil {
		return false
	}
	e.CharMessage(c, "is Cured", 0xA, true)
	e.RemoveAffect(c, typ, a)
	return true
}

// strValue = ovr25:10E4 (str, str00): one byte for strength, 18/xx as
// xx+1 (18/00 = 101), others +100.
func strValue(str, str00 byte) byte {
	if str == 18 {
		return str00 + 1
	}
	return str + 0x64
}

// strDecode = ovr25:110D: the inverse, from an affect's saved strength.
func strDecode(a *Affect) (str, str00 byte) {
	str = a.Data & 0x7F
	if str <= 0x65 {
		return 18, str - 1
	}
	return str - 0x64, 0
}

// SetStrength = ovr25:115C (char, str, str00, out): magical strength
// (affects 0x0C / 0x26 remember the natural value). Returns false, with
// the request encoded, when it would lower the current strength.
func (e *Engine) SetStrength(c *Character, str, str00 byte) (byte, bool) {
	cur, cur00 := c.Stats[statStr], c.Str00
	if str < cur || str == 18 && str00 < cur00 {
		return strValue(str, str00) + 0x80, false
	}
	var keep *Affect
	for _, a := range c.Affects {
		if keep != nil {
			break
		}
		if (a.Type == 0x0C || a.Type == 0x26) && a.Data < 0x80 {
			keep = a
		}
	}
	if keep != nil {
		v := strValue(cur, cur00)
		cur, cur00 = strDecode(keep)
		keep.Data = v + 0x80
	}
	c.Stats[statStr] = str
	c.Str00 = str00
	e.ApplyItems(c)
	return strValue(cur, cur00), true
}

// DamageMode values for Damage (what a successful save does).
const (
	saveNone   = 0
	saveNegate = 1
	saveHalf   = 2
)

// Damage = ovr25:1344 (char, damage, mode, saved): reports and applies
// damage, the lost spell, and going down.
func (e *Engine) Damage(c *Character, dmg int, mode byte, saved bool) {
	e.rv.damage = byte(dmg)
	e.affectGroup(c, grpDamage)
	if saved {
		switch mode {
		case saveNegate:
			e.rv.damage = 0
		case saveHalf:
			e.rv.damage /= 2
		}
	}
	if e.rv.damage == 0 {
		return
	}
	var msg string
	if e.rv.damage != 1 {
		msg = "takes " + itoa(int(e.rv.damage)) + " points of damage "
	} else {
		msg = "takes 1 point of damage "
	}
	switch e.rv.dmgType & 0xF7 {
	case 1:
		msg += "from Fire"
	case 2:
		msg += "from Cold"
	case 4:
		msg += "from Electricity"
	case 0x10:
		msg += "from Acid"
	}
	if e.rv.dmgType&8 == e.rv.dmgType {
		msg += "from Magic"
	}
	e.hitMessage(c, false, msg)
	e.ApplyDamage(c, int(e.rv.damage))
	if e.beginFlag == 5 && c.Combat != nil {
		c.Combat.CanCast = 0
		if c.Combat.Spell > 0 {
			e.CharMessage(c, "lost a spell", 0xC, true)
			e.forgetSpell(c.Combat.Spell)
			c.Combat.Spell = 0
		}
	}
	if c.Okay == 0 {
		msg = "Goes Down"
		if c.Status == 5 {
			msg += ", and is Dying"
		}
		if s := c.Status; s >= 6 && s <= 8 {
			msg = "is killed"
		}
		e.CharMessage(c, msg, e.msgRow+1, false)
		if e.beginFlag != 5 {
			e.MsgDelay()
		} else {
			e.removeCombatAffects(c)
			e.affectGroup(c, grpDown)
			if c.Okay == 0 {
				e.combatRemoveFromMap(c)
			} else {
				e.MsgDelay()
			}
		}
	}
	e.ClearMessageArea()
}

// AddAffectMsg = ovr25:1656 (char, type, time, data, call, saveMode, saved,
// message): adds an affect unless something blocks it (or a save negates
// it); a shorter or permanent copy is replaced.
func (e *Engine) AddAffectMsg(c *Character, typ byte, time int, data, call, saveMode byte, saved bool, msg string) {
	msg = cut(msg, 0x28)
	e.rv.affVal = typ
	e.affectGroup(c, grpAddAffect)
	if e.rv.affVal == 0 || saved && saveMode == 1 {
		e.CharMessage(c, "is Unaffected", 0xA, true)
		return
	}
	if a := e.FindAffect(c, typ); a != nil {
		if a.Time > 0 && time > int(a.Time) || time == 0 {
			e.RemoveAffect(c, typ, a)
		}
	}
	e.AddAffect(c, typ, time, data, call)
	if msg != "" {
		e.hitMessage(c, true, msg)
		e.ClearMessageArea()
	}
}

// Heal = ovr25:175D (char, amount, magical): only the living (okay,
// animated, unconscious, dying) heal; affect 0x32 blocks natural healing.
func (e *Engine) Heal(c *Character, amount int, magical bool) bool {
	switch c.Status {
	case 0, 1, 4, 5:
	default:
		return false
	}
	if !(c.HP < c.MaxHP && magical) {
		if e.FindAffect(c, 0x32) != nil || magical {
			return false
		}
	}
	c.HP += byte(amount)
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	if c.Okay == 0 {
		if c.Status == 5 {
			c.Status = 4
		}
		if c.Status == 4 && e.beginFlag != 5 {
			e.callAffect(0x4E, c, nil, 1)
		}
	}
	return true
}

// Revive = ovr25:1869 (char, hp): back on its feet if its square is free.
func (e *Engine) Revive(c *Character, hp byte) bool {
	if !e.combatSquareFree(c, e.combatX(c), e.combatY(c), true) {
		return false
	}
	c.Status = 0
	c.Okay = 1
	c.HP = hp
	if e.beginFlag == 5 {
		e.combatRedrawChar(c, 3, 0)
	}
	msg := "gets back up"
	if c.Side == 1 {
		msg = "stands up and grins"
	}
	e.CharMessage(c, msg, 0xA, true)
	e.countSides()
	return true
}

// CharMessage = 00BA:16EC (char, message, row, wait): the name and a
// message in the combat side panel, or in the box under the view.
func (e *Engine) CharMessage(c *Character, msg string, row int, wait bool) {
	if e.beginFlag == 5 {
		e.combatSay(c, msg, true) // extra: under the view, not in the side pane
	} else {
		r := 0x11
		if e.resting != 0 {
			r = 0x12
		} else {
			e.DrawBorder(1, r, 0x26, 0x16, 0)
		}
		e.ClearCells(1, r, 0x26, 0x16)
		e.displayName(c, 1, r+1, false)
		e.DisplayTextBlock(1, r+2, 0x26, 0x16, 0xA, true, msg)
	}
	if wait {
		e.MsgDelay()
		e.ClearMessageArea()
	}
}

// ClearMessageArea = 00BA:17E3.
func (e *Engine) ClearMessageArea() {
	if e.beginFlag == 5 {
		e.ClearCells(0, 0x17, 0x27, 0x18)
		if p := e.ext.panel; p != nil {
			e.combatPanel(p)
		}
	} else {
		e.ClearCells(1, 0x12, 0x26, 0x16)
	}
}

// hitMessage = 00BA:1FEB (char, magical, message): in combat the target
// flashes (once per message delay step for magic) with a sound.
func (e *Engine) hitMessage(c *Character, magical bool, msg string) {
	msg = cut(msg, 0x28)
	if e.beginFlag != 5 {
		e.CharMessage(c, msg, 0xA, true)
		return
	}
	e.soundWait(0)
	sel := 0x17
	if magical {
		sel = 0x16
	}
	e.hitColours(sel)
	if !e.combatOnScreen(c, true) {
		e.combatCentre(e.combatX(c), e.combatY(c), 3, 8)
	}
	if magical {
		e.PlaySound(sndMagicHit)
	} else {
		e.PlaySound(sndHit)
	}
	e.CharMessage(c, msg, 0xA, false)
	n := 0
	if magical {
		n = int(e.msgDelay)
	}
	idx := e.combatIndex(c)
	px, py := int(3*e.combatScreenX[idx]), int(3*e.combatScreenY[idx]) // byte products
	for i := 0; i <= n; i++ {
		for f := 0; f <= 3; f++ {
			e.drawHitFrame(f, px, py)
			e.Flip()
			e.Delay(0x46)
			e.restoreHitFrame(px, py)
		}
	}
	e.soundWait(0xA)
	e.Flip()
	if n == 0 {
		e.MsgDelay()
	}
}

// ApplyDamage = 00BA:2228 (char, damage): hit points down to 0, then
// unconscious / dying (bleeding rounds) / dead at 10 below.
func (e *Engine) ApplyDamage(c *Character, dmg int) {
	hp := int(c.HP)
	over, left := 0, 0
	if hp >= dmg {
		left = hp - dmg
	} else {
		over = dmg - hp
	}
	over &= 0xFF
	switch {
	case over > 9 || left == 0 && c.Status == 1:
		c.Status = 6
	case over > 0:
		c.Status = 5
		if e.beginFlag == 5 && c.Combat != nil {
			c.Combat.Dying = byte(over)
		}
	case left == 0:
		c.Status = 4
	}
	if s := c.Status; s == 0 || s == 1 {
		c.HP = byte(left)
		return
	}
	c.Okay = 0
	c.HP = 0
	if e.beginFlag == 5 {
		e.rv.side[int8(c.Side)&1]--
		if c.Combat != nil {
			c.Combat.Delay = 0
		}
	}
}

// forgetSpell = 00BA:148B (spell): the selected character's first
// memorised copy of the spell is gone.
func (e *Engine) forgetSpell(sp byte) {
	c := e.selected()
	if c == nil {
		return
	}
	for i := 0; i <= 0x14; i++ {
		if c.Memorized[i] == sp {
			c.Memorized[i] = 0
			break
		}
	}
}

// healedMessage = 00BA:2348 (char).
func (e *Engine) healedMessage(c *Character) {
	msg := "is partially healed"
	if c.HP == c.MaxHP {
		msg = "is fully healed"
	}
	e.CharMessage(c, msg, 0xA, true)
	if e.beginFlag != 5 {
		e.DisplayPartyList()
	}
}

// countSides = 00BA:23DB: standing members of each side (DS:6814/6815).
func (e *Engine) countSides() {
	e.rv.side = [2]int8{}
	for _, c := range e.party {
		if c.Okay != 0 {
			e.rv.side[int8(c.Side)&1]++
		}
	}
}

// clearCombatInfo = 00BA:2637 (char): reset the combat block; always true.
func (e *Engine) clearCombatInfo(c *Character) bool {
	if c.Combat != nil {
		ci := c.Combat
		ci.Delay, ci.Spell, ci.Guarding, ci.Moves = 0, 0, 0, 0
	}
	return true
}
