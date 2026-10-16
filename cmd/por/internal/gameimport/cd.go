package gameimport

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

const isoSector = 2048

// listCD lists an ISO 9660 image, or a raw BIN track (2352-byte sectors, mode 1 or mode 2
// form 1) by its 2048 user bytes per sector. Not an image: no files, no error.
func listCD(r io.ReaderAt, size int64) ([]file, error) {
	if !isISO(r) {
		var s [16]byte
		if _, err := r.ReadAt(s[:], 16*2352); err != nil || string(s[:12]) != "\x00\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\x00" {
			return nil, nil
		}
		raw := rawCD{r, 16}
		if s[15] == 2 {
			raw.skip = 24 // mode 2 form 1: an 8-byte subheader after the header
		}
		if !isISO(raw) {
			return nil, nil
		}
		r = raw
	}
	pvd := make([]byte, isoSector)
	r.ReadAt(pvd, 16*isoSector)
	var out []file
	return out, isoWalk(r, binary.LittleEndian.Uint32(pvd[156+2:]), binary.LittleEndian.Uint32(pvd[156+10:]), "", &out, 0)
}

func isISO(r io.ReaderAt) bool {
	var b [6]byte
	_, err := r.ReadAt(b[:], 16*isoSector)
	return err == nil && string(b[:]) == "\x01CD001"
}

// isoWalk appends the files of the directory at lba (size bytes) and below it, names without
// the ";1" version.
func isoWalk(r io.ReaderAt, lba, size uint32, dir string, out *[]file, depth int) error {
	if depth > 16 {
		return errors.New("an ISO directory tree too deep")
	}
	buf := make([]byte, size)
	if _, err := r.ReadAt(buf, int64(lba)*isoSector); err != nil && err != io.EOF {
		return err
	}
	for off := 0; off < len(buf); {
		n := int(buf[off])
		if n == 0 { // no record crosses a sector: the rest of this one is padding
			off = (off/isoSector + 1) * isoSector
			continue
		}
		if n < 34 || off+n > len(buf) || 33+int(buf[off+32]) > n {
			return errors.New("a broken ISO directory")
		}
		rec := buf[off : off+n]
		off += n
		name := string(rec[33 : 33+int(rec[32])])
		if name == "\x00" || name == "\x01" { // . and ..
			continue
		}
		if i := strings.IndexByte(name, ';'); i >= 0 {
			name = name[:i]
		}
		name = dir + strings.TrimSuffix(name, ".")
		at, n32 := binary.LittleEndian.Uint32(rec[2:]), binary.LittleEndian.Uint32(rec[10:])
		if rec[25]&2 != 0 {
			if err := isoWalk(r, at, n32, name+"/", out, depth+1); err != nil {
				return err
			}
			continue
		}
		*out = append(*out, file{name, int64(n32), func() (io.ReaderAt, error) {
			return io.NewSectionReader(r, int64(at)*isoSector, int64(n32)), nil
		}})
	}
	return nil
}

// rawCD reads a raw track's user data: 2048 bytes at skip in every 2352-byte sector.
type rawCD struct {
	r    io.ReaderAt
	skip int64
}

func (c rawCD) ReadAt(p []byte, off int64) (int, error) {
	n := 0
	for n < len(p) {
		sec, in := off/isoSector, off%isoSector
		m, err := c.r.ReadAt(p[n:min(len(p), n+int(isoSector-in))], sec*2352+c.skip+in)
		n += m
		off += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
