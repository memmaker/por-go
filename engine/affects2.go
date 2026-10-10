package engine

import "fmt"

// Affect handlers of GAME.OVR overlay 13 from 0x1C00 to the end, and the
// ovr13:31A2 routine that fills the DS:6828 handler table.

// o13_31A2 = ovr13:31A2: points each affect number at its handler. The
// original stores overlay thunk addresses; here the slots get the Go methods.
// Unlisted numbers keep no handler.
func (e *Engine) o13_31A2() {
	affectHandlers[0x01] = (*Engine).o13_010D
	affectHandlers[0x02] = (*Engine).o13_011D
	affectHandlers[0x03] = (*Engine).o13_013B
	affectHandlers[0x04] = (*Engine).o13_017F
	affectHandlers[0x05] = (*Engine).o13_319B
	affectHandlers[0x06] = (*Engine).o13_01C3
	affectHandlers[0x07] = (*Engine).o13_0233
	affectHandlers[0x08] = (*Engine).o13_0377
	affectHandlers[0x09] = (*Engine).o13_03AC
	affectHandlers[0x0A] = (*Engine).o13_03E1
	affectHandlers[0x0B] = (*Engine).o13_0400
	affectHandlers[0x0C] = (*Engine).o13_04BA
	affectHandlers[0x0D] = (*Engine).o13_319B
	affectHandlers[0x0E] = (*Engine).o13_05D6
	affectHandlers[0x0F] = (*Engine).o13_05EB
	affectHandlers[0x10] = (*Engine).o13_0647
	affectHandlers[0x11] = (*Engine).o13_064E
	affectHandlers[0x12] = (*Engine).o13_0679
	affectHandlers[0x13] = (*Engine).o13_319B
	affectHandlers[0x14] = (*Engine).o13_06E4
	affectHandlers[0x15] = (*Engine).o13_070F
	affectHandlers[0x16] = (*Engine).o13_0775
	affectHandlers[0x17] = (*Engine).o13_07E0
	affectHandlers[0x18] = (*Engine).o13_319B
	affectHandlers[0x19] = (*Engine).o13_0918
	affectHandlers[0x1A] = (*Engine).o13_0947
	affectHandlers[0x1B] = (*Engine).o13_319B
	affectHandlers[0x1C] = (*Engine).o13_09C0
	affectHandlers[0x1D] = (*Engine).o13_0A3D
	affectHandlers[0x1E] = (*Engine).o13_0A65
	affectHandlers[0x1F] = (*Engine).o13_133F
	affectHandlers[0x20] = (*Engine).o13_0B1B
	affectHandlers[0x21] = (*Engine).o13_0BAC
	affectHandlers[0x22] = (*Engine).o13_0BCF
	affectHandlers[0x23] = (*Engine).o13_0C06
	affectHandlers[0x24] = (*Engine).o13_0C15
	affectHandlers[0x25] = (*Engine).o13_0C26
	affectHandlers[0x26] = (*Engine).o13_04BA
	affectHandlers[0x27] = (*Engine).o13_0C4B
	affectHandlers[0x28] = (*Engine).o13_0CC3
	affectHandlers[0x29] = (*Engine).o13_1086
	affectHandlers[0x2A] = (*Engine).o13_10C1
	affectHandlers[0x2B] = (*Engine).o13_10DE
	affectHandlers[0x2C] = (*Engine).o13_1168
	affectHandlers[0x2D] = (*Engine).o13_0377
	affectHandlers[0x2E] = (*Engine).o13_03AC
	affectHandlers[0x2F] = (*Engine).o13_11F9
	affectHandlers[0x30] = (*Engine).o13_1276
	affectHandlers[0x31] = (*Engine).o13_12B2
	affectHandlers[0x32] = (*Engine).o13_12FB
	affectHandlers[0x33] = (*Engine).o13_133F
	affectHandlers[0x34] = (*Engine).o13_133F
	affectHandlers[0x35] = (*Engine).o13_133F
	affectHandlers[0x36] = (*Engine).o13_1355
	affectHandlers[0x37] = (*Engine).o13_135C
	affectHandlers[0x38] = (*Engine).o13_1363
	affectHandlers[0x39] = (*Engine).o13_1399
	affectHandlers[0x3A] = (*Engine).o13_1452
	affectHandlers[0x3B] = (*Engine).o13_1472
	affectHandlers[0x3C] = (*Engine).o13_319B
	affectHandlers[0x3D] = (*Engine).o13_1490
	affectHandlers[0x3E] = (*Engine).o13_14EB
	affectHandlers[0x3F] = (*Engine).o13_319B
	affectHandlers[0x40] = (*Engine).o13_165D
	affectHandlers[0x41] = (*Engine).o13_1671
	affectHandlers[0x42] = (*Engine).o13_1685
	affectHandlers[0x43] = (*Engine).o13_1699
	affectHandlers[0x44] = (*Engine).o13_16BB
	affectHandlers[0x45] = (*Engine).o13_16E6
	affectHandlers[0x46] = (*Engine).o13_170B
	affectHandlers[0x47] = (*Engine).o13_171F
	affectHandlers[0x48] = (*Engine).o13_1730
	affectHandlers[0x49] = (*Engine).o14_39D2
	affectHandlers[0x4A] = (*Engine).o14_3A86
	affectHandlers[0x4B] = (*Engine).o13_176F
	affectHandlers[0x4C] = (*Engine).o13_18B5
	affectHandlers[0x4D] = (*Engine).o13_1935
	affectHandlers[0x4E] = (*Engine).o13_19D9
	affectHandlers[0x4F] = (*Engine).o13_1A19
	affectHandlers[0x50] = (*Engine).o13_1A4C
	affectHandlers[0x51] = (*Engine).o13_1AAA
	affectHandlers[0x52] = (*Engine).o13_1BA8
	affectHandlers[0x53] = (*Engine).o13_1CAD
	affectHandlers[0x54] = (*Engine).o13_1E77
	affectHandlers[0x55] = (*Engine).o13_23A4
	affectHandlers[0x56] = (*Engine).o13_23C2
	affectHandlers[0x57] = (*Engine).o13_23EC
	affectHandlers[0x59] = (*Engine).o13_2490
	affectHandlers[0x5A] = (*Engine).o13_24D9
	affectHandlers[0x5B] = (*Engine).o13_2556
	affectHandlers[0x5C] = (*Engine).o13_319B
	affectHandlers[0x5D] = (*Engine).o13_25A2
	affectHandlers[0x5E] = (*Engine).o13_25BC
	affectHandlers[0x5F] = (*Engine).o13_2610
	affectHandlers[0x60] = (*Engine).o13_264B
	affectHandlers[0x61] = (*Engine).o13_268A
	affectHandlers[0x62] = (*Engine).o13_2700
	affectHandlers[0x63] = (*Engine).o13_2730
	affectHandlers[0x64] = (*Engine).o13_27D9
	affectHandlers[0x65] = (*Engine).o13_2814
	affectHandlers[0x66] = (*Engine).o13_2866
	affectHandlers[0x67] = (*Engine).o13_28BF
	affectHandlers[0x68] = (*Engine).o13_2901
	affectHandlers[0x69] = (*Engine).o13_2976
	affectHandlers[0x6A] = (*Engine).o13_2984
	affectHandlers[0x6B] = (*Engine).o13_2992
	affectHandlers[0x6C] = (*Engine).o13_29B6
	affectHandlers[0x6D] = (*Engine).o13_29CB
	affectHandlers[0x6E] = (*Engine).o13_29D9
	affectHandlers[0x6F] = (*Engine).o13_29F0
	affectHandlers[0x70] = (*Engine).o13_2A11
	affectHandlers[0x71] = (*Engine).o13_2A28
	affectHandlers[0x72] = (*Engine).o13_2A6D
	affectHandlers[0x73] = (*Engine).o13_2A87
	affectHandlers[0x74] = (*Engine).o13_2AE6
	affectHandlers[0x75] = (*Engine).o13_2B20
	affectHandlers[0x76] = (*Engine).o13_2B63
	affectHandlers[0x77] = (*Engine).o13_2B7D
	affectHandlers[0x78] = (*Engine).o13_2BC8
	affectHandlers[0x79] = (*Engine).o13_2C1C
	affectHandlers[0x7A] = (*Engine).o13_2D38
	affectHandlers[0x7B] = (*Engine).o13_2D88
	affectHandlers[0x7C] = (*Engine).o13_2DE4
	affectHandlers[0x7D] = (*Engine).o13_2E08
	affectHandlers[0x7E] = (*Engine).o13_2E37
	affectHandlers[0x7F] = (*Engine).o13_319B
	affectHandlers[0x80] = (*Engine).o13_2EDA
	affectHandlers[0x81] = (*Engine).o13_2EDA
	affectHandlers[0x82] = (*Engine).o13_2EDA
	affectHandlers[0x83] = (*Engine).o13_2F57
	affectHandlers[0x84] = (*Engine).o13_30A5
	affectHandlers[0x85] = (*Engine).o13_2EDA
	affectHandlers[0x86] = (*Engine).o13_2EDA
	affectHandlers[0x87] = (*Engine).o13_3134
	affectHandlers[0x88] = (*Engine).o13_2EDA
	affectHandlers[0x89] = (*Engine).o13_317A
	affectHandlers[0x8A] = (*Engine).o13_2EDA
	affectHandlers[0x8B] = (*Engine).o13_2EDA
}

