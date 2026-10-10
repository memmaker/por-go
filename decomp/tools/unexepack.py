"""Unpack an EXEPACK-compressed MZ executable into a flat load image (relocations applied with base 0)."""
import struct, sys

def unpack(path):
    d = open(path, 'rb').read()
    h = struct.unpack('<14H', d[:28])
    hdr = h[4] * 16
    cs, ip = h[11], h[10]
    eh = hdr + cs * 16
    real_ip, real_cs, mem_start, exepack_size, real_sp, real_ss, dest_len, skip_len, sig = struct.unpack('<8H2s', d[eh:eh + 18])
    assert sig == b'RB', sig
    packed = d[hdr:eh]
    # strip trailing 0xFF padding
    src = len(packed)
    while src > 0 and packed[src - 1] == 0xFF:
        src -= 1
    out = bytearray(dest_len * 16)
    dst = len(out)
    while True:
        src -= 1; cmd = packed[src]
        src -= 2; ln = struct.unpack('<H', packed[src:src + 2])[0]
        op = cmd & 0xFE
        if op == 0xB0:
            src -= 1; v = packed[src]
            out[dst - ln:dst] = bytes([v]) * ln; dst -= ln
        elif op == 0xB2:
            out[dst - ln:dst] = packed[src - ln:src]; src -= ln; dst -= ln
        else:
            raise SystemExit('bad cmd %02x at %d' % (cmd, src))
        if cmd & 1:
            break
    # any bytes not covered at the front are copied verbatim
    if src > 0:
        out[dst - src:dst] = packed[:src]
    # relocation table follows the unpacker code
    rp = eh + exepack_size
    rt = eh + 18 + 0  # not needed: find via exepack_size
    relocs = []
    p = eh + exepack_size - 0  # placeholder
    # The reloc table starts after the unpacker code; its size = exepack_size - 18 - codelen. Locate by scanning for the known layout:
    # 16 sections, each: count u16 then count u16 offsets. We find it as the tail of the exepack region.
    # Search backwards for a consistent table.
    end = eh + exepack_size
    for start in range(eh + 18, end):
        q = start; ok = True; r = []
        for seg in range(16):
            if q + 2 > end: ok = False; break
            n = struct.unpack('<H', d[q:q + 2])[0]; q += 2
            if q + 2 * n > end: ok = False; break
            r += [(seg * 0x1000 + struct.unpack('<H', d[q + 2 * i:q + 2 * i + 2])[0]) for i in range(n)]; q += 2 * n
        if ok and q == end:
            relocs = r; break
    return bytes(out), dict(real_ip=real_ip, real_cs=real_cs, real_ss=real_ss, real_sp=real_sp, dest_len=dest_len, relocs=relocs)

if __name__ == '__main__':
    img, info = unpack(sys.argv[1])
    open(sys.argv[2], 'wb').write(img)
    print('image', len(img), 'entry %04X:%04X ss %04X sp %04X relocs %d' % (info['real_cs'], info['real_ip'], info['real_ss'], info['real_sp'], len(info['relocs'])))
    open(sys.argv[2] + '.relocs', 'w').write('\n'.join('%05X' % r for r in info['relocs']))
