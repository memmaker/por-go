package main

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"por/dax"
	"por/ega"
)

// importGame (-import, for goplay) finds a Pool of Radiance install
// anywhere under src (the folder holding START.EXE and TITLE.DAX), copies
// the files the engine reads into dst, its SAVE and FONTS folders, and
// writes dst/COVER.PNG from the title picture for the floppy label.
func importGame(src, dst string) error {
	var dir string
	filepath.WalkDir(src, func(p string, d fs.DirEntry, _ error) error {
		if d != nil && strings.EqualFold(d.Name(), "START.EXE") && has(filepath.Dir(p), "TITLE.DAX") && has(filepath.Dir(p), "8X8D1.DAX") {
			dir = filepath.Dir(p)
			return fs.SkipAll
		}
		return nil
	})
	if dir == "" {
		return errors.New("no Pool of Radiance found")
	}
	if err := copyFiles(dir, dst, func(name string) bool {
		ext := strings.ToUpper(filepath.Ext(name))
		return ext == ".DAX" || strings.EqualFold(name, "START.EXE") || strings.EqualFold(name, "POOL.CFG") || strings.EqualFold(name, "JOURNAL.TXT") ||
			strings.EqualFold(name, "ITEMS") // the base item table (engine/init.go readItems)
	}); err != nil {
		return err
	}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if e.IsDir() && strings.EqualFold(e.Name(), "SAVE") {
			if err := copyFiles(filepath.Join(dir, e.Name()), filepath.Join(dst, "SAVE"), func(string) bool { return true }); err != nil {
				return err
			}
		}
		if e.IsDir() && strings.EqualFold(e.Name(), "FONTS") { // fonts for Ctrl-L (engine/fonts.go)
			if err := copyFiles(filepath.Join(dir, e.Name()), filepath.Join(dst, "FONTS"), func(n string) bool {
				return strings.EqualFold(filepath.Ext(n), ".f8")
			}); err != nil {
				return err
			}
		}
	}
	return writeCover(dst)
}

// copyFiles copies the plain files of dir that keep accepts, upper-cased
// like the DOS originals.
func copyFiles(dir, dst string, keep func(string) bool) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if e.IsDir() || !keep(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, strings.ToUpper(e.Name())), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func has(dir, name string) bool {
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if strings.EqualFold(e.Name(), name) {
			return true
		}
	}
	return false
}

// writeCover renders TITLE.DAX block 2, the title picture.
func writeCover(dst string) error {
	l, err := dax.NewLoader(dst)
	if err != nil {
		return err
	}
	b := l.Block("TITLE", 2)
	if b == nil {
		return errors.New("TITLE.DAX has no title picture")
	}
	im := dax.ParseImage(b, false, 0)
	pal := make(color.Palette, 16)
	for i, c := range ega.Palette {
		pal[i] = color.RGBA{c[0], c[1], c[2], 255}
	}
	img := image.NewPaletted(image.Rect(0, 0, ega.W, ega.H), pal)
	d, w := im.Frame(0), im.Width*8
	for y := 0; y < im.Height && y < ega.H; y++ {
		for x := 0; x < w && x < ega.W; x++ {
			img.SetColorIndex(x, y, d[y*w+x]&15)
		}
	}
	f, err := os.Create(filepath.Join(dst, "COVER.PNG"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