// itemBaseFlags is DS:5589 (+7 of the 16-byte base item record) for the item's type.
func (e *Engine) itemBaseFlags(it *Item) byte {
	if i := int(it.Type)<<4 + 7; i < len(e.items) {
		return e.items[i]
	}
	return 0
}

// affectItem is the item standing in for the affect record (the original
// reads the item's bytes through the affect pointer).
func affectItem(a *Affect) *Item {
	if a.Item != nil {
		return a.Item
	}
	return &Item{}
}

// halveDamage is the shared "damage = damage div 2" of the resistance handlers.
func (e *Engine) halveDamage() { e.rv.damage >>= 1 }

// o13_1CAD (c, a, flag): a gaze that turns the target to stone; an item with
// 0x76 on a readied piece reflects it back. DS:6826/6828 (the target pointer
// the original keeps in globals) is a local here.
func (e *Engine) o13_1CAD(c *Character, a *Affect, flag byte) {
	t := c.Combat.Target
	e.CharMessage(c, "gazes...", 0xA, false)
	e.hitColours(0x12)
	e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), 4, 0x2D)
	if e.FindAffect(c, 0x7F) != nil {
		reflected := false
		for _, p := range t.Items {
			if reflected {
				break
			}
			if p.Readied != 0 && (p.Words[0] == 0x76 || p.Words[1] == 0x76 || p.Words[2] == 0x76) {
				e.CharMessage(t, "reflects it!", 0xC, false)
				e.flyMissile(e.combatX(t), e.combatY(t), e.combatX(c), e.combatY(c), 4, 0x2D)
				reflected = true
				t = c
			}
		}
	}
	if !e.SavingThrow(t, 1, 0) {
		e.killWithMessage(t, 7, "is Stoned")
	}
}

