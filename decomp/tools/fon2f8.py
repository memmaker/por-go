#!/usr/bin/env python3
"""Convert 8x8 Windows bitmap fonts (.FON, code page 437) to raw .f8 files:
256 glyphs x 8 rows, one byte per row, bit 7 = left pixel.

    fon2f8.py OUTDIR FILE.FON...
"""
import os, struct, sys

def fnt(data):
    mz = data
    ne = struct.unpack_from('<H', mz, 0x3C)[0]
    rt = ne + struct.unpack_from('<H', mz, ne + 0x24)[0]
    shift = struct.unpack_from('<H', mz, rt)[0]
    p = rt + 2
    while True:
        typ, count = struct.unpack_from('<HH', mz, p)
        if typ == 0:
            return None
        p += 8
        for _ in range(count):
            off, ln = struct.unpack_from('<HH', mz, p)
            if typ == 0x8008:
                return mz[off << shift:(off << shift) + (ln << shift)]
            p += 12

def glyphs(f):
    ver = struct.unpack_from('<H', f, 0)[0]
    w, h = struct.unpack_from('<HH', f, 0x56)
    first, last = f[0x5F], f[0x60]
    if h != 8 or w not in (0, 8):
        return None
    out = bytearray(256 * 8)
    tab, size = (0x76, 4) if ver == 0x200 else (0x94, 6)
    for i in range(last - first + 1):
        if ver == 0x200:
            cw, off = struct.unpack_from('<HH', f, tab + i * size)
        else:
            cw, off = struct.unpack_from('<HI', f, tab + i * size)
        if cw != 8:
            return None
        out[(first + i) * 8:(first + i) * 8 + 8] = f[off:off + 8]
    return bytes(out)

out = sys.argv[1]
for path in sys.argv[2:]:
    f = fnt(open(path, 'rb').read())
    g = f and glyphs(f)
    name = os.path.basename(path)[6:-4]
    if not g:
        print('skip', name)
        continue
    open(os.path.join(out, name + '.f8'), 'wb').write(g)
    print('ok', name)
