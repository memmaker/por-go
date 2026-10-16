package gameimport

// Inno Setup, the subset that gets files out of a GOG installer: versions 5.5 to 6.4, stored,
// zlib, bzip2, LZMA1 and LZMA2 chunks, data inside the .exe or in setup-N.bin slices beside
// it. The layout follows innoextract's description of the format (setup/header.cpp and
// friends): the loader table points at a 64-byte version string and two CRC'd blocks, the
// first holding the setup header and its entry lists (we read through them to the file
// entries), the second the data entries saying where each file's bytes lie in which chunk.

import (
	"bytes"
	"compress/bzip2"
	"compress/zlib"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"path"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/ulikunitz/xz/lzma"
)

func iv(a, b, c, d uint32) uint32 { return a<<24 | b<<16 | c<<8 | d }

// listInno lists the files of the Inno Setup installer r (named name, among siblings for its
// slices). Not an installer: no files, no error.
func listInno(r io.ReaderAt, size int64, name string, siblings []file) ([]file, error) {
	head := make([]byte, min(size, 16<<20))
	n, _ := r.ReadAt(head, 0)
	head = head[:n]
	at := bytes.Index(head, []byte("rDlPtS\xcd\xe6\xd7{\x0b*"))
	if at < 0 {
		at = bytes.Index(head, []byte("nS5W7dT\x83\xaa\x1b\x0fj"))
	}
	if at < 0 || at+12+28 > len(head) {
		return nil, nil
	}
	t := head[at+12:]
	le := binary.LittleEndian
	if rev := le.Uint32(t); rev != 1 {
		return nil, fmt.Errorf("%s: Inno Setup loader table revision %d, not supported", name, rev)
	}
	hdrOff, dataOff := int64(le.Uint32(t[20:])), int64(le.Uint32(t[24:]))

	vs := make([]byte, 64)
	if _, err := r.ReadAt(vs, hdrOff); err != nil {
		return nil, err
	}
	vstr := string(bytes.TrimRight(vs, "\x00"))
	var a, b, c, d uint32
	i := strings.IndexByte(vstr, '(')
	if !strings.HasPrefix(vstr, "Inno Setup Setup Data") || i < 0 {
		return nil, fmt.Errorf("%s: unknown Inno Setup data version %q", name, vstr)
	}
	fmt.Sscanf(vstr[i:], "(%d.%d.%d.%d", &a, &b, &c, &d)
	v := iv(a, b, c, d)
	uni := strings.Contains(vstr, "(u)") || strings.Contains(vstr, "(U)") || a >= 6
	if v < iv(5, 5, 0, 0) {
		return nil, fmt.Errorf("%s: %s is older than Inno Setup 5.5, not supported", name, vstr)
	}
	fail := func(err error) ([]file, error) { return nil, fmt.Errorf("%s (%s): %w", name, vstr, err) }

	b1, next, err := innoBlock(r, hdrOff+64)
	if err != nil {
		return fail(err)
	}
	b2, _, err := innoBlock(r, next)
	if err != nil {
		return fail(err)
	}

	// The setup header, skipped field by field up to what we need.
	h := &rd{b: b1, uni: uni}
	nstr := 27 // app name ... close applications filter
	if v >= iv(5, 5, 6, 0) {
		nstr++
	}
	if v >= iv(5, 6, 1, 0) {
		nstr += 2
	}
	if v >= iv(6, 3, 0, 0) {
		nstr += 2
	}
	h.strs(nstr + 4) // + license, info before, info after, compiled code
	if !uni {
		h.skip(32) // lead bytes
	}
	var cnt [16]int // languages, messages, permissions, types, components, tasks, directories, files, data entries, ...
	for i := range cnt {
		cnt[i] = int(h.u32())
	}
	h.skip(20) // windows version range
	if v < iv(6, 4, 0, 1) {
		h.skip(8) // back colors
	}
	if v < iv(5, 5, 7, 0) {
		h.skip(4) // image back color
	}
	if v >= iv(6, 0, 0, 0) {
		h.skip(9) // wizard style, resize percents
	}
	if v >= iv(5, 5, 7, 0) {
		h.skip(1) // image alpha format
	}
	if v >= iv(6, 4, 0, 0) {
		h.skip(4 + 44) // password hash, salt
	} else {
		h.skip(20 + 8)
	}
	h.skip(8) // extra disk space
	slicesPerDisk := int(h.u32())
	h.skip(3) // uninstall log mode, dir exists warning, privileges required
	if v >= iv(5, 7, 0, 0) {
		h.skip(1) // privileges override
	}
	h.skip(2) // language dialog, detection
	compression := h.u8()
	if v < iv(6, 3, 0, 0) {
		h.skip(2) // architectures
	}
	h.skip(2 + 8) // disable dir/group page, uninstall display size
	h.flags(headerFlagCount(v, uni))

	langFixed := 4 + 16 + 1
	if !uni {
		langFixed += 4
	}
	for _, e := range []struct{ strs, fixed int }{
		{10, langFixed},             // languages
		{2, 4},                      // messages
		{1, 0},                      // permissions
		{4, 20 + 1 + 1 + 8},         // types
		{5, 8 + 4 + 1 + 20 + 1 + 8}, // components
		{6, 4 + 1 + 20 + 1},         // tasks
		{7, 4 + 20 + 2 + 1},         // directories
	} {
		for range cnt[0] {
			h.strs(e.strs)
			h.skip(e.fixed)
		}
		cnt = [16]int(append(cnt[1:], 0))
	}
	// GOG Galaxy installers store a file as parts: the first entry's BeforeInstall is
	// before_install('md5', 'real name', parts), each part's AfterInstall
	// after_install('md5', size, unpacked size), and each part's bytes are zlib streams.
	type entry struct {
		dest   string
		locs   []uint32
		size   int64
		galaxy bool
		md5    string
	}
	var ents []*entry
	var cur *entry
	parts := 0
	for range cnt[0] { // files
		h.str() // source
		dest := h.str()
		h.strs(6) // font, assembly, components, tasks, languages, check
		after, before := h.str(), h.str()
		h.skip(20)
		loc := h.u32()
		h.skip(4 + 8 + 2 + 4 + 1) // attributes, external size, permission, flags, type
		if a := call(before, "before_install", "before_install_dependency"); a != nil {
			cur = &entry{dest: dest, galaxy: true, md5: a[0]}
			if len(a) > 1 && a[1] != "" {
				cur.dest = a[1]
			}
			parts = 1
			if len(a) > 2 {
				fmt.Sscan(a[2], &parts)
			}
			ents = append(ents, cur)
		}
		if a := call(after, "after_install", "after_install_dependency"); a != nil && parts > 0 && len(a) > 2 {
			var n int64
			fmt.Sscan(a[2], &n)
			cur.locs, cur.size = append(cur.locs, loc), cur.size+n
			parts--
			continue
		}
		parts = 0
		ents = append(ents, &entry{dest: dest, locs: []uint32{loc}})
	}
	if h.err != nil {
		return fail(h.err)
	}

	type data struct {
		slice, chunkOff  uint32
		off, size, csize uint64
		sum              []byte
		flags            uint16
	}
	g := &rd{b: b2}
	var dat []data
	for range cnt[1] {
		var e data
		e.slice = g.u32()
		g.u32() // last slice
		e.chunkOff = g.u32()
		e.off, e.size, e.csize = g.u64(), g.u64(), g.u64()
		if v >= iv(6, 4, 0, 0) {
			e.sum = g.bytes(32)
		} else {
			e.sum = g.bytes(20)
		}
		g.skip(16) // file time, file version
		e.flags = uint16(g.u8()) | uint16(g.u8())<<8
		if v >= iv(6, 3, 0, 0) {
			g.skip(1) // sign mode
		}
		dat = append(dat, e)
	}
	if g.err != nil {
		return fail(g.err)
	}

	// chunk returns a chunk's stored bytes, read on from slice to slice.
	chunk := func(e data) (io.Reader, error) {
		var src io.Reader
		if dataOff != 0 {
			src = io.NewSectionReader(r, dataOff+int64(e.chunkOff), 4+int64(e.csize))
		} else {
			base := strings.TrimSuffix(path.Base(name), path.Ext(name))
			src = &sliceReader{n: int(e.slice), pos: int64(e.chunkOff), open: func(n int) (io.ReaderAt, error) {
				sn := fmt.Sprintf("%s-%d.bin", base, n+1)
				if slicesPerDisk > 1 {
					sn = fmt.Sprintf("%s-%d%c.bin", base, n/slicesPerDisk+1, 'a'+n%slicesPerDisk)
				}
				for _, f := range siblings {
					if path.Dir(f.name) == path.Dir(name) && strings.EqualFold(path.Base(f.name), sn) {
						return f.open()
					}
				}
				return nil, errors.New("missing installer slice " + sn)
			}}
		}
		var magic [4]byte
		if _, err := io.ReadFull(src, magic[:]); err != nil {
			return nil, err
		}
		if string(magic[:]) != "zlb\x1a" {
			return nil, errors.New("bad chunk magic")
		}
		return io.LimitReader(src, int64(e.csize)), nil
	}

	// extract returns the bytes of data entry e.
	extract := func(e data) ([]byte, error) {
		if e.flags&(1<<6) != 0 {
			return nil, errors.New("encrypted (password-protected) installer, not supported")
		}
		if e.flags&(1<<4) != 0 {
			return nil, errors.New("x86 call filter not supported")
		}
		src, err := chunk(e)
		if err != nil {
			return nil, err
		}
		if e.flags&(1<<7) != 0 {
			// ponytail: a solid chunk is decompressed from its start for every file read
			// out of it; cache the decoder per chunk if a port wants many files from one.
			if src, err = decompress(src, compression, e.off+e.size); err != nil {
				return nil, err
			}
		}
		if _, err := io.CopyN(io.Discard, src, int64(e.off)); err != nil {
			return nil, err
		}
		buf := make([]byte, e.size)
		if _, err := io.ReadFull(src, buf); err != nil {
			return nil, err
		}
		var sum []byte
		if len(e.sum) == 32 {
			s := sha256.Sum256(buf)
			sum = s[:]
		} else {
			s := sha1.Sum(buf)
			sum = s[:]
		}
		if !bytes.Equal(sum, e.sum) {
			return nil, errors.New("checksum mismatch")
		}
		return buf, nil
	}

	var out []file
	for _, en := range ents {
		if len(en.locs) == 0 || int(en.locs[0]) >= len(dat) {
			continue // an external file, not in the installer
		}
		if !en.galaxy {
			en.size = int64(dat[en.locs[0]].size)
		}
		fname := strings.TrimPrefix(strings.ReplaceAll(en.dest, `\`, "/"), "{app}/")
		out = append(out, file{fname, en.size, func() (io.ReaderAt, error) {
			var all []byte
			for _, l := range en.locs {
				if int(l) >= len(dat) {
					return nil, fmt.Errorf("%s/%s: bad data location", name, fname)
				}
				b, err := extract(dat[l])
				if err == nil && en.galaxy {
					var z io.Reader
					if z, err = zlib.NewReader(bytes.NewReader(b)); err == nil {
						b, err = io.ReadAll(z)
					}
				}
				if err != nil {
					return nil, fmt.Errorf("%s/%s: %w", name, fname, err)
				}
				all = append(all, b...)
			}
			if en.galaxy && len(en.md5) == 32 && fmt.Sprintf("%x", md5.Sum(all)) != strings.ToLower(en.md5) {
				return nil, fmt.Errorf("%s/%s: checksum mismatch", name, fname)
			}
			return bytes.NewReader(all), nil
		}})
	}
	return out, nil
}

// call returns the arguments of a Pascal call to one of names in code ('quoted' strings
// unquoted), or nil when code is no such call.
func call(code string, names ...string) []string {
	code = strings.TrimSpace(code)
	i := strings.IndexByte(code, '(')
	if i < 0 || !slices.Contains(names, strings.TrimSpace(code[:i])) {
		return nil
	}
	var args []string
	var arg strings.Builder
	quoted := false
	for j := i + 1; j < len(code); j++ {
		c := code[j]
		switch {
		case c == '\'' && quoted && j+1 < len(code) && code[j+1] == '\'':
			arg.WriteByte(c)
			j++
		case c == '\'':
			quoted = !quoted
		case !quoted && (c == ',' || c == ')'):
			args = append(args, strings.TrimSpace(arg.String()))
			arg.Reset()
			if c == ')' {
				return args
			}
		default:
			arg.WriteByte(c)
		}
	}
	return nil
}

// headerFlagCount is the number of option flags the setup header stores (load_flags).
func headerFlagCount(v uint32, uni bool) int {
	n := 5 + 4 + 1 + 3 + 19 + 4 + 3 // the flags every version from 5.5 has
	if v < iv(6, 4, 0, 1) {
		n += 5 // window flags, back color horizontal
	}
	if v < iv(5, 6, 1, 0) {
		n += 2 // changes associations, changes environment
	}
	if !uni {
		n++
	}
	if v >= iv(5, 5, 7, 0) {
		n++
	}
	if v >= iv(6, 0, 0, 0) {
		n += 3
	}
	if v >= iv(6, 3, 0, 0) {
		n++
	}
	return n
}

// innoBlock reads the CRC'd (and LZMA1-compressed) block at at; next is the offset after it.
func innoBlock(r io.ReaderAt, at int64) (data []byte, next int64, err error) {
	var h [9]byte
	if _, err := r.ReadAt(h[:], at); err != nil {
		return nil, 0, err
	}
	le := binary.LittleEndian
	if crc32.ChecksumIEEE(h[4:]) != le.Uint32(h[:]) {
		return nil, 0, errors.New("header block CRC mismatch")
	}
	raw := make([]byte, le.Uint32(h[4:]))
	if _, err := r.ReadAt(raw, at+9); err != nil {
		return nil, 0, err
	}
	next = at + 9 + int64(len(raw))
	for len(raw) > 0 { // 4096-byte pieces, each after its CRC32
		if len(raw) < 5 {
			return nil, 0, errors.New("truncated header block")
		}
		p := raw[4 : 4+min(4096, len(raw)-4)]
		if crc32.ChecksumIEEE(p) != le.Uint32(raw) {
			return nil, 0, errors.New("header block CRC mismatch")
		}
		data = append(data, p...)
		raw = raw[4+len(p):]
	}
	if h[8] == 0 {
		return data, next, nil
	}
	src, err := decompress(bytes.NewReader(data), 3, 0)
	if err != nil {
		return nil, 0, err
	}
	data, err = io.ReadAll(src)
	if err == io.ErrUnexpectedEOF { // no end marker: the stream just stops
		err = nil
	}
	return data, next, err
}

// decompress decodes src by Inno's compression method (5.3.9+ numbering); need, when not 0,
// is how many bytes the caller reads, which bounds the LZMA dictionary.
func decompress(src io.Reader, method byte, need uint64) (io.Reader, error) {
	dict := func(d uint64) int {
		if need > 0 && d > need {
			d = max(need, lzma.MinDictCap)
		}
		return int(d)
	}
	switch method {
	case 0:
		return src, nil
	case 1:
		return zlib.NewReader(src)
	case 2:
		return bzip2.NewReader(src), nil
	case 3: // 5 property bytes, then a raw LZMA1 stream: give it a .lzma header
		var p [5]byte
		if _, err := io.ReadFull(src, p[:]); err != nil {
			return nil, err
		}
		var hdr [13]byte
		hdr[0] = p[0]
		binary.LittleEndian.PutUint32(hdr[1:], uint32(dict(uint64(binary.LittleEndian.Uint32(p[1:])))))
		size := uint64(1<<64 - 1)
		if need > 0 {
			size = need
		}
		binary.LittleEndian.PutUint64(hdr[5:], size)
		return lzma.NewReader(io.MultiReader(bytes.NewReader(hdr[:]), src))
	case 4: // 1 property byte, then raw LZMA2
		var p [1]byte
		if _, err := io.ReadFull(src, p[:]); err != nil {
			return nil, err
		}
		if p[0] > 40 {
			return nil, errors.New("bad LZMA2 property")
		}
		d := uint64(0xffffffff)
		if p[0] < 40 {
			d = uint64(2|p[0]&1) << (p[0]/2 + 11)
		}
		return lzma.Reader2Config{DictCap: dict(d)}.NewReader2(src)
	}
	return nil, fmt.Errorf("compression method %d not supported", method)
}

// sliceReader reads on through setup-N.bin slices from slice n at pos.
type sliceReader struct {
	n         int
	pos, size int64
	r         io.ReaderAt
	open      func(n int) (io.ReaderAt, error)
}

func (s *sliceReader) Read(p []byte) (int, error) {
	for s.r == nil || s.pos >= s.size {
		if s.r != nil {
			s.n, s.pos, s.r = s.n+1, 12, nil
		}
		r, err := s.open(s.n)
		if err != nil {
			return 0, err
		}
		var h [12]byte
		if _, err := r.ReadAt(h[:], 0); err != nil || string(h[:4]) != "idsk" || h[7] != 0x1a {
			return 0, errors.New("bad installer slice header")
		}
		s.r, s.size = r, int64(binary.LittleEndian.Uint32(h[8:]))
	}
	n, err := s.r.ReadAt(p[:min(int64(len(p)), s.size-s.pos)], s.pos)
	s.pos += int64(n)
	if err == io.EOF && n > 0 {
		err = nil
	}
	return n, err
}

// rd reads the little-endian fields of a header block; past its end it records an error.
type rd struct {
	b   []byte
	uni bool
	err error
}

func (r *rd) bytes(n int) []byte {
	if n < 0 || n > len(r.b) {
		r.err, r.b = errors.New("setup header ends early (unsupported Inno Setup variant?)"), nil
		return make([]byte, max(n, 0))
	}
	p := r.b[:n]
	r.b = r.b[n:]
	return p
}
func (r *rd) skip(n int)  { r.bytes(n) }
func (r *rd) u8() byte    { return r.bytes(1)[0] }
func (r *rd) u32() uint32 { return binary.LittleEndian.Uint32(r.bytes(4)) }
func (r *rd) u64() uint64 { return binary.LittleEndian.Uint64(r.bytes(8)) }

// flags skips a set of n flag bits; 3-byte sets are padded to 4.
func (r *rd) flags(n int) {
	if n = (n + 7) / 8; n == 3 {
		n = 4
	}
	r.skip(n)
}
func (r *rd) strs(n int) {
	for range n {
		r.str()
	}
}

// str reads a length-prefixed string: UTF-16LE in Unicode installers, else Latin-1.
func (r *rd) str() string {
	b := r.bytes(int(r.u32()))
	if !r.uni {
		u := make([]rune, len(b))
		for i, c := range b {
			u[i] = rune(c)
		}
		return string(u)
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u))
}