// o13_1E77 (c, a, flag): charm gaze; needs ovr14:1085 and a line of sight.
func (e *Engine) o13_1E77(c *Character, a *Affect, flag byte) {
	t := c.Combat.Target
	tx, ty := e.combatX(t), e.combatY(t)
	if !e.o14_1085(c, t) {
		return
	}
	rng := 0 // [bp-2] is never set in the original
	if !e.lineOfSight(&rng, &tx, &ty, e.combatX(c), e.combatY(c)) {
		return
	}
	e.CharMessage(c, "Gazes...", 0xA, false)
	e.hitColours(0x12)
	e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), 4, 0x2D)
	e.curSpell = 0xA
	data := byte(int(int8(c.Side))<<7 + 0xC)
	saved := e.SavingThrow(t, 4, 0xFE)
	e.AddAffectMsg(t, 0xB, 0, data, 1, 1, saved, "is charmed")
	if x := e.FindAffect(t, 0xB); x != nil {
		e.callAffect(0xB, t, x, 0)
	}
}

// drainLevels = ovr13:2122 (char, n): n levels are drained; hit points,
// class levels and experience follow. Its nested helper at 2002 is bestClass.
func (e *Engine) drainLevels(c *Character, n int) {
	var best uint32 // parent local [bp-0x31], also read after bestClass runs
	// bestClass = ovr13:2002: the class whose level has the highest experience
	// entry (DS:409B, 14 longs per class); sets best to that experience.
	bestClass := func() int {
		best = 0
		idx, maxLvl := 0, int8(0)
		for i := 0; i < 8; i++ {
			lvl := int8(c.Levels[i])
			var cur uint32
			if lvl > 1 {
				b := e.X.Bytes(0x409B+0x38*i+4*int(lvl), 4)
				cur = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
			}
			if lvl >= maxLvl && (int32(cur) >= int32(best) || cur == 0xFFFFFFFF) {
				best, idx, maxLvl = cur, i, lvl
			}
		}
		return idx
	}
	msg := "a level"
	if n > 1 {
		msg = cut(itoa(n)+" levels", 0x28)
	}
	e.CharMessage(c, "is drained "+msg, 0xA, true)
	if c.Status == 1 {
		e.killWithMessage(c, 8, "is dead forever")
		c.MaxHP = 0
		return
	}
	for lv := 1; lv <= n; lv++ {
		if c.Status == 8 {
			continue
		}
		c.Drained++
		var sum byte
		for i := 0; i < 8; i++ {
			sum += c.Levels[i]
		}
		per := 0
		if sum != 0 { // the original divides by zero (runtime error) here
			per = int(c.MaxHP) / int(sum)
		}
		e.debugValues("max,cur,lost", int(c.MaxHP), int(c.HP), int(c.DrainedHP))
		for k := 1; k <= per; k++ {
			if c.MaxHP > 0 {
				c.MaxHP--
			}
			if c.HPRoll > 0 {
				c.HPRoll--
			}
			if c.HP > 0 {
				c.HP--
			}
			c.DrainedHP++
		}
		if int8(c.MaxLevel) < 2 {
			e.killWithMessage(c, 8, "is dead forever")
			c.MaxHP = 0
			if i := bestClass(); int8(c.Levels[i]) > 0 && n == 2 {
				c.Levels[bestClass()] = 0
			}
			// jumps over RecomputeCharacter
		} else {
			c.Levels[bestClass()]--
			if c.HP == 0 && c.Status != 6 {
				e.killWithMessage(c, 6, "is killed")
			}
			e.RecomputeCharacter(c)
		}
		i := bestClass()
		c.MaxLevel = c.Levels[i]
		c.Exp = uint32(best)
		e.ApplyItems(c)
	}
}

