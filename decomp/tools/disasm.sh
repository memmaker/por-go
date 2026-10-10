#!/bin/sh
# Disassemble each overlay unit and the resident START.EXE code with radare2 (16-bit x86).
cd "$(dirname "$0")/../out"
for f in ovr/ovr*.bin; do
  n=$(basename $f .bin)
  r2 -q -a x86 -b 16 -e scr.color=0 -e asm.bytes=true -e asm.lines=false -c 'pD $s' $f > asm/$n.asm 2>/dev/null
done
r2 -q -a x86 -b 16 -e scr.color=0 -e asm.bytes=true -e asm.lines=false -c 'pD $s' START.bin > asm/start.asm 2>/dev/null
wc -l asm/*.asm | tail -1
