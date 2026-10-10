"""Framebuffer reconstruction from the intercepted primitives (one 320x200 page).

Not an EGA model: page flipping and plane tricks are ignored. It paints what the
traced primitives would have painted, so a PNG of a run shows where it got to.
"""
import prims

# EGA default palette registers -> 6-bit rgbRGB colour
DEFAULT = [0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x14, 0x07, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F]


def rgb(c6):
    def ch(hi, lo):
        return 0x55 * ((2 if c6 & hi else 0) + (1 if c6 & lo else 0))
    return (ch(0x04, 0x20), ch(0x02, 0x10), ch(0x01, 0x08))


class FB:
    def __init__(self):
        self.px = bytearray(320 * 200)
        self.palette = list(DEFAULT)

    def fill(self, x1, y1, x2, y2, c):
        for y in range(max(0, y1), min(200, y2)):
            self.px[y * 320 + max(0, x1):y * 320 + min(320, x2)] = bytes([c]) * (min(320, x2) - max(0, x1))

    def blit(self, x, y, w, h, data, mask=None):
        for r in range(h):
            yy = y + r
            if not 0 <= yy < 200:
                continue
            for c in range(w):
                xx = x + c
                v = data[r * w + c]
                if 0 <= xx < 320 and v != mask:
                    self.px[yy * 320 + xx] = v

    def char(self, m, x, y, colour, count, ch, upcase):
        if x > 0x27 or y > 0x18:
            return
        font = m.rd(m.dsa(0x5E20))
        if not font:
            return
        g = ch
        if upcase:
            g = (ord(chr(ch).upper()) if 0x61 <= ch <= 0x7A else ch) % 0x40
        bits = bytes(m.mu.mem_read(m.far(font) + g * 8, 8))
        for i in range(count):
            for r in range(8):
                for b in range(8):
                    xx, yy = (x + i) * 8 + b, y * 8 + r
                    if xx < 320:
                        self.px[yy * 320 + xx] = colour & 0xF if bits[r] & (0x80 >> b) else 0

    def pic(self, m, x, y, p):
        if not p:
            return
        h, w, data = prims.pic_pixels(m, p)
        if data:
            self.blit(x, y, w, h, data)

    def png(self, path):
        from PIL import Image
        im = Image.new('RGB', (320, 200))
        pal = [rgb(self.palette[i]) for i in range(16)]
        im.putdata([pal[v & 0xF] for v in self.px])
        im.resize((640, 400), Image.NEAREST).save(path)