// o13_23A4 (c, a, flag): drains one level from the gaze target.
func (e *Engine) o13_23A4(c *Character, a *Affect, flag byte) { e.drainLevels(c.Combat.Target, 1) }

// o13_23C2 (c, a, flag): drains two levels from the gaze target.
func (e *Engine) o13_23C2(c *Character, a *Affect, flag byte) { e.drainLevels(c.Combat.Target, 2) }

// o13_23EC (c, a, flag): disease; the target stops healing (0x32) and gets
// the wasting affect 0x39.
func (e *Engine) o13_23EC(c *Character, a *Affect, flag byte) {
	t := c.Combat.Target
	if e.FindAffect(t, 0x32) == nil {
		e.AddAffect(t, 0x32, 0x3840, 0xFF, 1)
		e.AddAffect(t, 0x39, 0xA8C0, byte(e.Dice(1, 6)<<4+0xF), 1)
		e.CharMessage(t, "is Diseased", 0xA, true)
	}
}

// o13_2490 (c, a, flag): marks the affect (bit 4 of its data) the first time
// it is rolled with a hit roll set.
func (e *Engine) o13_2490(c *Character, a *Affect, flag byte) {
	if e.v6DE3 == 0 && e.rv.hitRoll == 0 {
		a.Data &= 0x0F
	} else if a.Data&0x10 == 0 {
		e.rv.hitRoll = -1
		a.Data |= 0x10
	}
}

