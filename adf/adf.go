// Package adf reads the files of an Amiga OFS/FFS floppy image (.adf).
package adf

import (
	"encoding/binary"
	"errors"
)

const bsize = 512

// File is one file of the disk, Path relative to the root ("dir/name").
type File struct {
	Path string
	Data []byte
}

// Read returns the volume name and every file of an 880K/1760K image.
func Read(img []byte) (string, []File, error) {
	if len(img) < 1760*bsize || string(img[:3]) != "DOS" {
		return "", nil, errors.New("not an Amiga DOS disk")
	}
	ffs := img[3]&1 == 1
	nblk := len(img) / bsize
	blk := func(n uint32) []byte {
		if int(n) >= nblk {
			return nil
		}
		return img[int(n)*bsize : int(n+1)*bsize]
	}
	u32 := func(b []byte, off int) uint32 { return binary.BigEndian.Uint32(b[off:]) }
	name := func(b []byte) string {
		n := int(b[432])
		if n > 30 {
			n = 30
		}
		return string(b[433 : 433+n])
	}
	root := blk(uint32(nblk / 2))
	if u32(root, 0) != 2 || u32(root, bsize-4) != 1 {
		return "", nil, errors.New("no root block")
	}
	var files []File
	seen := map[uint32]bool{}
	var walk func(dir []byte, prefix string)
	walk = func(dir []byte, prefix string) {
		for i := 0; i < 72; i++ {
			for h := u32(dir, 24+4*i); h != 0 && !seen[h]; {
				seen[h] = true
				b := blk(h)
				if b == nil {
					break
				}
				p := prefix + name(b)
				switch int32(u32(b, bsize-4)) {
				case 2: // directory
					walk(b, p+"/")
				case -3: // file
					files = append(files, File{p, readFile(b, blk, u32, ffs)})
				}
				h = u32(b, bsize-16) // hash chain
			}
		}
	}
	walk(root, "")
	return name(root), files, nil
}

func readFile(hdr []byte, blk func(uint32) []byte, u32 func([]byte, int) uint32, ffs bool) []byte {
	size := int(u32(hdr, bsize-188))
	out := make([]byte, 0, size)
	for h := hdr; h != nil && len(out) < size; {
		// data block pointers run backwards from offset 308
		n := int(u32(h, 8))
		for i := 0; i < n && i < 72 && len(out) < size; i++ {
			d := blk(u32(h, 308-4*i))
			if d == nil {
				return out
			}
			if ffs {
				out = append(out, d[:min(bsize, size-len(out))]...)
			} else {
				out = append(out, d[24:24+min(int(u32(d, 12)), size-len(out), 488)]...)
			}
		}
		ext := u32(h, bsize-8)
		if ext == 0 {
			break
		}
		h = blk(ext)
	}
	return out
}
