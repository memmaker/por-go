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

// adoptLettered copies each savgam<L>.dat in the save directory, with its
// CHRDAT<L>* files, into the slot "GAME <L>" as SAVGAMA.DAT, once (the
// marker saves/.lettered): the originals stay for the emulator harness, and
// a deleted slot stays deleted. The party file names inside the save stay as
// they are; loadSlot finds them in the folder.
func (e *Engine) adoptLettered() {
	mark := filepath.Join(e.slotRoot(), ".lettered")
	if _, err := os.Stat(mark); err == nil || e.Cfg.SaveDir == "" {
		return
	}
	ents, _ := os.ReadDir(e.Cfg.SaveDir)
	for _, en := range ents {
		up := strings.ToUpper(en.Name())
		if len(up) != 11 || !strings.HasPrefix(up, "SAVGAM") || !strings.HasSuffix(up, ".DAT") || up[6] < 'A' || up[6] > 'J' {
			continue
		}
		dir := filepath.Join(e.slotRoot(), "GAME "+up[6:7])
		if _, err := os.Stat(dir); err == nil {
			continue // a slot of that name exists already
		}
		os.MkdirAll(dir, 0o755)
		copyFile(filepath.Join(e.Cfg.SaveDir, en.Name()), filepath.Join(dir, "SAVGAMA.DAT"))
		for _, f := range ents {
			if strings.HasPrefix(strings.ToUpper(f.Name()), "CHRDAT"+up[6:7]) {
				copyFile(filepath.Join(e.Cfg.SaveDir, f.Name()), filepath.Join(dir, f.Name()))
			}
		}
	}
	os.MkdirAll(e.slotRoot(), 0o755)
	os.WriteFile(mark, nil, 0o644)
}

// copyFile copies src to dst with src's time (the slot list sorts by it).
func copyFile(src, dst string) {
	b, err := os.ReadFile(src)
	st, err2 := os.Stat(src)
	if err != nil || err2 != nil || os.WriteFile(dst, b, 0o644) != nil {
		return
	}
	os.Chtimes(dst, st.ModTime(), st.ModTime())
}

// lettered: the original's Save and Load (savgam<L>.dat) instead of the
// slot screen, for traced runs only.
func (e *Engine) lettered() bool { return e.Trace != nil }

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

// SlotIndex is the slot's line on the slot screen, or -1 (cmd/snap).
func (e *Engine) SlotIndex(name string) int {
	return slices.IndexFunc(e.slots(), func(s slot) bool { return s.name == name })
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

// SlotMenu is the F9 screen: "New Slot", then the saves, newest first. The
// cursor starts on the newest save, the one saved last. Enter saves (on New
// Slot under a new name, on a save as an overwrite after a Yes); L loads.
// loadOnly (the party menu's Load) offers no saving, Enter loads there, and
// it returns true after a load instead of unwinding to adventure.
func (e *Engine) SlotMenu(loadOnly bool) bool {
	defer e.modalOn()()
	restore := e.saveScreen()
	defer restore()
	canSave := !loadOnly && e.ext.inGame && len(e.party) > 0 && e.beginFlag != 5
	first := 0 // the row of the first save
	if canSave {
		first = 1
	}
	sel, top := first, 0
	type key struct {
		label string
		k     uint16
	}
	footer := []key{{"L=load", 'L'}, {"D=delete", 'D'}, {"Esc=back", 27}}
	switch {
	case canSave:
		footer = append([]key{{"Enter=save", 13}}, footer...)
	case loadOnly:
		footer = append([]key{{"Enter=load", 13}}, footer[1:]...)
	}
	for {
		list := e.slots()
		rows := len(list) + first
		sel = min(max(sel, 0), max(rows-1, 0))
		top = min(max(top, sel-slotRows+1), sel)
		e.DrawBoxTitled(1, 1, 0x26, 0x17, 0, 0xF, "Saved Games")
		e.ClearCells(2, 2, 0x25, 0x17)
		if rows == 0 {
			e.DisplayString(3, 3, 0xA, "No saved games yet")
		}
		for i := top; i < rows && i < top+slotRows; i++ {
			col := 0xA
			if i == sel {
				col = 0xF
				e.DisplayString(2, 3+i-top, 0xE, ">")
			}
			if i < first {
				e.DisplayString(3, 3+i-top, col, "New Slot")
				continue
			}
			e.DisplayString(3, 3+i-top, col, cut(list[i-first].name, 0x15))
			e.DisplayString(0x19, 3+i-top, 0x7, list[i-first].at.Format("Jan 02 15:04"))
		}
		if !canSave && !loadOnly {
			e.DisplayString(2, 0x15, 0xE, "No saving now")
		}
		// extra (nav.go): a click selects a line, a second one is Enter; the
		// keys in the footer are clicked by their words
		mark := e.regionMark()
		e.addRegion(region{x0: 2, y0: 3, x1: 0x25, y1: 3 + slotRows - 1, click: func(_, cy int) uint16 {
			i := top + cy - 3
			if i >= rows {
				return 0
			}
			if i == sel {
				return 13
			}
			sel = i
			return keyNone
		}})
		x, gap := 2, 0x24
		for _, f := range footer {
			gap -= len(f.label)
		}
		gap /= len(footer) - 1 // spread over the row inside the frame
		for _, f := range footer {
			e.DisplayString(x, 0x16, 0xE, f.label)
			e.addRegion(region{x0: x, y0: 0x16, x1: x + len(f.label) - 1, y1: 0x16, click: func(int, int) uint16 { return f.k }})
			x += len(f.label) + gap
		}
		k := e.ReadKey()
		e.regionCut(mark)
		cur := ""
		if sel >= first && sel < rows {
			cur = list[sel-first].name
		}
		switch {
		case k == 27 || k == KeyF9:
			return false
		case k == 0x4800:
			sel--
		case k == 0x5000:
			sel++
		case k == 13 && canSave && cur == "": // New Slot
			name := slotName(e.InputString("Save as: ", 0xA, 0x14))
			if name == "" {
				continue
			}
			if slices.ContainsFunc(list, func(s slot) bool { return s.name == name }) &&
				e.YesNo(0xE, 0xA, 0xF, cut("Overwrite "+name+"? ", 0x28)) != 'Y' {
				continue
			}
			e.saveTo(name)
			sel = first
		case k == 13 && canSave:
			if e.YesNo(0xE, 0xA, 0xF, cut("Overwrite "+cur+"? ", 0x28)) == 'Y' {
				e.saveTo(cur)
				sel = first
			}
		case (k == 'd' || k == 'D') && cur != "":
			if e.YesNo(0xE, 0xA, 0xF, cut("Delete "+cur+"? ", 0x28)) == 'Y' {
				os.RemoveAll(filepath.Join(e.slotRoot(), cur))
			}
		case (k == 'l' || k == 'L' || k == 13 && loadOnly) && cur != "":
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
