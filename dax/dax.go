// Package dax reads SSI Gold Box .DAX archives (DOS Pool of Radiance layout).
//
// File layout: u16 index size, then index entries of 9 bytes
// (u8 id, i32 offset from data start, u16 raw size, u16 stored size),
// then block data. Blocks are run-length coded:
//
//	n = int8(next)
//	n >= 0: copy the following n+1 bytes
//	n <  0: repeat the following byte -n times
package dax

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Entry is one index record.
type Entry struct {
	ID      int
	Offset  int32
	RawSize uint16
	Stored  uint16
}

// File is a loaded archive. Blocks are decoded on first access and cached.
type File struct {
	Entries []Entry
	data    []byte
	cache   map[int][]byte
	mu      sync.Mutex
}

// Open reads and indexes a DAX file.
func Open(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	n := int(binary.LittleEndian.Uint16(b))
	f := &File{data: b[2+n:], cache: map[int][]byte{}}
	for i := 2; i+9 <= 2+n; i += 9 {
		f.Entries = append(f.Entries, Entry{
			ID:      int(b[i]),
			Offset:  int32(binary.LittleEndian.Uint32(b[i+1:])),
			RawSize: binary.LittleEndian.Uint16(b[i+5:]),
			Stored:  binary.LittleEndian.Uint16(b[i+7:]),
		})
	}
	return f, nil
}

// Block returns the decoded block with the given id (a copy), or nil.
func (f *File) Block(id int) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	if b, ok := f.cache[id]; ok {
		return append([]byte(nil), b...)
	}
	for _, e := range f.Entries {
		if e.ID != id {
			continue
		}
		src := f.data[e.Offset : int(e.Offset)+int(e.Stored)]
		var out []byte
		if e.RawSize == 0 {
			out = append([]byte(nil), src...)
		} else {
			out = Decode(src, int(e.RawSize))
		}
		f.cache[id] = out
		return append([]byte(nil), out...)
	}
	return nil
}

// Decode expands the RLE stream. Output is capped at rawSize; the original
// decoded into a fixed buffer, so over-long blocks are truncated, not errors.
func Decode(src []byte, rawSize int) []byte {
	out := make([]byte, 0, rawSize)
	for i := 0; i < len(src) && len(out) < rawSize; {
		n := int(int8(src[i]))
		if n >= 0 {
			end := i + 1 + n + 1
			if end > len(src) {
				end = len(src)
			}
			out = append(out, src[i+1:end]...)
			i = end
		} else {
			if i+1 < len(src) {
				for k := 0; k < -n; k++ {
					out = append(out, src[i+1])
				}
			}
			i += 2
		}
	}
	if len(out) > rawSize {
		out = out[:rawSize]
	}
	return out
}

// Loader resolves archive names case-insensitively inside a data directory
// and keeps opened files cached, like the original's single-file cache but
// without the reload cost.
type Loader struct {
	Dir   string
	mu    sync.Mutex
	files map[string]*File
	names map[string]string // lower-case name -> real file name
}

// NewLoader scans dir for files.
func NewLoader(dir string) (*Loader, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	l := &Loader{Dir: dir, files: map[string]*File{}, names: map[string]string{}}
	for _, e := range ents {
		l.names[strings.ToLower(e.Name())] = e.Name()
	}
	return l, nil
}

// Path returns the real path for a case-insensitive file name, or "".
func (l *Loader) Path(name string) string {
	if real, ok := l.names[strings.ToLower(name)]; ok {
		return filepath.Join(l.Dir, real)
	}
	return ""
}

// Exists reports whether a data file is present.
func (l *Loader) Exists(name string) bool { return l.Path(name) != "" }

// File opens (or returns cached) archive "name" (e.g. "TITLE.DAX").
func (l *Loader) File(name string) *File {
	key := strings.ToLower(name)
	if !strings.HasSuffix(key, ".dax") {
		key += ".dax"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if f, ok := l.files[key]; ok {
		return f
	}
	p := l.Path(key)
	if p == "" {
		l.files[key] = nil
		return nil
	}
	f, err := Open(p)
	if err != nil {
		f = nil
	}
	l.files[key] = f
	return f
}

// Block loads block id from archive name; nil if either is missing.
func (l *Loader) Block(name string, id int) []byte {
	f := l.File(name)
	if f == nil {
		return nil
	}
	return f.Block(id)
}
