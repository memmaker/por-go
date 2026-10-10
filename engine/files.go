package engine

import (
	"bufio"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Character file I/O, ported from ovr18. Record layouts are kept raw so the
// original SAVE files load and save byte for byte.

const (
	chSpellsPtr = 0x7F  // list head, zeroed on load
	chItemsPtr  = 0xC8  // list head, zeroed on load
	chNextPtr   = 0x104 // party link, zeroed on load
	chCombatPtr = 0x108
	chIconSize  = 0xC0
	chIconHead  = 0xBD
	chIconBody  = 0xBE
	chOrder     = 0xBF
	chPortHead  = 0xBB
	chPortBody  = 0xBC
	chRandomID  = 0xAB
	chRace      = 0x2E
	chGender    = 0x9E
	chLevels    = 0x96 // 7 class levels

	itemRecSize   = 0x3F
	affectRecSize = 9
	gsPartyCount  = 0x67C // game-state word
	gsSymbolSet   = 0x624
)

// Item and Affect are the raw list nodes (next pointers live in the slices).
type Item struct{ Raw [itemRecSize]byte }
type Affect struct {
	Raw  [affectRecSize]byte
	Item *Item // set when an item stands in for the affect (ovr13 item handlers)
}

// savePath resolves a file name inside the save directory case-insensitively
// (DOS was case-insensitive; the shipped files are upper case).
func (e *Engine) savePath(name string) string {
	dir := e.Cfg.SaveDir
	if ents, err := os.ReadDir(dir); err == nil {
		for _, en := range ents {
			if strings.EqualFold(en.Name(), name) {
				return filepath.Join(dir, en.Name())
			}
		}
	}
	return filepath.Join(dir, strings.ToUpper(name))
}

func (e *Engine) fileExists(path string) bool { // 0802:02EB
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// removeSubstr = 0802:0356; makeFileName = 0802:0425.
// The scan runs over the original length and does not recheck the position
// after a deletion, so adjacent copies leave every second one (original quirk).
func removeSubstr(s, sub string) string {
	if sub == "" {
		return s
	}
	for i, n := 1, len(s)-len(sub)+1; i <= n; i++ {
		if i-1+len(sub) <= len(s) && s[i-1:i-1+len(sub)] == sub {
			s = s[:i-1] + s[i-1+len(sub):]
		}
	}
	return s
}
func makeFileName(s string) string {
	for _, c := range " .*,?/\\:;|" { // DS:0D1A..0D23
		s = removeSubstr(s, string(c))
	}
	if len(s) > 8 {
		s = s[:8]
	}
	return strings.ToUpper(s)
}

// charBase = savePath + MakeFileName(RemoveSubstr(name,'.')) for roster files.
func (e *Engine) charBase(c *Character) string {
	return makeFileName(removeSubstr(c.Name(), "."))
}

// fileError reports a save file that could not be read or written, once.
// (DOS asked for the save disk here and retried until it worked.)
func (e *Engine) fileError(verb, path string) {
	s := "Cannot " + verb + " " + filepath.Base(path)
	if len(s) > 40 {
		s = s[:40]
	}
	e.MessageRow24b(0xE, s)
}

// MessageRow24b = 0709:0BF5: clear row 24, show, wait for a key.
func (e *Engine) MessageRow24b(colour int, s string) byte {
	if e.Trace != nil {
		e.Trace("0709:0BF5", "MessageRow24b", colour, s)
	}
	e.ClearRow24()
	e.DisplayString(0, 24, colour, s)
	return e.ReadKeyByte()
}

// BuildCharList = ovr18:0010: names from charlist.txt, "* " for party members.
func (e *Engine) BuildCharList() []string {
	p := e.savePath("charlist.txt")
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var list []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name := strings.TrimRight(sc.Text(), "\r")
		if len(name) > 40 {
			name = name[:40]
		}
		for _, c := range e.party {
			if c.Name() == name {
				name = "* " + name
				if len(name) > 40 {
					name = name[:40]
				}
				break
			}
		}
		list = append(list, name)
	}
	return list
}

// WriteCharList = ovr18:01B8: rewrite charlist.txt without "*" entries.
func (e *Engine) WriteCharList(list []string) {
	os.MkdirAll(e.Cfg.SaveDir, 0o755)
	f, err := os.Create(e.savePath("charlist.txt"))
	if err != nil {
		e.fileError("write", "CHARLIST.TXT")
		return
	}
	defer f.Close()
	for _, n := range list {
		if len(n) > 0 && n[0] == '*' {
			continue
		}
		f.WriteString(n + "\r\n")
	}
}

// SaveCharacter = ovr18:069E (c, name): name "" = roster save (.cha, listed
// in charlist.txt), else <name>.sav as used by SaveGame.
func (e *Engine) SaveCharacter(c *Character, name string) {
	var base, ext string
	if name == "" {
		list := append(e.BuildCharList(), c.Name())
		e.WriteCharList(list)
		base, ext = e.charBase(c), ".cha"
	} else {
		if len(name) > 8 {
			name = name[:8]
		}
		base, ext = name, ".sav"
	}
	os.MkdirAll(e.Cfg.SaveDir, 0o755)
	if err := os.WriteFile(e.savePath(base+ext), c.Raw[:], 0o644); err != nil {
		e.fileError("write", e.savePath(base+ext))
	}
	os.Remove(e.savePath(base + ".itm"))
	if len(c.Items) > 0 {
		var b []byte
		for _, it := range c.Items {
			b = append(b, it.Raw[:]...)
		}
		os.WriteFile(e.savePath(base+".itm"), b, 0o644)
	}
	os.Remove(e.savePath(base + ".spc"))
	if len(c.Affects) > 0 {
		var b []byte
		for _, sp := range c.Affects {
			b = append(b, sp.Raw[:]...)
		}
		os.WriteFile(e.savePath(base+".spc"), b, 0o644)
	}
}

// LoadCharacter = ovr18:0AEC (name, c, useSav). Returns false when the file
// is missing.
func (e *Engine) LoadCharacter(name string, c *Character, useSav bool) bool {
	nm := makeFileName(name)
	ext := ".cha"
	if useSav {
		ext = ".sav"
	}
	b, err := os.ReadFile(e.savePath(nm + ext))
	if err != nil {
		e.fileError("read", e.savePath(nm+ext))
		return false
	}
	copy(c.Raw[:], b)
	c.Items, c.Affects = nil, nil
	for i := 0; i < 4; i++ { // pointers zeroed on load
		c.Raw[chSpellsPtr+i], c.Raw[chItemsPtr+i], c.Raw[chNextPtr+i], c.Raw[chCombatPtr+i] = 0, 0, 0, 0
	}
	if b, err := os.ReadFile(e.savePath(nm + ".itm")); err == nil {
		for i := 0; i+itemRecSize <= len(b); i += itemRecSize {
			it := &Item{}
			copy(it.Raw[:], b[i:])
			c.Items = append(c.Items, it)
		}
	}
	if b, err := os.ReadFile(e.savePath(nm + ".spc")); err == nil {
		for i := 0; i+affectRecSize <= len(b); i += affectRecSize {
			sp := &Affect{}
			copy(sp.Raw[:], b[i:])
			c.Affects = append(c.Affects, sp)
		}
	}
	e.ClearKeyboard()
	return true
}

// LoadMonster = ovr18:0EC4 (index, c): MON<n>CHA/SPC/ITM.DAX block index.
func (e *Engine) LoadMonster(index int, c *Character) bool {
	n := strconv.Itoa(int(e.symDAX[4] - '0'))
	b := e.L.Block("MON"+n+"CHA", index)
	if b == nil {
		e.MessageRow24b(0xF, "Unable to load monster")
		e.Exit()
		return false
	}
	copy(c.Raw[:], b)
	c.Items, c.Affects = nil, nil
	switch c.Raw[chRace] {
	case 1, 3, 5:
		c.Raw[chIconSize] = 1
	case 2, 4, 6, 7:
		c.Raw[chIconSize] = 2
	}
	for i := 1; i <= 3; i++ {
		r := byte(e.Dice(1, 6))
		c.Raw[chIconSize+i] = (r+8)<<4 | r
	}
	c.Raw[0xC4] = 0xC4
	for i := 5; i <= 6; i++ {
		r := byte(e.Dice(1, 6))
		c.Raw[chIconSize+i] = (r+8)<<4 | r
	}
	e.loadingMonster = true // DS:557C
	if b := e.L.Block("MON"+n+"SPC", index); b != nil {
		for i := 0; i+affectRecSize <= len(b); i += affectRecSize {
			sp := &Affect{}
			copy(sp.Raw[:], b[i:])
			c.Affects = append(c.Affects, sp)
		}
	}
	if b := e.L.Block("MON"+n+"ITM", index); b != nil {
		for i := 0; i+itemRecSize <= len(b); i += itemRecSize {
			it := &Item{}
			copy(it.Raw[:], b[i:])
			c.Items = append(c.Items, it)
		}
	}
	e.loadingMonster = false
	e.ClearKeyboard()
	return true
}

// Portrait tables DS:8C7.. (gender-major).
var (
	portHeads     = [2][8]byte{{1, 2, 3, 4, 5, 8, 0xB, 0xC}, {6, 7, 9, 0xA, 0xD, 0xE, 6, 7}}
	portBodyCls0  = [2][5]byte{{2, 4, 2, 4, 2}, {2, 4, 8, 2, 4}}
	portBodyOther = [2][5]byte{{1, 3, 4, 5, 6}, {7, 9, 0xB, 7, 9}}
	portBodyCls5  = [2][5]byte{{2, 2, 2, 2, 2}, {8, 0xA, 0xC, 8, 0xA}}
	portBodyCls6  = [2][5]byte{{6, 6, 6, 6, 6}, {8, 0xA, 8, 0xA, 8}}
)

// AddMonsterToParty = ovr18:1262 (index): NPC joins the party.
func (e *Engine) AddMonsterToParty(index int) {
	if e.partyCount() > 7 {
		return
	}
	c := &Character{}
	e.LoadMonster(index, c)
	c.Raw[chRandomID] = byte(e.Random(0x100))
	var classes []int
	for cls := 0; cls < 7; cls++ {
		if int8(c.Raw[chLevels+cls]) > 0 { // jle on a signed byte (ovr18:12CA)
			classes = append(classes, cls)
		}
	}
	if len(classes) == 0 {
		classes = []int{2}
	}
	cls := classes[e.Dice(1, len(classes))-1]
	r := e.Dice(1, 5) - 1
	g := int(c.Raw[chGender]) & 1
	switch cls {
	case 0:
		c.Raw[chPortBody] = portBodyCls0[g][r]
	case 5:
		c.Raw[chPortBody] = portBodyCls5[g][r]
	case 6:
		c.Raw[chPortBody] = portBodyCls6[g][r]
	default:
		c.Raw[chPortBody] = portBodyOther[g][r]
	}
	c.Raw[chPortHead] = portHeads[g][e.Dice(1, 8)-1]
	e.AddToParty(c)
	e.BuildCharIcon(c, true)
}

func (e *Engine) partyCount() int { return int(binary.LittleEndian.Uint16(e.game[gsPartyCount:])) }
func (e *Engine) setPartyCount(n int) {
	binary.LittleEndian.PutUint16(e.game[gsPartyCount:], uint16(n))
}

// AddToParty = ovr18:1446 (c): append, give the lowest free order 0..7.
func (e *Engine) AddToParty(c *Character) {
	c.Raw[chOrder] = 0xFF
	e.party = append(e.party, c)
	var used [8]bool
	for _, m := range e.party {
		if m.Raw[chOrder] < 8 {
			used[m.Raw[chOrder]] = true
		}
	}
	e.partySel = len(e.party) - 1
	o := 0
	for o < 8 && used[o] {
		o++
	}
	c.Raw[chOrder] = byte(o)
	e.setPartyCount(e.partyCount() + 1)
	e.RecomputeCharacter(c) // ovr24:0000
}

// removeFromParty unlinks c (used by Remove/Drop); order values stay.
func (e *Engine) removeFromParty(c *Character) {
	for i, m := range e.party {
		if m == c {
			e.party = append(e.party[:i], e.party[i+1:]...)
			break
		}
	}
	if e.partySel >= len(e.party) {
		e.partySel = len(e.party) - 1
	}
	if e.partySel < 0 {
		e.partySel = 0
	}
}

// eraseRosterCopies deletes <name>.cha/.itm/.spc (nested check as in LoadGame
// when nested is true, independent checks as in SaveGame otherwise).
func (e *Engine) eraseRosterCopies(c *Character, nested bool) {
	base := e.charBase(c)
	cha := e.savePath(base + ".cha")
	if e.fileExists(cha) {
		os.Remove(cha)
		os.Remove(e.savePath(base + ".itm"))
		os.Remove(e.savePath(base + ".spc"))
	} else if !nested {
		os.Remove(e.savePath(base + ".itm"))
		os.Remove(e.savePath(base + ".spc"))
	}
}

// LoadGame = ovr18:1602.
func (e *Engine) LoadGame() {
	menu := ""
	for L := byte('A'); L <= 'J'; L++ {
		if e.fileExists(e.savePath("savgam" + string(L) + ".dat")) {
			menu += string(L) + " "
		}
	}
	if menu == "" {
		return
	}
	var key byte
	for {
		key, _ = e.DisplayInput("Load Which Game: ", menu, 0xD, 0xA, 0xF, false, false)
		if key >= 'A' && key <= 'J' && e.fileExists(e.savePath("savgam"+string(key)+".dat")) {
			break
		}
		if key == 0 {
			return
		}
	}
	b, err := os.ReadFile(e.savePath("savgam" + string(key) + ".dat"))
	if err != nil || len(b) < 0x3351 {
		e.fileError("read", e.savePath("savgam"+string(key)+".dat"))
		return
	}
	e.ClearRow24()
	e.DisplayString(0, 24, 0xA, "Loading...Please Wait")
	p := 0
	rd := func(n int) []byte { s := b[p : p+n]; p += n; return s }
	e.symDAX = "8X8D" + string('0'+rd(1)[0])
	copy(e.area, rd(0x800))
	copy(e.game, rd(0x800))
	copy(e.aux, rd(0x400))
	copy(e.ecl, rd(0x1E00))
	pos := rd(5)
	e.posX, e.posY, e.facing = int(pos[0]), int(pos[1]), int(pos[2])
	e.pos3, e.pos4 = pos[3], pos[4]
	e.viewMode = int(rd(1)[0])
	e.beginFlag = rd(1)[0]
	count := int(rd(1)[0])
	names := rd(0x148)
	w := binary.LittleEndian.Uint16(e.area[0x1FE:])
	e.flag49F6, e.flag49F5 = byte(w>>1), byte(w&1)
	e.msgDelay = e.area[0x1F8]
	e.setPartyCount(0)
	e.party, e.partySel = nil, 0
	for i := 0; i < count && i < 8; i++ {
		ps := names[i*0x29:]
		n := int(ps[0])
		if n > 40 {
			n = 40
		}
		nm := makeFileName(removeSubstr(string(ps[1:1+n]), "."))
		if e.fileExists(e.savePath(nm + ".sav")) {
			c := &Character{}
			if e.LoadCharacter(nm, c, true) {
				e.AddToParty(c)
			}
		}
	}
	for _, c := range e.party {
		e.eraseRosterCopies(c, true)
	}
	for _, c := range e.party {
		e.BuildCharIcon(c, true)
	}
	e.partySel = 0
	e.symDAX = "8X8D" + string('0'+e.game[gsSymbolSet])
	if e.viewMode < 2 {
		e.LoadGeo(int(e.area[0x18A])) // ovr30:10C7
	}
	if e.viewMode > 1 {
		e.LoadIconPic("ICON", 0, 0xB) // ovr33:01C8
	} else {
		for i := 1; i <= 3; i++ {
			e.mapPairA[i-1] = int(int16(binary.LittleEndian.Uint16(e.area[0x3F2+2*i:])))
			e.mapPairB[i-1] = int(int16(binary.LittleEndian.Uint16(e.area[0x3F8+2*i:])))
			if e.mapPairA[i-1] >= 0 {
				e.LoadWalldef(e.mapPairA[i-1]&0xFF, e.mapPairB[i-1]&0xFF) // ovr30:0E08
			}
		}
	}
	if binary.LittleEndian.Uint16(e.area[0x344:]) != 0 {
		e.EndgameHook(0) // ovr19:06B3
	}
	e.ClearKeyboard()
	e.ClearRow24()
}

// SaveGame = ovr18:1D58.
func (e *Engine) SaveGame() {
	var key byte
	for {
		key, _ = e.DisplayInput("Save Which Game: ", "A B C D E F G H I J", 0xD, 0xA, 0xF, false, e.beginFlag == 2) // ovr18:1D8B
		if key == 0 || key >= 'A' && key <= 'J' {
			break
		}
	}
	if key == 0 {
		return
	}
	e.ClearRow24()
	e.DisplayString(0, 24, 0xA, "Saving...Please Wait")
	for i := 1; i <= 3; i++ {
		binary.LittleEndian.PutUint16(e.area[0x3F2+2*i:], uint16(e.mapPairA[i-1]))
		binary.LittleEndian.PutUint16(e.area[0x3F8+2*i:], uint16(e.mapPairB[i-1]))
	}
	binary.LittleEndian.PutUint16(e.area[0x1F8:], uint16(e.msgDelay))
	binary.LittleEndian.PutUint16(e.area[0x1FE:], uint16(e.flag49F6)*2+uint16(e.flag49F5))
	binary.LittleEndian.PutUint16(e.game[gsSymbolSet:], uint16(e.symDAX[4]-'0'))
	b := []byte{e.symDAX[4] - '0'}
	b = append(b, e.area...)
	b = append(b, e.game...)
	b = append(b, e.aux...)
	b = append(b, e.ecl...)
	b = append(b, byte(e.posX), byte(e.posY), byte(e.facing), e.pos3, e.pos4)
	b = append(b, byte(e.viewMode), e.beginFlag, byte(len(e.party)))
	names := make([]byte, 0x148)
	for i := range e.party {
		s := "CHRDAT" + string(key) + strconv.Itoa(i+1)
		names[i*0x29] = byte(len(s))
		copy(names[i*0x29+1:], s)
	}
	b = append(b, names...)
	os.MkdirAll(e.Cfg.SaveDir, 0o755)
	if err := os.WriteFile(e.savePath("savgam"+string(key)+".dat"), b, 0o644); err != nil {
		e.fileError("write", e.savePath("savgam"+string(key)+".dat"))
	}
	for i, c := range e.party {
		e.SaveCharacter(c, "CHRDAT"+string(key)+strconv.Itoa(i+1))
		e.eraseRosterCopies(c, false)
	}
	e.saved = true
	e.ClearRow24()
}