// levelBonus is the 1..5 step of the level ranges 4-6, 7-10, 11-13, 14-17,
// 18-20 used by the save handlers (0 otherwise; the original leaves its
// local uninitialised there).
func levelBonus(level byte) byte {
	switch {
	case level >= 4 && level <= 6:
		return 1
	case level >= 7 && level <= 10:
		return 2
	case level >= 11 && level <= 13:
		return 3
	case level >= 14 && level <= 17:
		return 4
	case level >= 18 && level <= 20:
		return 5
	}
	return 0
}

// o13_24D9 (c, a, flag): saving throw bonus by level for save type 0.
func (e *Engine) o13_24D9(c *Character, a *Affect, flag byte) {
	if e.rv.saveType == 0 {
		e.rv.save += levelBonus(c.Stats[statCon])
	}
}

// o13_2556 (c, a, flag): fully resisted by source 0xF, fire/electric (4)
// otherwise only drops the affect being added.
func (e *Engine) o13_2556(c *Character, a *Affect, flag byte) {
	if e.curSpell == 0xF {
		e.blockAffect(0)
		e.CharMessage(c, "is unaffected", 0xA, true)
	} else if e.rv.dmgType&4 != 0 {
		e.blockAffect(0)
	}
}

// o13_25A2 (c, a, flag): fire damage (bit 0) is halved.
func (e *Engine) o13_25A2(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&1 != 0 {
		e.halveDamage()
	}
}

// o13_25BC (c, a, flag): halves damage when the selected character's weapon
// has base flags 0x81.
func (e *Engine) o13_25BC(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil && e.itemBaseFlags(w)&0x81 != 0 {
		e.halveDamage()
	}
}

// o13_2610 (c, a, flag): the affect ends; a living character falls dead.
func (e *Engine) o13_2610(c *Character, a *Affect, flag byte) {
	a.Call = 0
	if c.Okay != 0 {
		e.killWithMessage(c, 6, "Falls dead")
	}
}

// o13_264B (c, a, flag): damage is cancelled unless the weapon is magical
// (+0x32 non-zero, or 0xB1 in +0x31).
func (e *Engine) o13_264B(c *Character, a *Affect, flag byte) {
	w := e.readiedWeapon(e.selected())
	if w == nil || w.Plus == 0 && w.Words[2] != 0xB1 {
		e.rv.damage = 0
	}
}

// o13_268A (c, a, flag): like 24D9 for save types 4 and 2.
func (e *Engine) o13_268A(c *Character, a *Affect, flag byte) {
	if e.rv.saveType == 4 || e.rv.saveType == 2 {
		e.rv.save += levelBonus(c.Stats[statCon])
	}
}

// o13_2700 (c, a, flag): +3 hit points up to the maximum.
func (e *Engine) o13_2700(c *Character, a *Affect, flag byte) {
	c.HP += 3
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
}

// o13_2730 (c, a, flag): a dying or unconscious character gets back up with
// the hit points it had left to bleed (6 minus the rounds), then 0x5F runs
// for 1d4+1 and this affect (0x63) is removed.
func (e *Engine) o13_2730(c *Character, a *Affect, flag byte) {
	var n byte
	if c.Status == 5 && c.Combat.Dying < 6 {
		n = 6 - c.Combat.Dying
	}
	if c.Status == 4 {
		n = 6
	}
	if n > 0 && e.Revive(c, n) {
		e.AddAffect(c, 0x5F, e.Dice(1, 4)+1, 0xFF, 1)
		a.Call = 0
		e.RemoveAffect(c, 0x63, a)
	}
}

// o13_27D9 (c, a, flag): damage that is neither fire (1) nor acid (0x10)
// starts affect 0x66 for 3d6.
func (e *Engine) o13_27D9(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&1 == 0 && e.rv.dmgType&0x10 == 0 {
		e.AddAffect(c, 0x66, e.Dice(3, 6), 0xFF, 1)
	}
}

// o13_2814 (c, a, flag): adds 0x3B for 3 rounds unless 0x62 or 0x3B is there.
func (e *Engine) o13_2814(c *Character, a *Affect, flag byte) {
	if e.FindAffect(c, 0x62) == nil && e.FindAffect(c, 0x3B) == nil {
		e.AddAffect(c, 0x3B, 3, 0xFF, 1)
	}
}

