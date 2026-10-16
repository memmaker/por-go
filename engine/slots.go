package engine

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Named save slots (extra), F9 at any prompt once a game is under way, and
// the party menu's Load (loading only). They replace the original's lettered
// saves A-J: Save is gone from camp and the party menu, and lettered saves
// in the save directory (an imported game's) become slots "GAME A".."GAME J".
// A slot is a folder <SaveDir>/saves/<name> holding the game's own save A
// (savgamA.dat and its CHRDATA* files), so the format is the original's.
// Loading unwinds whatever is running (a menu, a script, a battle) back to
// adventure, which resumes from the loaded game like the party menu's Load.
// Saving during combat is refused: the battle is not part of the format.

// slotLoaded is the panic that carries a load out to adventure.
type slotLoaded struct{}

type slot struct {
	name string
	at   time.Time
}

// adventure is the party menu, then the game loop; a slot load starts
// the game loop again on the loaded game.
func (e *Engine) adventure() {
	start := func() {
		e.PartyMenu()                 // ovr17:0161
		e.SetSkyColours(0xB, 0, 7, 6) // ovr30:0155
	}
	for e.guardedRun(start) {
		start = func() {}
	}
}

func (e *Engine) guardedRun(start func()) (reload bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(slotLoaded); !ok {
				panic(r)
			}
			e.unwindReset()
			reload = true
		}
	}()
	e.ext.guarded = true
	defer func() { e.ext.guarded = false }()
	start()
	e.GameLoop() // ovr04:32F8
	return false
}

// unwindReset drops the input state of the screens the load left.
func (e *Engine) unwindReset() {
	x := &e.ext
	x.inCommand, x.padHot, x.onSelect, x.menu, x.list, x.choosing = false, 0, nil, 0, false, false
	x.quick, x.ahead, x.later, x.regions, x.jump, x.raw = nil, nil, nil, nil, nil, ""
	x.typing, x.modal, x.floor, x.viewing, x.items, x.toItems, x.toSheet = false, 0, 0, false, false, false, false
	x.switchTo, x.buying, x.campBar, x.selSet, x.skip, x.paging = false, false, false, false, false, false
	x.fullMap, x.walk, x.back, x.aoe, x.picker, x.enemiesOnly = false, nil, false, 0, false, false
	e.pendingScan, e.draining = 0, false
	e.loadedGame = true // DS:49F8, as the party menu's Load
}

func (e *Engine) slotRoot() string { return filepath.Join(e.Cfg.SaveDir, "saves") }

// slots lists the saved games, newest first.
func (e *Engine) slots() []slot {
	e.adoptLettered()
	ents, _ := os.ReadDir(e.slotRoot())
	var out []slot
	for _, en := range ents {
		st, err := os.Stat(filepath.Join(e.slotRoot(), en.Name(), "SAVGAMA.DAT"))
		if en.IsDir() && err == nil {
			out = append(out, slot{en.Name(), st.ModTime()})
		}
	}
	slices.SortFunc(out, func(a, b slot) int { return b.at.Compare(a.at) })
	return out
}

// adoptLettered moves each savgam<L>.dat in the save directory, with its
// CHRDAT<L>* files, into the slot "GAME <L>" as SAVGAMA.DAT. The party file
// names inside the save stay as they are; loadSlot finds them in the folder.
func (e *Engine) adoptLettered() {
	ents, _ := os.ReadDir(e.Cfg.SaveDir)
	for _, en := range ents {
		up := strings.ToUpper(en.Name())
		if len(up) != 11 || !strings.HasPrefix(up, "SAVGAM") || !strings.HasSuffix(up, ".DAT") || up[6] < 'A' || up[6] > 'J' {
			continue
		}
		dir := filepath.Join(e.slotRoot(), "GAME "+up[6:7])
		if _, err := os.Stat(dir); err == nil {
			continue // ponytail: a slot of that name already exists; the old save stays put
		}
		if os.MkdirAll(dir, 0o755) != nil || os.Rename(filepath.Join(e.Cfg.SaveDir, en.Name()), filepath.Join(dir, "SAVGAMA.DAT")) != nil {
			continue
		}
		for _, f := range ents {
			if strings.HasPrefix(strings.ToUpper(f.Name()), "CHRDAT"+up[6:7]) {
				os.Rename(filepath.Join(e.Cfg.SaveDir, f.Name()), filepath.Join(dir, f.Name()))
			}
		}
	}
}

// slotName keeps what a folder name may hold.
func slotName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(" -_!'", r) {
			return r
		}
		return -1
	}, strings.ToUpper(s))
	return strings.TrimSpace(s)
}

// inSlot runs f with the save directory pointed at the slot's folder.
func (e *Engine) inSlot(name string, f func()) {
	dir := e.Cfg.SaveDir
	e.Cfg.SaveDir = filepath.Join(e.slotRoot(), name)
	defer func() { e.Cfg.SaveDir = dir }()
	f()
}

