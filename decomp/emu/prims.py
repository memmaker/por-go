import os
"""START primitives: intercepted (hardware-facing, skipped) or probed (traced, then run).

Every record starts with the original address so the Go side can emit the same
key from the function that ports it (see README "Trace records").

Arg spec letters, in Pascal order (left to right; the last arg sits next to the
return address):
  b  byte/char/boolean value (one stack word, low byte kept)
  w  word/integer value
  p  far pointer, shown as SSSS:OOOO (absolute segment: not comparable across runs)
  s  far pointer to a Pascal string, shown as its text
  v  far pointer to a picture record, shown as 'HxW:hash' of its decoded pixels
     (pic_id below; the Go side hashes dax.Image.Frame(frame) the same way)
kind: 'far' / 'near' for intercepts (the RET is emulated), 'probe' to trace and run.
"""
import hashlib
from unicorn import UC_HOOK_CODE
from unicorn.x86_const import *

PRIMS = {}


def prim(addr, name, spec, kind='far', quiet=False):
    """Register an intercept; the decorated handler may return a value for DX:AX."""
    def deco(fn=None):
        PRIMS[addr] = (name, spec, kind, fn, quiet)
        return fn
    return deco


def probe(addr, name, spec):
    PRIMS[addr] = (name, spec, 'probe', None, False)


def install(m):
    for addr, ent in PRIMS.items():
        a = m.lin(addr)
        m.mu.hook_add(UC_HOOK_CODE, _hit, (m, addr, ent), a, a)
    return PRIMS


def read_args(m, spec, kind):
    sp = (m.reg(UC_X86_REG_SS) << 4) + m.reg(UC_X86_REG_SP)
    o = sp + (2 if kind == 'near' else 4)
    vals, raw = [], []
    for c in reversed(spec):
        if c in 'psv':
            p = m.rd(o); o += 4
            raw.append(p)
            vals.append(m.pstr(m.far(p)) if c == 's' else pic_id(m, p) if c == 'v' else '%04X:%04X' % (p >> 16, p & 0xFFFF))
        else:
            w = m.rw(o); o += 2
            raw.append(w & 0xFF if c == 'b' else w)
            vals.append(raw[-1])
    vals.reverse(); raw.reverse()
    return vals, raw, o - sp - (2 if kind == 'near' else 4)


def _hit(mu, address, size, ud):
    m, addr, (name, spec, kind, fn, quiet) = ud
    vals, raw, nbytes = read_args(m, spec, kind)
    if kind == 'probe':
        m.emit(addr, name, *vals)
        return
    m.rec_extra = ()
    r = fn(m, *raw) if fn else None           # handler first, so --png-at N shows record N's effect
    if not quiet:
        m.emit(addr, name, *vals, *m.rec_extra)
    if m.stop_reason:
        return
    if r is not None:
        m.setreg(UC_X86_REG_AX, r & 0xFFFF)
        m.setreg(UC_X86_REG_DX, (r >> 16) & 0xFFFF)
    ss, sp = m.reg(UC_X86_REG_SS), m.reg(UC_X86_REG_SP)
    base = (ss << 4) + sp
    if kind == 'far':
        ip, cs = m.rw(base), m.rw(base + 2)
        m.setreg(UC_X86_REG_SP, (sp + 4 + nbytes) & 0xFFFF)
        m.setreg(UC_X86_REG_CS, cs)
    else:
        ip = m.rw(base)
        m.setreg(UC_X86_REG_SP, (sp + 2 + nbytes) & 0xFFFF)
    m.setreg(UC_X86_REG_IP, ip)


