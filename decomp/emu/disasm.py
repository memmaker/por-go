"""Disassemble START.bin / an overlay from an exact address (no linear-sweep desync).

  .venv/bin/python disasm.py 0709:01DF [count]      .venv/bin/python disasm.py ovr13:0233 [count]
Stops after `count` instructions (default: at the first ret/retf after 0x400 bytes, or 200).
"""
import sys, os, json
from capstone import Cs, CS_ARCH_X86, CS_MODE_16

OUT = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), 'out')


def code(addr):
    s, o = addr.split(':')
    o = int(o, 16)
    if s.lower().startswith('ovr'):
        return open(os.path.join(OUT, 'ovr', 'ovr%02d.bin' % int(s[3:])), 'rb').read(), o, 0
    b = open(os.path.join(OUT, 'START.bin'), 'rb').read()
    base = int(s, 16) * 16
    return b[base:base + 0x10000], o, base


def main():
    b, o, _ = code(sys.argv[1])
    n = int(sys.argv[2]) if len(sys.argv) > 2 else 200
    md = Cs(CS_ARCH_X86, CS_MODE_16)
    for i, ins in enumerate(md.disasm(b[o:], o)):
        print('%04X  %-6s %s' % (ins.address, ins.mnemonic, ins.op_str))
        if i + 1 >= n or (len(sys.argv) <= 2 and ins.mnemonic in ('ret', 'retf', 'iret')):
            break


if __name__ == '__main__':
    main()
