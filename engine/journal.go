package engine

import (
	"embed"
	"image"
	_ "image/png"
	"regexp"
	"strconv"
	"strings"

	"por/ega"
)

// Journal texts (extra): the game often only names an
// entry of the printed Adventurer's Journal ("SEE JOURNAL ENTRY 26", "YOU
// OVERHEAR TAVERN TALE 5", the proclamations at City Hall). When the text
// window names one, the entry is shown right there, with its pictures. The
// texts and pictures (pre-scaled to the screen, one shared 16-colour
// palette) are built in from journal/.

//go:embed journal
var journalFS embed.FS

// journalRef finds the references in the text window: "ENTRY 7", "ENTRIES
// 23 AND 14", "TAVERN TALE 5", "PROCLAMATION LIX.", "PROCLAMATIONS LXIV,
// LXXVIII, CIX, AND LIX.".
var journalRef = regexp.MustCompile(`\bENTRY\s*(\d+)|\bENTRIES\s*(\d+)\s+AND\s+(\d+)|\bTAVERN TALE\s*(\d+)|\bPROCLAMATIONS?\s*([IVXLCDM]+(?:(?:,\s*|\s+)(?:AND\s+)?[IVXLCDM]+)*)\b`)

var romanList = regexp.MustCompile(`[IVXLCDM]+`)

// journalRefs lists the titles of the texts s names, in order.
func journalRefs(s string) []string {
	var t []string
	for _, m := range journalRef.FindAllStringSubmatch(s, -1) {
		switch {
		case m[1] != "":
			t = append(t, "Journal Entry "+m[1])
		case m[2] != "":
			t = append(t, "Journal Entry "+m[2], "Journal Entry "+m[3])
		case m[4] != "":
			t = append(t, "Tavern Tale "+m[4])
		case m[5] != "":
			for _, r := range romanList.FindAllString(strings.ReplaceAll(m[5], "AND", ""), -1) {
				t = append(t, "Proclamation "+r)
			}
		}
	}
	return t
}

// loadJournal reads JOURNAL.TXT: "[Title]" starts a text, a blank line a
// paragraph, "<FILE.PNG>" a picture, "#" lines are comments.
func loadJournal() map[string][]string {
	b, _ := journalFS.ReadFile("journal/JOURNAL.TXT")
	j := map[string][]string{}
	var title string
	for _, l := range strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n") {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, "#"):
		case strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]"):
			title = l[1 : len(l)-1]
			j[title] = nil
		case title != "":
			j[title] = append(j[title], l)
		}
	}
	return j
}

// journalPrinted follows what opPrint put into the text window (clear: it
// started over) and shows the journal texts it names for the first time.
func (e *Engine) journalPrinted(s string, clear bool) {
	if clear {
		e.ext.printed, e.ext.refsShown = "", 0
	}
	e.ext.printed += s
	refs := journalRefs(strings.ToUpper(e.ext.printed))
	if len(refs) <= e.ext.refsShown {
		return
	}
	newRefs := refs[e.ext.refsShown:]
	e.ext.refsShown = len(refs)
	j := loadJournal()
	var lines []string
	for _, t := range newRefs {
		paras, ok := j[t]
		if !ok {
			continue
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "\x00"+t)
		for _, p := range paras {
			switch {
			case p == "":
				lines = append(lines, "")
			case strings.HasPrefix(p, "<") && strings.HasSuffix(p, ">"):
				if im := journalPicture(p[1 : len(p)-1]); im != nil {
					lines = append(lines, "\x01"+p[1:len(p)-1])
					_, h := pictureSize(im)
					for range (h+7)/8 - 1 {
						lines = append(lines, "\x02") // rows the picture covers
					}
				}
			case len(p) <= jrWidth: // as written: keeps the spacing of tables (entry 53)
				lines = append(lines, p)
			default:
				lines = append(lines, wrapText(p, jrWidth)...)
			}
		}
	}
	if len(lines) > 0 {
		e.showJournal(lines)
	}
}

