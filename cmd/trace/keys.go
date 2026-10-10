package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseKeys = decomp/emu/keys.py parse_keys: BIOS words (scan<<8 | ascii).
func ParseKeys(toks []string) ([]uint16, error) {
	named := map[string]uint16{
		"enter": 0x1C0D, "esc": 0x011B, "space": 0x3920, "bs": 0x0E08, "tab": 0x0F09,
		"up": 0x4800, "down": 0x5000, "left": 0x4B00, "right": 0x4D00,
		"home": 0x4700, "end": 0x4F00, "pgup": 0x4900, "pgdn": 0x5100, "ins": 0x5200, "del": 0x5300,
	}
	for i := 0; i < 10; i++ {
		named[fmt.Sprintf("f%d", i+1)] = uint16(0x3B+i) << 8
	}
	scan := map[byte]uint16{' ': 0x39}
	shift := map[byte]byte{}
	plain, shifted := "1234567890-=[];'`,./", "!@#$%^&*()_+{}:\"~<>?"
	for i := range plain {
		shift[plain[i]] = shifted[i]
	}
	for _, row := range []struct {
		base  uint16
		chars string
	}{{0x02, "1234567890-="}, {0x10, "qwertyuiop[]"}, {0x1E, "asdfghjkl;'`"}, {0x2C, "zxcvbnm,./"}} {
		for i := 0; i < len(row.chars); i++ {
			c := row.chars[i]
			scan[c] = row.base + uint16(i)
			if c >= 'a' && c <= 'z' {
				scan[c-32] = row.base + uint16(i)
			} else if s, ok := shift[c]; ok {
				scan[s] = row.base + uint16(i)
			}
		}
	}
	var out []uint16
	for _, t := range toks {
		lt := strings.ToLower(t)
		switch {
		case named[lt] != 0:
			out = append(out, named[lt])
		case strings.HasPrefix(lt, "0x"):
			v, err := strconv.ParseUint(lt[2:], 16, 16)
			if err != nil {
				return nil, err
			}
			out = append(out, uint16(v))
		case len(t) == 1:
			out = append(out, scan[t[0]]<<8|uint16(t[0]))
		default:
			return nil, fmt.Errorf("bad key token %q", t)
		}
	}
	return out, nil
}