# ------------------------------------------------------------- pictures
def pic_pixels(m, p):
    """Decode an EGA picture record (0464:1951 layout): +0 height in pixel rows,
    +2 width in bytes (8-pixel cells), +0x17 data: per row, per cell, one byte per
    plane in DS:25F6 map-mask order. Returns (h, w_pixels, bytes of colour indices)."""
    a = m.far(p)
    h, wb = m.rw(a), m.rw(a + 2)
    if h > 200 or wb > 40:
        return h, wb * 8, None
    masks = bytes(m.mu.mem_read(m.dsa(0x25F6), 4))
    data = bytes(m.mu.mem_read(a + 0x17, h * wb * 4))
    out = bytearray(h * wb * 8)
    i = 0
    for r in range(h):
        row = r * wb * 8
        for c in range(wb):
            planes = data[i:i + 4]; i += 4
            for b in range(8):
                bit = 0x80 >> b
                v = 0
                for k in range(4):
                    if planes[k] & bit:
                        v |= masks[k]
                out[row + c * 8 + b] = v & 0xF
    return h, wb * 8, bytes(out)


def pic_id(m, p):
    if p == 0:
        return 'nil'
    h, w, px = pic_pixels(m, p)
    if px is None:
        return '%dx%d:?' % (h, w)
    return '%dx%d:%s' % (h, w, hashlib.md5(px).hexdigest()[:10])


# ------------------------------------------------------------------ RTL
@prim('0AF8:1247', 'Randomize', '')
def _randomize(m):
    m.wd(m.dsa(0x4428), m.seed)       # RandSeed := the run's seed (Go: Engine.FixedSeed)


@prim('0A40:029C', 'Delay', 'w')      # Crt.Delay(ms)
def _delay(m, ms):
    m.ticks += ms


@prim('0A40:02C9', 'Sound', 'w')      # Crt.Sound(hz)
def _sound(m, hz):
    pass


@prim('0A40:02F6', 'NoSound', '')
def _nosound(m):
    pass


# ------------------------------------------------------------- timer
@prim('0709:0C3F', 'Ticks', '')   # hundredths of a second since midnight
def _ticks(m):
    # time model (Go trace mode does the same): virtual ms = sum of Delay(ms)
    # + 10 per Ticks() call, so busy-wait loops on the clock terminate.
    t = m.ticks // 10
    m.ticks += 10
    return t


# ------------------------------------------------------------- text / cells
@prim('0709:01DF', 'DisplayChar', 'bbbbbb')   # (x, y, colour, count, ch, upcase)
def _dchar(m, x, y, colour, count, ch, upcase):
    m.fb.char(m, x, y, colour, count, ch, upcase)


@prim('0709:00B7', 'FillCells', 'bbbbbb')     # (x1, y1, x2, y2, page, value)
def _fillcells(m, x1, y1, x2, y2, page, value):
    m.fb.fill(x1 * 8, y1 * 8, (x2 + 1) * 8, (y2 + 1) * 8, value & 0xF)


# ------------------------------------------------------------- pictures / pages
@prim('0464:1951', 'DrawPicture', 'wwv')      # (x, y, pic)
def _drawpic(m, x, y, pic):
    m.fb.pic(m, x * 8, y * 8, pic)


@prim('0464:1B52', 'Flip', '')                # copy the dirty spans of the drawing page DS:4A20 to the screen
def _flip(m):
    p = m.rd(m.dsa(0x4A20))
    m.rec_extra = (pic_id(m, p),)             # the whole page, as a picture hash
    m.flips = getattr(m, 'flips', 0) + 1
    if p and os.environ.get('PAGEDUMP') == str(m.flips):   # debug: raw page pixels of the Nth Flip
        open('out/page.emu.bin', 'wb').write(bytes(pic_pixels(m, p)[2]))
    # rows 0..0xA7: DS:4A28[r] dirty flag; page byte offsets DS:4C20[r]..DS:4D70[r]
    # (4 plane bytes per cell); screen cell column DS:4AD0[r]+1, pixel row r+8.
    if p:
        h, w, px = pic_pixels(m, p)
        flags = bytes(m.mu.mem_read(m.dsa(0x4A28), 0xA8))
        for r in range(min(0xA8, h)):
            if not flags[r]:
                continue
            lo, hi, dc = m.rw(m.dsa(0x4C20 + 2 * r)), m.rw(m.dsa(0x4D70 + 2 * r)), m.rw(m.dsa(0x4AD0 + 2 * r))
            c0 = lo // 4 - r * (w // 8)
            n = (hi - lo) // 4 + 1
            if px is None or c0 < 0 or n <= 0:
                continue
            row = px[r * w + c0 * 8: r * w + (c0 + n) * 8]
            m.fb.blit((dc + 1) * 8, r + 8, len(row), 1, row)
    # the original clears its dirty tables on the way out
    m.mu.mem_write(m.dsa(0x4A28), bytes(0xA8))
    m.mu.mem_write(m.dsa(0x4C20), b'\xFF' * 0x150)
    m.mu.mem_write(m.dsa(0x4D70), bytes(0x150))