// The pager: the text inside a full-screen box, jrRows lines a page.
const (
	jrWidth = 0x24
	jrRows  = 0x15
)

// wrapText breaks s into lines of at most w characters at spaces.
func wrapText(s string, w int) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		for len(word) > w { // (no word is that long, but never loop)
			out = append(out, word[:w])
			word = word[w:]
		}
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	return append(out, line)
}

// showJournal pages through lines ("\x00" marks a title) until the last
// page or Esc, then puts the screen back.
func (e *Engine) showJournal(lines []string) {
	e.ext.modal++
	e.ext.paging = true
	defer func() { e.ext.modal--; e.ext.paging = false }()
	restore := e.saveScreen()
	pages := journalPages(lines)
	for p, page := range pages {
		e.setOverlays(nil)
		e.DrawBoxTitled(1, 1, 0x26, 0x17, 0, 0xF, "Adventurer's Journal")
		e.ClearCells(2, 2, 0x25, 0x17)
		for i, l := range page {
			switch {
			case strings.HasPrefix(l, "\x00"):
				e.DisplayString(2, 2+i, 0xE, l[1:])
			case strings.HasPrefix(l, "\x01"):
				e.drawPicture(journalPicture(l[1:]), 2*8+jrWidth*4, (2+i)*8)
			case l != "\x02":
				e.DisplayString(2, 2+i, 0xF, l)
			}
		}
		more := "press a key"
		if p < len(pages)-1 {
			more = "more: " + strconv.Itoa(p+1) + "/" + strconv.Itoa(len(pages))
		}
		e.DisplayString(0x25-len(more), 0x17, 0xA, more)
		if e.ReadKey() == 27 {
			break
		}
	}
	e.setOverlays(nil)
	restore()
}

func (e *Engine) setOverlays(o []ega.Overlay) {
	e.S.Lock()
	e.S.Overlays = o
	e.S.Unlock()
}

// journalPages cuts lines into pages of at most jrRows; a page does not
// start with a blank line or end with a title, and a picture is not split.
func journalPages(lines []string) [][]string {
	var pages [][]string
	for len(lines) > 0 {
		for len(lines) > 0 && lines[0] == "" {
			lines = lines[1:]
		}
		n := min(jrRows, len(lines))
		for n < len(lines) && n > 1 && lines[n] == "\x02" { // back to the picture's first row
			n--
		}
		if n < len(lines) && n > 1 && strings.HasPrefix(lines[n-1], "\x00") {
			n--
		}
		if n > 0 {
			pages = append(pages, lines[:n])
		}
		lines = lines[n:]
	}
	return pages
}

// journalPicture decodes a picture from journal/ once; nil if missing.
func journalPicture(name string) image.Image {
	if im, ok := pictures[name]; ok {
		return im
	}
	var im image.Image
	if f, err := journalFS.Open("journal/" + name); err == nil {
		im, _, _ = image.Decode(f)
		f.Close()
	}
	pictures[name] = im
	return im
}

var pictures = map[string]image.Image{} // the engine goroutine's alone

// pictureSize is the screen area a picture covers: the scans are 300 dpi,
// shown at about 90 screen pixels an inch, at most the text width wide
// and 19 rows high (with a title and a line it fills a page).
func pictureSize(im image.Image) (w, h int) {
	b := im.Bounds()
	s := min(0.3, float64(jrWidth*8)/float64(b.Dx()), 152/float64(b.Dy()))
	return int(float64(b.Dx()) * s), int(float64(b.Dy()) * s)
}

// drawPicture puts a journal picture centred on x, top at y: paper white
// on the screen, the scan itself over it at the window's resolution.
func (e *Engine) drawPicture(im image.Image, x, y int) {
	if im == nil {
		return
	}
	w, h := pictureSize(im)
	r := image.Rect(x-w/2, y, x-w/2+w, y+h)
	e.S.Lock()
	e.S.Fill(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, 15)
	e.S.Overlays = append(e.S.Overlays, ega.Overlay{Img: im, At: r})
	e.S.Unlock()
}
