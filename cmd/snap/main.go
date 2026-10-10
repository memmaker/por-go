// Command snap renders engine screens to PNG without a window (dev check).
//
//	snap DATA OUTDIR [title|party|view [SAVE-LETTER]]
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"por/ega"
	"por/engine"
)

func dump(s *ega.Screen, name string) {
	img := image.NewRGBA(image.Rect(0, 0, ega.W, ega.H))
	for i, c := range s.Pix {
		img.Set(i%ega.W, i/ega.W, s.RGBA(c))
	}
	f, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: snap DATA OUTDIR [title|party|view [A-J]]")
	}
	data, out := os.Args[1], os.Args[2]
	mode := "title"
	if len(os.Args) > 3 {
		mode = os.Args[3]
	}
	scr := ega.New()
	e, err := engine.New(data, scr)
	if err != nil {
		log.Fatal(err)
	}
	switch mode {
	case "title":
		e.InitGraphics()
		e.DrawPicture(0, 0, e.LoadPic("TITLE", 1), 0)
		dump(scr, out+"/title1.png")
		e.DrawPicture(0, 0, e.LoadPic("TITLE", 2), 0)
		dump(scr, out+"/title2.png")
		scr.PushKey(13)
		e.Credits()
		dump(scr, out+"/credits.png")
	case "party":
		e.InitGraphics()
		go e.PartyMenu()
		time.Sleep(300 * time.Millisecond)
		dump(scr, out+"/party.png")
		scr.PushKey('E')
		time.Sleep(300 * time.Millisecond)
		dump(scr, out+"/party_exit.png")
	case "game":
		// snap DATA OUT game SLOT KEYS: load a saved game through the party
		// menu, then type KEYS, one shot per key: ^ v < > arrows, | Enter,
		// @ # $ gamepad A/Y/X, [ ] ( ) d-pad up/down/left/right, !N F1..F6, and the
		// mouse: %cX,Y; click at text cell X,Y, %hX,Y; point there, %u %d
		// wheel up/down, %r right button, _ waits; %e is Esc, %l Ctrl-L. SLOT "-" stays in the
		// party menu. Each shot's hint line goes to game_NN.txt.
		slot, keys := "A", ""
		if len(os.Args) > 4 {
			slot = os.Args[4]
		}
		if len(os.Args) > 5 {
			keys = os.Args[5]
		}
		e.Init()
		if len(os.Args) > 6 {
			e.DevAction = e.DevScene(os.Args[6])
		}
		go e.Play()
		if slot != "-" {
			scr.PushKey('L')
			scr.PushKey(uint16(slot[0]))
		}
		time.Sleep(1500 * time.Millisecond)
		dump(scr, out+"/game_00.png")
		shot := 0
		rs := []rune(keys)
		for j := 0; j < len(rs); j++ {
			k := rs[j]
			code := uint16(k)
			if k == '%' && j+1 < len(rs) {
				j++
				var x, y int
				cell := func() {
					n, _ := fmt.Sscanf(string(rs[j+1:]), "%d,%d;", &x, &y)
					if n != 2 {
						log.Fatalf("bad mouse token at %q", string(rs[j-1:]))
					}
					for rs[j] != ';' {
						j++
					}
				}
				switch rs[j] {
				case 'c':
					cell()
					code = engine.MouseKey(engine.MouseLeft, x, y)
				case 'h':
					cell()
					scr.SetMouse(x, y, true)
					code = 0
				case 'u':
					code = engine.MouseKey(engine.MouseWheelUp, 0, 0)
				case 'd':
					code = engine.MouseKey(engine.MouseWheelDown, 0, 0)
				case 'r':
					code = engine.MouseKey(engine.MouseRight, 0, 0)
				case 'e':
					code = 27
				case 'l':
					code = 0x0C // Ctrl-L: next font
				}
			}
			switch k {
			case '^':
				code = 0x4800
			case 'v':
				code = 0x5000
			case '<':
				code = 0x4B00
			case '>':
				code = 0x4D00
			case '|':
				code = 13
			case '_':
				code = 0 // just wait
			case '{':
				code = 0x4700 // Home: previous member
			case '}':
				code = 0x4F00 // End: next member
			case '[':
				code = engine.KeyPadUp
			case ']':
				code = engine.KeyPadDown
			case '(':
				code = engine.KeyPadLeft
			case ')':
				code = engine.KeyPadRight
			case '@':
				code = engine.KeyPadSelect
			case '#':
				code = engine.KeyPadMinimap
			case '$':
				code = engine.KeyPadOverview
			case '!':
				j++
				code = engine.KeyF1 + uint16(rs[j]-'1')<<8 // !N: F1..F6
			}
			if code != 0 {
				scr.PushKey(code)
			}
			time.Sleep(700 * time.Millisecond)
			shot++
			dump(scr, fmt.Sprintf("%s/game_%02d.png", out, shot))
			os.WriteFile(fmt.Sprintf("%s/game_%02d.txt", out, shot), []byte(scr.Hint()+"\n"), 0o644)
		}
	case "view":
		slot := "A"
		if len(os.Args) > 4 {
			slot = os.Args[4]
		}
		e.Init()
		scr.PushKey(uint16(slot[0]))
		e.LoadGame()
		for _, v := range e.DevViews() {
			v.Draw()
			dump(scr, out+"/view_"+v.Name+".png")
		}
	}
}
