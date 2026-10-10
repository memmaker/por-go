"""List START far-call targets in the given segments with their RET size (arg bytes)."""
import sys, os, re
from capstone import Cs, CS_ARCH_X86, CS_MODE_16
OUT = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), 'out')
b = open(OUT + '/START.bin', 'rb').read()
md = Cs(CS_ARCH_X86, CS_MODE_16)
segs = sys.argv[1:]
for ln in open(OUT + '/xref.tsv'):
    t, n, who = ln.rstrip('\n').split('\t')
    s = t.split(':')[0]
    if s not in segs: continue
    base = int(s, 16) * 16; o = int(t.split(':')[1], 16)
    ret = '?'
    for ins in md.disasm(b[base + o: base + o + 0x1000], o):
        if ins.mnemonic in ('ret', 'retf'):
            ret = ins.mnemonic + ' ' + ins.op_str; break
    print('%s %4s  %-10s %s' % (t, n, ret, who[:90]))
