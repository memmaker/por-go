"""Find the START routines that touch hardware (EGA window writes, port I/O).

Runs the game un-intercepted (only the RTL Randomize hook) with a key script and,
on every A000 write / OUT, walks the BP frame chain to name the routine chain
that led there. Prints, per outermost START routine called from non-hardware
code, how many hardware accesses happened under it. Used to choose prims.py.

  .venv/bin/python hwprofile.py [insns] [keys...]
"""
import sys, collections
from unicorn import *
from unicorn.x86_const import *
import emu

HWSEGS = {0x0464, 0x0709, 0x0802, 0x0899, 0x08AE, 0x0A40, 0x03F1, 0x00BA, 0x0AF8, 0x0A31}


def main():
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 300_000_000
    from keys import parse_keys
    keys = parse_keys(sys.argv[2:] or ['enter'] * 8)
    m = emu.Machine(keys=keys)
    m.prims.clear()
    hits = collections.Counter()
    chains = {}

    def callee(cs, ip):
        """Target of the call instruction that returns to cs:ip."""
        a = (cs << 4) + ip
        b = m.mu.mem_read(a - 5, 5)
        if b[0] == 0x9A:
            o, s = int.from_bytes(b[1:3], 'little'), int.from_bytes(b[3:5], 'little')
            return m.where(s, o)
        if b[2] == 0xE8:
            return m.where(cs, (ip + int.from_bytes(b[3:5], 'little', signed=True)) & 0xFFFF)
        return '?'

    def frames():
        bp = m.reg(UC_X86_REG_BP); ss = m.reg(UC_X86_REG_SS)
        cs = m.reg(UC_X86_REG_CS)
        out = [m.where()]
        for _ in range(12):
            a = (ss << 4) + bp
            nbp, rip, rcs = m.rw(a), m.rw(a + 2), m.rw(a + 4)
            # near or far frame: a far one has a plausible code segment
            far = emu.LOAD <= rcs < 0xF000 and (rcs - emu.LOAD < 0x1000 or rcs >= emu.OVSEG)
            if not far:
                rcs = cs
            out.append((callee(rcs, rip), m.where(rcs, rip)))
            cs = rcs
            if nbp <= bp:
                break
            bp = nbp
        return out

    def outer(fr):
        # first frame whose call site is outside the hardware segments
        prev = fr[0]
        for tgt, site in fr[1:]:
            seg = site.split(':')[0]
            if seg.startswith('ovr') or int(seg, 16) not in HWSEGS:
                return tgt, site
        return fr[-1]

    def note(kind):
        fr = frames()
        k = (kind, outer(fr)[0], fr[0].split(':')[0])
        hits[k] += 1
        chains.setdefault(k, fr)

    def memw(mu, acc, addr, size, val, ud):
        note('A000')

    def out(mu, port, size, val, ud):
        note('out%X' % port)

    m.mu.hook_add(UC_HOOK_MEM_WRITE, memw, None, 0xA0000, 0xAFFFF)
    m.mu.hook_add(UC_HOOK_INSN, out, None, 1, 0, UC_X86_INS_OUT)
    print(m.run(n))
    for k, c in sorted(hits.items(), key=lambda kv: (kv[0][1], kv[0][0])):
        print('%-8s %-14s %-6s %8d   %s' % (k[0], k[1], k[2], c, chains[k][1:4]))


if __name__ == '__main__':
    main()