// o13_2866 (c, a, flag): back up at full hit points, else re-add 0x66.
func (e *Engine) o13_2866(c *Character, a *Affect, flag byte) {
	if !e.Revive(c, c.MaxHP) {
		e.addAffectNow(c, 0x66, a.Data, 1)
	}
}

// o13_28BF (c, a, flag): turns gaseous and leaves the fight.
func (e *Engine) o13_28BF(c *Character, a *Affect, flag byte) {
	a.Call = 0
	e.combatRedrawChar(c, 3, 0)
	e.KillCharacter(c, 3, "Turns gaseous and escapes")
}

// o13_2901 (c, a, flag).
func (e *Engine) o13_2901(c *Character, a *Affect, flag byte) { e.avoidDistantAttack(c, 0x3C) }

// resistRoll = ovr13:2915 (base): the chance base - 5*(11 - caster level)
// percent (as a byte) to drop the affect being added.
func (e *Engine) resistRoll(base byte) {
	lvl := e.casterLevel(int(e.curSpell)) & 0xFF
	thr := byte(int(base) - 5*(0xB-lvl))
	if e.rv.affVal != 0 || e.rv.dmgType&8 != 0 {
		if byte(e.Dice(1, 100)) <= thr {
			e.blockAffect(0)
		}
	}
}

// o13_2976 (c, a, flag).
func (e *Engine) o13_2976(c *Character, a *Affect, flag byte) { e.resistRoll(0x32) }

// o13_2984 (c, a, flag).
func (e *Engine) o13_2984(c *Character, a *Affect, flag byte) { e.resistRoll(0xF) }

// o13_2992 (c, a, flag): 90%: blocks 0x35 and 0xB.
func (e *Engine) o13_2992(c *Character, a *Affect, flag byte) {
	if e.Dice(1, 100) <= 0x5A {
		e.blockAffect(0x35)
		e.blockAffect(0xB)
	}
}

// o13_29B6 (c, a, flag): blocks 0xB and 0x35.
func (e *Engine) o13_29B6(c *Character, a *Affect, flag byte) {
	e.blockAffect(0xB)
	e.blockAffect(0x35)
}

// o13_29CB (c, a, flag): blocks 0x34.
func (e *Engine) o13_29CB(c *Character, a *Affect, flag byte) { e.blockAffect(0x34) }

// o13_29D9 (c, a, flag): cold damage (bit 1) is blocked.
func (e *Engine) o13_29D9(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&2 != 0 {
		e.blockAffect(0)
	}
}

// o13_29F0 (c, a, flag): blocks 0x37 and 0x34; saves of type 0 always pass.
func (e *Engine) o13_29F0(c *Character, a *Affect, flag byte) {
	e.blockAffect(0x37)
	e.blockAffect(0x34)
	if e.rv.saveType == 0 {
		e.rv.save = 0x64
	}
}

// o13_2A11 (c, a, flag): fire damage (bit 0) is blocked.
func (e *Engine) o13_2A11(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&1 != 0 {
		e.blockAffect(0)
	}
}

// o13_2A28 (c, a, flag): fire damage loses 1 per die, never below one per die.
func (e *Engine) o13_2A28(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&1 != 0 {
		for k := byte(1); k <= e.rv.diceN; k++ {
			e.rv.damage--
			if e.rv.damage < e.rv.diceN {
				e.rv.damage = e.rv.diceN
			}
		}
	}
}

// o13_2A6D (c, a, flag): electricity damage (bit 2) is halved.
func (e *Engine) o13_2A6D(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&4 != 0 {
		e.halveDamage()
	}
}

// o13_2A87 (c, a, flag): halves damage unless the weapon is a plain
// (flags non-zero, bit 0 clear) base item.
func (e *Engine) o13_2A87(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil {
		if f := e.itemBaseFlags(w); f == 0 || f&1 != 0 {
			e.halveDamage()
		}
	}
}

// o13_2AE6 (c, a, flag): halves damage when the weapon's +0x32 is positive.
func (e *Engine) o13_2AE6(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil && int8(w.Plus) > 0 {
		e.halveDamage()
	}
}

