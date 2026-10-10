// Command fixture builds a save fixture for a trace scenario
// (decomp/emu/saves/NAME): save A of SAVE/ with its party, moved into
// another area. The area's ECL block replaces the saved script, as when a
// script enters it (ovr04:0BD1 opNewECL), and the party stands at x,y.
//
//	fixture -out DIR [-block N -walls A,B,C [-geo G]] [-x X -y Y -dir D] [-set VAR=VAL ...] [-strong]
//
// -walls are the WALLDEF blocks of wall slots 1..3 as SaveGame records them
// (255 none; a block of two chunks fills its slot and the next, which is
// then 255), as the area's LOAD PIECES leaves them. LoadGame reloads them.
// -set pokes an ECL variable (0x4900.. area record, 0x6B00.. game record)
// as a word. -strong makes every member hit always and take a long time to
// die, so a scripted Quick fight ends in a win.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"por/dax"
)

// SAVGAMx.DAT layout (engine/files.go LoadGame).
const (
	svDisk  = 0x0000
	svArea  = 0x0001 // 0x800
	svGame  = 0x0801 // 0x800
	svECL   = 0x1401 // 0x1E00
	svPos   = 0x3201 // x, y, facing, 2 more
	svCount = 0x3208
	svNames = 0x3209 // 8 x 0x29 Pascal strings
	svSize  = 0x3351
)

type sets []string

func (s *sets) String() string     { return strings.Join(*s, ",") }
func (s *sets) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	out := flag.String("out", "", "fixture directory to write")
	block := flag.Int("block", -1, "ECL block to enter (-1 keeps the saved area)")
	x := flag.Int("x", -1, "party x")
	y := flag.Int("y", -1, "party y")
	dir := flag.Int("dir", -1, "facing 0 N, 2 E, 4 S, 6 W")
	geo := flag.Int("geo", -1, "GEO block to load (default: the ECL block)")
	walls := flag.String("walls", "", "WALLDEF blocks of wall slots 1,2,3 (255 none)")
	strong := flag.Bool("strong", false, "make the party win fights")
	var vars sets
	flag.Var(&vars, "set", "VAR=VAL, hex, repeatable")
	flag.Parse()
	if *out == "" {
		log.Fatal("-out is required")
	}
	root := ".."
	sv, err := os.ReadFile(filepath.Join(root, "SAVE", "SAVGAMA.DAT"))
	if err != nil || len(sv) != svSize {
		log.Fatalf("SAVE/SAVGAMA.DAT: %v (%d bytes)", err, len(sv))
	}
	area, game := sv[svArea:svArea+0x800], sv[svGame:svGame+0x800]

	if *block >= 0 {
		l, err := dax.NewLoader(root)
		if err != nil {
			log.Fatal(err)
		}
		disk := 0
		var b []byte
		for d := 1; d <= 8 && disk == 0; d++ {
			if b = l.Block(fmt.Sprintf("ECL%d", d), *block); len(b) > 2 {
				disk = d
			}
		}
		if disk == 0 {
			log.Fatalf("no ECL file has block %d", *block)
		}
		sv[svDisk] = byte(disk)
		game[0x624] = byte(disk) // DS:6E12 symbol set
		binary.LittleEndian.PutUint16(area[0x1E4:], uint16(*block))
		g := *geo
		if g < 0 {
			g = *block // most areas load the GEO block of their own number
		}
		binary.LittleEndian.PutUint16(area[0x18A:], uint16(g))
		ecl := sv[svECL : svECL+0x1E00]
		clear(ecl)
		copy(ecl, b[2:])
		ws := strings.Split(*walls, ",")
		if len(ws) != 3 {
			log.Fatal("-block needs -walls A,B,C")
		}
		for i, w := range ws { // SaveGame: area 0x3F4+2i block, 0x3FA+2i slot (-1 none)
			id, err := strconv.Atoi(w)
			if err != nil {
				log.Fatalf("bad -walls %q", *walls)
			}
			blk, slot := id, i+1
			if id == 0xFF {
				blk, slot = -1, -1
			}
			binary.LittleEndian.PutUint16(area[0x3F4+2*i:], uint16(blk))
			binary.LittleEndian.PutUint16(area[0x3FA+2*i:], uint16(slot))
		}
		for v := 0x4A00; v <= 0x4A1F; v++ { // zeroed on entry (InitAreaScript)
			binary.LittleEndian.PutUint16(area[2*(v-0x4900):], 0)
		}
	}
	for i, v := range []int{*x, *y, *dir} {
		if v >= 0 {
			sv[svPos+i] = byte(v)
		}
	}
	for _, s := range vars {
		k, v, ok := strings.Cut(s, "=")
		a, err1 := strconv.ParseUint(k, 16, 16)
		n, err2 := strconv.ParseUint(v, 16, 16)
		if !ok || err1 != nil || err2 != nil {
			log.Fatalf("bad -set %q", s)
		}
		switch {
		case a >= 0x4900 && a <= 0x4CFF:
			binary.LittleEndian.PutUint16(area[2*(a-0x4900):], uint16(n))
		case a >= 0x6B00 && a <= 0x6EFF:
			binary.LittleEndian.PutUint16(game[2*(a-0x6B00):], uint16(n))
		default:
			log.Fatalf("-set %s: not an area or game variable", k)
		}
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*out, "SAVGAMA.DAT"), sv, 0o644); err != nil {
		log.Fatal(err)
	}
	for i := 0; i < int(sv[svCount]); i++ {
		p := sv[svNames+i*0x29:]
		name := string(p[1 : 1+p[0]])
		for _, ext := range []string{".SAV", ".SPC", ".ITM"} {
			b, err := os.ReadFile(filepath.Join(root, "SAVE", name+ext))
			if err != nil {
				continue
			}
			if ext == ".SAV" && *strong {
				b[0x2D] = 40                             // hit dice: THAC0 is recomputed from it
				b[0x32], b[0x11B] = 250, 250             // max HP, HP
				b[0xA9], b[0x111], b[0x112] = 80, 80, 80 // AC -20 (60 - AC)
				b[0xA3], b[0xA5] = 6, 12                 // unarmed damage 6d12
			}
			if err := os.WriteFile(filepath.Join(*out, name+ext), b, 0o644); err != nil {
				log.Fatal(err)
			}
		}
	}
}
