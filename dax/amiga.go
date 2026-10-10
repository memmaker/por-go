package dax

import (
	"encoding/binary"
	"sort"
)

// Amiga Pool of Radiance archives: u16 BE index size, 10-byte BE entries
// (u16 id, u32 offset, u16 stored size, u16 raw size), then ByteKiller
// crunched blocks (decruncher at 0x63346 of the Amiga "program").

// OpenAmiga indexes an Amiga .dax file held in b; blocks come out raw.
func OpenAmiga(b []byte) *File {
	n := int(binary.BigEndian.Uint16(b))
	f := &File{data: b[2+n:], cache: map[int][]byte{}}
	for i := 2; i+10 <= 2+n; i += 10 {
		e := Entry{ID: int(binary.BigEndian.Uint16(b[i:])), Offset: int32(binary.BigEndian.Uint32(b[i+2:])), Stored: binary.BigEndian.Uint16(b[i+6:])}
		f.Entries = append(f.Entries, e)
		f.cache[e.ID] = Unbytekill(f.data[e.Offset : int(e.Offset)+int(e.Stored)])
	}
	return f
}

// Unbytekill decrunches a ByteKiller stream, read backwards from its end:
// u32 raw size, u32 checksum, then the bit stream; output is filled from
// the end too. Returns nil on a malformed stream.
func Unbytekill(src []byte) (out []byte) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	p := len(src)
	long := func() uint32 { p -= 4; return binary.BigEndian.Uint32(src[p:]) }
	out = make([]byte, long())
	long() // checksum
	d0 := long()
	bit := func() uint32 {
		c := d0 & 1
		d0 >>= 1
		if d0 == 0 {
			d0 = long()
			c = d0 & 1
			d0 = d0>>1 | 0x80000000
		}
		return c
	}
	bits := func(n int) int {
		v := 0
		for ; n > 0; n-- {
			v = v<<1 | int(bit())
		}
		return v
	}
	q := len(out)
	copyBack := func(nbits, count int) {
		off := bits(nbits)
		for ; count > 0; count-- {
			q--
			out[q] = out[q+off]
		}
	}
	literal := func(count int) {
		for ; count > 0; count-- {
			q--
			out[q] = byte(bits(8))
		}
	}
	for q > 0 {
		if bit() == 0 {
			if bit() == 1 {
				copyBack(8, 2)
			} else {
				literal(bits(3) + 1)
			}
			continue
		}
		switch d := bits(2); d {
		case 0, 1:
			copyBack(9+d, d+3)
		case 2:
			copyBack(12, bits(8)+1)
		case 3:
			literal(bits(8) + 9)
		}
	}
	return out
}

// Amiga image header (BE): u16 height, u16 width cells, u16 x, u16 y,
// u16 frame count, u16 plane size; then per frame four bitplanes of
// height rows padded to whole 16-pixel words, plane 0 the low bit.

// chunky converts one frame of planes to DOS packed nibbles (w cells wide).
func chunky(src []byte, h, w, stride, plane int, lut *[16]byte) []byte {
	out := make([]byte, h*w*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w*8; x++ {
			var c byte
			for p := 0; p < 4; p++ {
				i := p*plane + y*stride + x/8
				if i < len(src) && src[i]&(0x80>>(x%8)) != 0 {
					c |= 1 << p
				}
			}
			c = lut[c]
			if x%2 == 0 {
				out[(y*w*8+x)/2] = c << 4
			} else {
				out[(y*w*8+x)/2] |= c
			}
		}
	}
	return out
}

func be16(b []byte, i int) int { return int(binary.BigEndian.Uint16(b[i:])) }

func le16(v int) []byte { return []byte{byte(v), byte(v >> 8)} }

// AmigaImage converts a plain picture block to the DOS layout ParseImage
// reads. split cuts each frame into its two side-by-side tiles (the Amiga symbol
// and combat tile sheets pack two per frame).
func AmigaImage(b []byte, split bool, lut *[16]byte) []byte {
	h, w, x, y, n, plane := be16(b, 0), be16(b, 2), be16(b, 4), be16(b, 6), be16(b, 8), be16(b, 10)
	var frames [][]byte
	for f := 0; f < n && 12+(f+1)*4*plane <= len(b); f++ {
		px := chunky(b[12+f*4*plane:], h, w, (w+1)/2*2, plane, lut)
		if !split {
			frames = append(frames, px)
			continue
		}
		hw := w / 2 * 4 // bytes per half row
		l, r := make([]byte, 0, h*hw), make([]byte, 0, h*hw)
		for row := 0; row < h; row++ {
			l = append(l, px[row*w*4:row*w*4+hw]...)
			r = append(r, px[row*w*4+hw:row*w*4+2*hw]...)
		}
		frames = append(frames, l, r)
	}
	if split {
		w /= 2
	}
	out := append(append(append(append(le16(h), le16(w)...), le16(x)...), le16(y)...), byte(len(frames)))
	out = append(out, make([]byte, 8)...)
	for _, f := range frames {
		out = append(out, f...)
	}
	return out
}