// o13_2B20 (c, a, flag): the selected character's first readied item of
// type 0x55 makes the damage 1d6+1.
func (e *Engine) o13_2B20(c *Character, a *Affect, flag byte) {
	if w := e.selected().Slots[0]; w != nil && w.Type == 0x55 {
		e.rv.damage = byte(e.RollDice(1, 6) + 1)
	}
}

// o13_2B63 (c, a, flag): cold damage (bit 1) is halved.
func (e *Engine) o13_2B63(c *Character, a *Affect, flag byte) {
	if e.rv.dmgType&2 != 0 {
		e.halveDamage()
	}
}

// o13_2B7D (c, a, flag): damage is cancelled unless the weapon has +0x32
// set, or the attacker has item type > 0 or level (+0x73) of 4 or more.
func (e *Engine) o13_2B7D(c *Character, a *Affect, flag byte) {
	if w := e.readiedWeapon(e.selected()); w != nil && w.Plus != 0 {
		return
	}
	s := e.selected()
	if int8(s.Race) > 0 || int8(s.MaxLevel) < 4 {
		e.rv.damage = 0
	}
}

// o13_2BC8 (c, a, flag): the selected character's readied item of type
// 0x57 or 0x58 triggers affect 0x32 handling.
func (e *Engine) o13_2BC8(c *Character, a *Affect, flag byte) {
	if w := e.selected().Slots[0]; w != nil && (w.Type == 0x57 || w.Type == 0x58) {
		e.avoidDistantAttack(c, 0x32)
	}
}

// o13_2C1C (c, a, flag): spits acid (25%) at a target under 4 squares away:
// 8d4 damage, half on a save; the affects 0x79 and 0x50 end.
func (e *Engine) o13_2C1C(c *Character, a *Affect, flag byte) {
	t := c.Combat.Target
	if e.Dice(1, 100) > 0x19 || e.rangeTo(c, t) >= 4 {
		return
	}
	e.clearCombatInfo(c)
	e.CharMessage(c, "Spits Acid", 0xA, true)
	e.hitColours(0x17)
	e.flyMissile(e.combatX(c), e.combatY(c), e.combatX(t), e.combatY(t), 1, 0x1E)
	dmg := e.RollDice(8, 4)
	saved := e.SavingThrow(t, 3, 0)
	e.Damage(t, dmg, saveHalf, saved)
	e.RemoveAffect(c, 0x79, a)
	e.RemoveAffect(c, 0x50, nil)
}

// o13_2D38 (c, a, flag): the selected character's readied item of type 0x56
// makes the damage 3d8; fire or magic types (mask 9) add the dice count.
func (e *Engine) o13_2D38(c *Character, a *Affect, flag byte) {
	if w := e.selected().Slots[0]; w != nil && w.Type == 0x56 {
		e.rv.damage = byte(e.RollDice(3, 8))
	}
	if e.rv.dmgType&9 != 0 {
		e.rv.damage += e.rv.diceN
	}
}

// o13_2D88 (c, a, flag): halves damage with a +0x32 = 0, +0x31 = 0xB1
// weapon, cancels it for another +0x32 = 0 weapon or no weapon.
func (e *Engine) o13_2D88(c *Character, a *Affect, flag byte) {
	w := e.readiedWeapon(e.selected())
	switch {
	case w == nil:
		e.rv.damage = 0
	case w.Plus == 0 && w.Words[2] == 0xB1:
		e.halveDamage()
	case w.Plus == 0:
		e.rv.damage = 0
	}
}

// o13_2DE4 (c, a, flag): 30%: blocks 0xB and 0x35.
func (e *Engine) o13_2DE4(c *Character, a *Affect, flag byte) {
	if e.Dice(1, 100) <= 0x1E {
		e.blockAffect(0xB)
		e.blockAffect(0x35)
	}
}

// o13_2E08 (c, a, flag): blocks 0xB, 0x35, 0x34 and 0x37; saves of type 0 pass.
func (e *Engine) o13_2E08(c *Character, a *Affect, flag byte) {
	e.blockAffect(0xB)
	e.blockAffect(0x35)
	e.blockAffect(0x34)
	e.blockAffect(0x37)
	if e.rv.saveType == 0 {
		e.rv.save = 0x64
	}
}

