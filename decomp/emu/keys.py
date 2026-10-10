"""Key script parsing, shared with the Go side (por-go/cmd/trace reads the same names).

A key is a BIOS INT 16h word (scan<<8 | ascii). Script tokens:
  enter esc space bs tab up down left right home end pgup pgdn ins del f1..f10
  a single character (letters, digits, punctuation)  e.g. "c" or "C"
  0xHHHH  raw word
A script file holds tokens separated by whitespace; '#' starts a comment.
"""
NAMED = {
    'enter': 0x1C0D, 'esc': 0x011B, 'space': 0x3920, 'bs': 0x0E08, 'tab': 0x0F09,
    'up': 0x4800, 'down': 0x5000, 'left': 0x4B00, 'right': 0x4D00,
    'home': 0x4700, 'end': 0x4F00, 'pgup': 0x4900, 'pgdn': 0x5100, 'ins': 0x5200, 'del': 0x5300,
}
for i in range(10):
    NAMED['f%d' % (i + 1)] = (0x3B + i) << 8
# scan codes of the main block, for single characters
_ROWS = [(0x02, '1234567890-='), (0x10, 'qwertyuiop[]'), (0x1E, "asdfghjkl;'`"), (0x2C, 'zxcvbnm,./')]
_SHIFT = dict(zip('1234567890-=[];\'`,./', '!@#$%^&*()_+{}:"~<>?'))
SCAN = {}
for base, chars in _ROWS:
    for i, c in enumerate(chars):
        SCAN[c] = base + i
        if c.isalpha():
            SCAN[c.upper()] = base + i
        elif c in _SHIFT:
            SCAN[_SHIFT[c]] = base + i
SCAN[' '] = 0x39


def parse_keys(tokens):
    out = []
    for t in tokens:
        if t.lower() in NAMED:
            out.append(NAMED[t.lower()])
        elif t.lower().startswith('0x'):
            out.append(int(t, 16))
        elif len(t) == 1:
            out.append(SCAN.get(t, 0) << 8 | ord(t))
        else:
            raise ValueError('bad key token %r' % t)
    return out


def load_script(path):
    toks = []
    for ln in open(path):
        toks += ln.split('#')[0].split()
    return parse_keys(toks)


if __name__ == '__main__':
    assert parse_keys(['enter', 'a', 'A', '0x4800', 'f1']) == [0x1C0D, 0x1E61, 0x1E41, 0x4800, 0x3B00]
    print('ok')