// AmigaSprite converts an animation block (u32 frame count, per frame u32
// delay in 50 Hz ticks and a one-frame picture) to the ParseSprite layout.
// xor stores frames after the first XORed against it, as PIC/FINAL expect.
func AmigaSprite(b []byte, xor bool, lut *[16]byte) []byte {
	n := int(binary.BigEndian.Uint32(b))
	out := []byte{byte(n)}
	var first []byte
	for f, p := 0, 4; f < n && p+16 <= len(b); f++ {
		delay := int(binary.BigEndian.Uint32(b[p:])) / 5
		h, w, x, y, plane := be16(b, p+4), be16(b, p+6), be16(b, p+8), be16(b, p+10), be16(b, p+14)
		p += 16
		if p+4*plane > len(b) {
			break
		}
		px := chunky(b[p:], h, w, (w+1)/2*2, plane, lut)
		p += 4 * plane
		if f == 0 {
			first = px
		} else if xor {
			for i := range px {
				if i < len(first) {
					px[i] ^= first[i]
				}
			}
		}
		out = append(out, byte(delay), byte(delay>>8), 0, 0)
		out = append(append(append(append(out, le16(h)...), le16(w)...), le16(x)...), le16(y)...)
		out = append(out, make([]byte, 9)...)
		out = append(out, px...)
	}
	return out
}

// WriteDOS packs blocks (id -> decoded bytes) into a DOS .DAX archive with
// every block stored uncompressed (raw size 0).
func WriteDOS(blocks map[int][]byte) []byte {
	var ids []int
	for id := range blocks {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	idx, data := []byte{}, []byte{}
	for _, id := range ids {
		b := blocks[id]
		idx = append(idx, byte(id))
		idx = binary.LittleEndian.AppendUint32(idx, uint32(len(data)))
		idx = append(idx, 0, 0)
		idx = binary.LittleEndian.AppendUint16(idx, uint16(len(b)))
		data = append(data, b...)
	}
	return append(append(le16(len(idx)), idx...), data...)
}

// Amiga palettes (12-bit RGB) from the program's data at 0x54020..0x54080.
var (
	PalUI       = [16]uint16{0x000, 0x06b, 0x096, 0x0aa, 0x000, 0x940, 0xb44, 0xfaa, 0xfe9, 0x59f, 0x0d8, 0x5ff, 0x555, 0x777, 0xaaa, 0xfff}
	PalPortrait = [16]uint16{0x000, 0xdb9, 0xb97, 0x964, 0x521, 0x742, 0xa84, 0xcb6, 0xff9, 0x007, 0x44b, 0x444, 0x555, 0x777, 0xaaa, 0xfff}
	PalCombat   = [16]uint16{0x000, 0x124, 0x953, 0xb97, 0xecb, 0x710, 0xa42, 0xdb0, 0xff5, 0x074, 0x053, 0x369, 0x555, 0x777, 0xaaa, 0xfff}
)

// EGALut maps an Amiga palette to the nearest EGA colours. Index 0 stays 0
// (the sprite mask colour); no other index lands on 0 or 13, the colours
// the engine treats as transparent.
// ponytail: nearest-colour remap loses the Amiga shades; a per-picture
// palette in the renderer would keep them.
func EGALut(pal [16]uint16, ega [16][3]uint8) *[16]byte {
	var lut [16]byte
	for i := 1; i < 16; i++ {
		r, g, b := int(pal[i]>>8&15)*17, int(pal[i]>>4&15)*17, int(pal[i]&15)*17
		best := -1
		for j := 1; j < 16; j++ {
			if j == 13 {
				continue
			}
			dr, dg, db := r-int(ega[j][0]), g-int(ega[j][1]), b-int(ega[j][2])
			d := dr*dr + dg*dg + db*db
			if best < 0 || d < best {
				best, lut[i] = d, byte(j)
			}
		}
	}
	return &lut
}