// LoadSlotNamed loads the slot without the screen (cmd/snap).
func (e *Engine) LoadSlotNamed(name string) (ok bool) {
	e.adoptLettered()
	e.inSlot(name, func() { ok = e.loadSlot('A') })
	return ok
}

func (e *Engine) saveTo(name string) {
	os.RemoveAll(filepath.Join(e.slotRoot(), name)) // no stale CHRDAT files of a bigger party
	e.inSlot(name, func() { e.saveSlot('A') })
}

const slotRows = 0x11

// SlotMenu is the F9 screen: the saves, newest first, and what to do.
// loadOnly (the party menu's Load) offers no saving and returns true after
// a load instead of unwinding to adventure.
func (e *Engine) SlotMenu(loadOnly bool) bool {
	defer e.modalOn()()
	restore := e.saveScreen()
	defer restore()
	canSave := !loadOnly && e.ext.inGame && len(e.party) > 0 && e.beginFlag != 5
	sel, top := 0, 0
	for {
		list := e.slots()
		sel = min(max(sel, 0), max(len(list)-1, 0))
		top = min(max(top, sel-slotRows+1), sel)
		e.DrawBoxTitled(1, 1, 0x26, 0x17, 0, 0xF, "Saved Games")
		e.ClearCells(2, 2, 0x25, 0x17)
		if len(list) == 0 {
			e.DisplayString(3, 3, 0xA, "No saved games yet")
		}
		for i := top; i < len(list) && i < top+slotRows; i++ {
			col := 0xA
			if i == sel {
				col = 0xF
				e.DisplayString(2, 3+i-top, 0xE, ">")
			}
			e.DisplayString(3, 3+i-top, col, cut(list[i].name, 0x15))
			e.DisplayString(0x19, 3+i-top, 0x7, list[i].at.Format("Jan 02 15:04"))
		}
		if canSave {
			e.DisplayString(2, 0x15, 0xE, "S=save new  O=overwrite")
		}
		e.DisplayString(2, 0x16, 0xE, "Enter=load  D=delete  Esc=back")
		// extra (nav.go): a click selects a line, a second one loads; the
		// keys in the footer are clicked by their words
		mark := e.regionMark()
		e.addRegion(region{x0: 2, y0: 3, x1: 0x25, y1: 3 + slotRows - 1, click: func(_, cy int) uint16 {
			i := top + cy - 3
			if i >= len(list) {
				return 0
			}
			if i == sel {
				return 13
			}
			sel = i
			return keyNone
		}})
		for _, b := range []struct {
			x, y, w int
			k       uint16
		}{{2, 0x15, 10, 'S'}, {0xE, 0x15, 11, 'O'}, {2, 0x16, 10, 13}, {0xE, 0x16, 8, 'D'}, {0x18, 0x16, 8, 27}} {
			if b.y == 0x15 && !canSave {
				continue
			}
			e.addRegion(region{x0: b.x, y0: b.y, x1: b.x + b.w - 1, y1: b.y, click: func(int, int) uint16 { return b.k }})
		}
		k := e.ReadKey()
		e.regionCut(mark)
		var cur string
		if sel < len(list) {
			cur = list[sel].name
		}
		switch {
		case k == 27 || k == KeyF9:
			return false
		case k == 0x4800:
			sel--
		case k == 0x5000:
			sel++
		case (k == 's' || k == 'S') && canSave:
			name := slotName(e.InputString("Save as: ", 0xA, 0x14))
			if name == "" {
				continue
			}
			if slices.ContainsFunc(list, func(s slot) bool { return s.name == name }) &&
				e.YesNo(0xE, 0xA, 0xF, "Overwrite "+name+"? ") != 'Y' {
				continue
			}
			e.saveTo(name)
			sel = 0
		case (k == 'o' || k == 'O') && canSave && cur != "":
			if e.YesNo(0xE, 0xA, 0xF, cut("Overwrite "+cur+"? ", 0x28)) == 'Y' {
				e.saveTo(cur)
				sel = 0
			}
		case (k == 'd' || k == 'D') && cur != "":
			if e.YesNo(0xE, 0xA, 0xF, cut("Delete "+cur+"? ", 0x28)) == 'Y' {
				os.RemoveAll(filepath.Join(e.slotRoot(), cur))
			}
		case (k == 13 || k == 'l' || k == 'L') && cur != "":
			if e.beginFlag == 5 {
				e.combatViewLeave()
			}
			ok := false
			e.inSlot(cur, func() { ok = e.loadSlot('A') })
			if ok && loadOnly {
				return true
			}
			if ok {
				panic(slotLoaded{})
			}
		}
	}
}