@prim('0464:2A0D', 'SetPaletteColor', 'bb')   # (colour, index)
def _setpal(m, colour, index):
    m.fb.palette[index & 0xF] = colour
    # Intr(0x10) register block the original fills: AX=1000h, BL=index(+8 if >=8), BH=colour
    m.wb(m.dsa(0x4A03), 0x10); m.wb(m.dsa(0x4A02), 0)
    m.wb(m.dsa(0x4A05), index if index < 8 else index + 8); m.wb(m.dsa(0x4A04), colour)


# ------------------------------------------------------------- sound driver (08AE)
# Only the DS side effects are kept: DS:5372 sound type (2 = off), 5373 saved type,
# 5374 wait mode. The driver's own state lives in its code segment.
@prim('08AE:00DC', 'PlaySound', 'b')
def _play(m, n):
    pass


@prim('08AE:01A5', 'SoundWait', 'w')
def _soundwait(m, n):
    if m.rb(m.dsa(0x5372)) != 2:
        m.ww(m.dsa(0x5374), n)


@prim('08AE:017B', 'SoundReset', '')
def _soundreset(m):
    m.wb(m.dsa(0x5373), m.rb(m.dsa(0x5372)))


prim('08AE:01D2', 'SoundInit', '')(None)     # empty unit init in the original too

# ------------------------------------------------------------- probes (traced, then run)
for _a, _n, _s in [
    ('0709:0626', 'DisplayString', 'bbbs'),          # (x, y, colour, s)
    ('0709:0768', 'DisplayTextBlock', 'bbbbbbs'),    # (x1, y1, x2, y2, colour, clear, s)
    ('03F1:02E8', 'DrawBoxTitled', 'bbbbbbs'),       # (x1, y1, x2, y2, style, colour, title)
    ('03F1:0137', 'ClearCells', 'bbbb'),             # (x1, y1, x2, y2)
    ('0709:0DD3', 'ClearBox', 'bbbb'),
    ('0709:0CAB', 'ClearScreen', ''),
    ('0709:05EB', 'DisplaySpaces', 'bbbb'),          # (x, y, colour, count)
    ('0709:0D68', 'DisplayMessageWait', 'sb'),       # (s, colour)
    ('0709:0BF5', 'MessageRow24b', 'bs'),
    ('0464:0111', 'LoadPic', 'sbbbp'),               # (name, block, ?, mask, var pic)
    ('0464:0D71', 'DrawPage', 'wwbbvp'),             # (x, y, ?, mode, src, dest page)
    ('0464:08D4', 'DrawPictureMasked', 'wwbbvp'),
    ('0464:2632', 'FillRect', 'wwwwbpp'),            # (x, y, w, h, colour, ?, page)
]:
    probe(_a, _n, _s)

# ------------------------------------------------------------- deliberate skips (as the Go port)
@prim('ovr03:00FD', 'CopyProtection', '')     # code wheel: the Go skips it, so must we (it calls Random)
def _protect(m):
    pass