// o13_2E37 (c, a, flag): sets DS:681E when the gaze target carries a ring
// (0x98 with 0xFC) or 0x76 item in slots 1..3 of a used item.
func (e *Engine) o13_2E37(c *Character, a *Affect, flag byte) {
	t := c.Combat.Target
	for _, p := range t.Items {
		if e.v681E != 0 {
			break
		}
		if p.Readied != 0 {
			for k := 1; k <= 3; k++ {
				if p.Words[k-1] == 0x98 && p.Words[2] == 0xFC || p.Words[k-1] == 0x76 {
					e.v681E = 1
				}
			}
		}
	}
}

// o13_2EDA (c, a, flag): an item stands in for a; its +0x3D is the affect it
// grants (added on use with call 0, then run once; removed with flag set).
func (e *Engine) o13_2EDA(c *Character, a *Affect, flag byte) {
	typ := affectItem(a).Spells[1]
	if flag != 0 {
		e.RemoveAffect(c, typ, nil)
		return
	}
	e.AddAffect(c, typ, 0, 0xC, 0)
	e.callAffect(typ, c, nil, 0)
}

// o13_2F57 (c, a, flag): girdle of giant strength (affect 0x26). Removed
// (flag set): the affect goes unless the character already has 18/00 and
// the stored value cannot be told apart; added: strength 18/00.
func (e *Engine) o13_2F57(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		found := false
		giant := c.Stats[statStr] == 0x12 && c.Str00 == 0x64
		for _, p := range append([]*Affect(nil), c.Affects...) {
			if found {
				break
			}
			if p.Type == 0x26 {
				str, str00 := strDecode(p)
				// p.Raw[0] is the type byte (the original compares the first byte, not the data)
				if giant && p.Type < 0x80 || str == 0x12 && str00 == 0x64 && !giant {
					e.RemoveAffect(c, 0x26, p)
					found = true
				}
			}
		}
		return
	}
	out, ok := e.SetStrength(c, 0x12, 0x64)
	if ok {
		if e.beginFlag == 5 {
			e.beginFlag = 2
			e.CharMessage(c, "is stronger", 0xA, true)
			e.beginFlag = 5
		} else {
			e.CharMessage(c, "is stronger", 0xA, true)
		}
	}
	e.itemsRedraw = true // DS:6821
	e.AddAffect(c, 0x26, 0, out, 1)
}

// o13_30A5 (c, a, flag): cursed item (a stands in as the item): with the
// wrong alignment (+0x3D low nibble vs char +0xA0) it is unreadied and
// hurts by its high nibble.
func (e *Engine) o13_30A5(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		return
	}
	it := affectItem(a)
	if it.Spells[1]&0xF != c.Alignment {
		it.Readied = 0
		n := int(it.Spells[1] >> 4)
		e.rv.dmgType = 8
		if e.beginFlag == 5 {
			e.combatUseHook()
		}
		e.Damage(c, n, 0, false)
		e.itemsRedraw = true
	}
}

// o13_3134 (c, a, flag): the item needs strength 19 or more.
func (e *Engine) o13_3134(c *Character, a *Affect, flag byte) {
	if flag == 0 && c.Stats[statStr] < 0x13 {
		affectItem(a).Readied = 0
		e.BottomMessage("Must have Giant Strength")
	}
}

// o13_317A (c, a, flag): removing it ends affect 0x17.
func (e *Engine) o13_317A(c *Character, a *Affect, flag byte) {
	if flag != 0 {
		e.RemoveAffect(c, 0x17, nil)
	}
}

// o13_319B (c, a, flag): empty handler.
func (e *Engine) o13_319B(c *Character, a *Affect, flag byte) {}

// debugValues = ovr34:0004 (msg, a, b, c): with DS:49F7 set, show the three
// numbers on the top row and msg on row 24.
func (e *Engine) debugValues(msg string, a, b, c int) {
	if len(msg) > 0x28 {
		msg = msg[:0x28]
	}
	if e.v49F7 == 0 {
		return
	}
	e.ClearBox(0, 0, 0x27, 0)
	e.DisplayString(0, 0, 0xE, fmt.Sprintf("%d , %d , %d", a, b, c))
	e.MessageRow24b(0xE, msg)
	e.ClearBox(0, 0, 0x27, 0)
}
