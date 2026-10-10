"""Command line for the emulator harness. See README.md.

  run.py game [--keys "enter esc c"] [--script FILE] [--seed N] [--insns N]
              [--out trace.json] [--png fb.png] [--save-snap FILE]
  run.py call ROUTINE [ARG...] --snap FILE [--near] [--poke DS:5376=03 ...]
              [--seed N] [--keys ...] [--insns N] [--out result.json]

ROUTINE: 'ovr13:0233' or a START address '0464:0D71' (image-relative segment).
ARG (Pascal order, left to right):  123 / 0x7B  word value
  ptr:SSSS:OOOO   far pointer (absolute segment; DS:OOOO means the data segment)
  str:TEXT        Pascal string copied to scratch memory, passed as a far pointer
  buf:SIZE        zeroed scratch buffer of SIZE bytes (var param), dumped afterwards
"""
import argparse, json, os, struct, sys, zlib
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import emu
from emu import Machine, json_safe
from keys import parse_keys, load_script
from unicorn import UC_HOOK_CODE, UC_HOOK_MEM_WRITE
from unicorn.x86_const import *

SNAP_MAGIC = b'PORSNAP1'
SCRATCH_SEG = 0x9F00            # 4K below TOPMEM for call-mode args: the TP heap rarely reaches it
RET_FAR = (emu.BIOSSEG, 0xFF00)


# ------------------------------------------------------------- snapshots
REGS = ['AX', 'BX', 'CX', 'DX', 'SI', 'DI', 'BP', 'SP', 'CS', 'DS', 'ES', 'SS', 'IP', 'EFLAGS']


def save_snap(m, path):
    regs = {r: m.reg(getattr(sys.modules['unicorn.x86_const'], 'UC_X86_REG_' + r)) for r in REGS}
    meta = json.dumps({'regs': regs, 'ticks': m.ticks, 'seed': m.seed, 'bios_tick': getattr(m, 'bios_tick', 0),
                       'stop': m.stop_reason}).encode()
    mem = zlib.compress(bytes(m.mu.mem_read(0, 0x110000)), 6)
    with open(path, 'wb') as f:
        f.write(SNAP_MAGIC + struct.pack('<I', len(meta)) + meta + mem)


def load_snap(m, path):
    d = open(path, 'rb').read()
    assert d[:8] == SNAP_MAGIC, 'not a snapshot'
    n = struct.unpack('<I', d[8:12])[0]
    meta = json.loads(d[12:12 + n])
    m.mu.mem_write(0, zlib.decompress(d[12 + n:]))
    for r, v in meta['regs'].items():
        m.setreg(getattr(sys.modules['unicorn.x86_const'], 'UC_X86_REG_' + r), v)
    m.ticks = meta['ticks']; m.bios_tick = meta['bios_tick']
    return meta


# ------------------------------------------------------------- dumps
def dump(m, extra):
    out = {'seed': m.seed, 'stop': m.stop_reason, 'virtual_ms': m.ticks, 'keys_left': len(m.keys)}
    out.update(extra)
    out['trace'] = [json_safe(r) for r in m.trace]
    out['regs'] = m.regs()
    out['memory'] = m.regions()
    return out


def diff_bytes(a, b, base=0):
    """Changed byte runs: [[offset_hex, old_hex, new_hex], ...]."""
    runs, i, n = [], 0, min(len(a), len(b))
    while i < n:
        if a[i] != b[i]:
            j = i
            while j < n and (a[j] != b[j] or (j + 1 < n and a[j + 1] != b[j + 1])):
                j += 1
            runs.append(['%04X' % (base + i), a[i:j].hex(), b[i:j].hex()])
            i = j
        else:
            i += 1
    return runs


def region_bytes(m):
    out = {}
    for name, ptr, size in emu.REGIONS:
        p = m.rd(m.dsa(ptr))
        if p:
            out[name] = bytes(m.mu.mem_read(m.far(p), size))
    for i, p in enumerate(m.party()):
        out['party%d' % i] = bytes(m.mu.mem_read(m.far(p), 0x11D))
    return out


# ------------------------------------------------------------- modes
def cmd_game(a):
    keys = (load_script(a.script) if a.script else []) + parse_keys(a.keys.split())
    m = Machine(seed=a.seed, keys=keys, verbose=a.verbose)
    for spec in a.png_at:                     # N:path, PNG after the Nth trace record
        n, path = spec.split(':', 1)
        m.png_at[int(n)] = path
    if os.environ.get('WATCH'):               # debug: WATCH=DS:5F1A reports writes there
        sg, ofs = os.environ['WATCH'].split(':')
        ofs = int(ofs, 16)
        lin = m.dsa(ofs) if sg == 'DS' else ofs  # or LIN:linear
        def wr(mu, acc, addr, size, val, ud):
            bp, ss, fr = m.reg(UC_X86_REG_BP), m.reg(UC_X86_REG_SS), []
            for _ in range(4):
                fa = (ss << 4) + bp
                fr.append(m.where(m.rw(fa + 4), m.rw(fa + 2)))
                bp = m.rw(fa)
            print('WRITE %05X' % addr, hex(val), m.where(m.reg(UC_X86_REG_CS)), len(m.trace), fr, file=sys.stderr)
        m.mu.hook_add(UC_HOOK_MEM_WRITE, wr, None, lin, lin + 3)
    if os.environ.get('BPAT'):                # debug: BPAT=ovr20:2468 prints BP each time there
        def at(mu, addr, size, ud):
            if m.where() in os.environ['BPAT'].split(','):
                bp, ss, fr = m.reg(UC_X86_REG_BP), m.reg(UC_X86_REG_SS), [m.caller()]
                for _ in range(4):
                    fa = (ss << 4) + bp
                    fr.append(m.where(m.rw(fa + 4), m.rw(fa + 2)))
                    bp = m.rw(fa)
                if os.environ.get('BPMEM'):
                    fr.append('mem=%02X' % m.mu.mem_read((m.reg(UC_X86_REG_SS) << 4) + m.reg(UC_X86_REG_BP) + int(os.environ['BPMEM']), 1)[0])
                print('BPAT', m.where(), fr, 'SS:BP=%04X:%04X' % (m.reg(UC_X86_REG_SS), m.reg(UC_X86_REG_BP)), len(m.trace), 'ms', m.ticks, 'AX=%04X' % m.reg(UC_X86_REG_AX), 'ARGS=%04X:%04X' % (m.rw((m.reg(UC_X86_REG_SS)<<4)+m.reg(UC_X86_REG_SP)+6), m.rw((m.reg(UC_X86_REG_SS)<<4)+m.reg(UC_X86_REG_SP)+4)), file=sys.stderr)
        m.mu.hook_add(UC_HOOK_CODE, at)
    stop = m.run(a.insns)
    print(stop, file=sys.stderr)
    out = dump(m, {'mode': 'game', 'keys': keys})
    out['party'] = ['%04X:%04X' % (p >> 16, p & 0xFFFF) for p in m.party()]
    if a.out:
        json.dump(out, open(a.out, 'w'), indent=None)
    if a.png:
        m.fb.png(a.png)
    if a.save_snap:
        save_snap(m, a.save_snap)
    print('%d trace records, %d keys left, virtual %d ms' % (len(m.trace), len(m.keys), m.ticks), file=sys.stderr)


def parse_arg(m, s, alloc):
    if s.startswith('ptr:'):
        sg, of = s[4:].split(':')
        sg = m.ds if sg.upper() == 'DS' else int(sg, 16)
        return 'p', (sg << 16) | int(of, 16)
    if s.startswith('str:'):
        t = s[4:].encode('latin1')
        a = alloc(256); m.mu.mem_write(a, bytes([len(t)]) + t)
        return 'p', ((a >> 4) << 16) | (a & 0xF)
    if s.startswith('buf:'):
        a = alloc(int(s[4:], 0))
        return 'p', ((a >> 4) << 16) | (a & 0xF)
    return 'w', int(s, 0) & 0xFFFF


def cmd_call(a):
    m = Machine(seed=a.seed, keys=parse_keys(a.keys.split()), verbose=a.verbose)
    load_snap(m, a.snap)
    for p in a.poke:
        where, val = p.split('=')
        sg, of = where.split(':')
        base = ((m.ds if sg.upper() == 'DS' else int(sg, 16)) << 4) + int(of, 16)
        m.mu.mem_write(base, bytes.fromhex(val))
    # scratch memory for str:/buf: args
    nxt = [SCRATCH_SEG << 4]
    bufs = []
    def alloc(n):
        r = nxt[0]; nxt[0] += (n + 15) & ~15
        m.mu.mem_write(r, bytes(n)); bufs.append((r, n))
        return r
    args = [parse_arg(m, s, alloc) for s in a.args]
    before_ds, before_rg = m.ds_bytes(), region_bytes(m)
    ss, sp = m.reg(UC_X86_REG_SS), m.reg(UC_X86_REG_SP) - 0x40   # below the snapshot's live frame
    def push(v):
        nonlocal sp
        sp = (sp - 2) & 0xFFFF
        m.ww((ss << 4) + sp, v)
    for kind, v in args:                      # Pascal: left to right
        if kind == 'p':
            push(v >> 16); push(v & 0xFFFF)
        else:
            push(v)
    tgt = m.lin(a.routine)
    sg = a.routine.split(':')[0]
    tcs = m.ovseg[int(sg[3:])] if sg.lower().startswith('ovr') else emu.seg(int(sg, 16))
    if a.near:
        ret = (tcs, 0xFFFE); push(0xFFFE)
    else:
        ret = RET_FAR; push(RET_FAR[0]); push(RET_FAR[1])
    want_sp = (sp + (2 if a.near else 4)) & 0xFFFF      # SP after the callee's RET (args not counted)
    def done(mu, addr, size, ud):
        if m.reg(UC_X86_REG_SP) >= want_sp:
            m.stop('returned')
    m.mu.hook_add(UC_HOOK_CODE, done, None, (ret[0] << 4) + ret[1], (ret[0] << 4) + ret[1])
    m.setreg(UC_X86_REG_SS, ss); m.setreg(UC_X86_REG_SP, sp)
    m.setreg(UC_X86_REG_CS, tcs); m.setreg(UC_X86_REG_IP, tgt - (tcs << 4))
    m.setreg(UC_X86_REG_DS, m.ds)            # TP code assumes DS = data segment on entry
    m.trace = []
    m.stop_reason = None
    if os.environ.get('BPAT'):                # debug: BPAT=ovr20:2468 prints BP each time there
        def at(mu, addr, size, ud):
            if m.where() in os.environ['BPAT'].split(','):
                bp, ss, fr = m.reg(UC_X86_REG_BP), m.reg(UC_X86_REG_SS), [m.caller()]
                for _ in range(4):
                    fa = (ss << 4) + bp
                    fr.append(m.where(m.rw(fa + 4), m.rw(fa + 2)))
                    bp = m.rw(fa)
                if os.environ.get('BPMEM'):
                    fr.append('mem=%02X' % m.mu.mem_read((m.reg(UC_X86_REG_SS) << 4) + m.reg(UC_X86_REG_BP) + int(os.environ['BPMEM']), 1)[0])
                print('BPAT', m.where(), fr, 'SS:BP=%04X:%04X' % (m.reg(UC_X86_REG_SS), m.reg(UC_X86_REG_BP)), len(m.trace), 'ms', m.ticks, 'AX=%04X' % m.reg(UC_X86_REG_AX), 'ARGS=%04X:%04X' % (m.rw((m.reg(UC_X86_REG_SS)<<4)+m.reg(UC_X86_REG_SP)+6), m.rw((m.reg(UC_X86_REG_SS)<<4)+m.reg(UC_X86_REG_SP)+4)), file=sys.stderr)
        m.mu.hook_add(UC_HOOK_CODE, at)
    stop = m.run(a.insns)
    print(stop, file=sys.stderr)
    after_rg = region_bytes(m)
    out = dump(m, {'mode': 'call', 'routine': a.routine, 'args': a.args,
                   'returned': stop == 'returned',
                   'arg_bytes_popped': (m.reg(UC_X86_REG_SP) - want_sp) & 0xFFFF})
    out['ax'], out['dx'] = m.reg(UC_X86_REG_AX), m.reg(UC_X86_REG_DX)
    out['diff'] = {'DS': diff_bytes(before_ds, m.ds_bytes())}
    for k in set(before_rg) | set(after_rg):
        out['diff'][k] = diff_bytes(before_rg.get(k, b''), after_rg.get(k, b''))
    out['bufs'] = [bytes(m.mu.mem_read(r, n)).hex() for r, n in bufs]
    del out['memory']['DS']                   # the diff says what changed; full DS on request
    if a.full_ds:
        out['memory']['DS'] = m.ds_bytes().hex()
    if a.out:
        json.dump(out, open(a.out, 'w'))
    else:
        print(json.dumps({k: out[k] for k in ('returned', 'ax', 'dx', 'arg_bytes_popped', 'diff', 'bufs')}, indent=1))
        print(json.dumps(out['trace'][:50]))


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest='mode', required=True)
    for name in ('game', 'call'):
        p = sub.add_parser(name)
        p.add_argument('--seed', type=lambda s: int(s, 0), default=0x12345678)
        p.add_argument('--keys', default='')
        p.add_argument('--insns', type=int, default=500_000_000 if name == 'game' else 50_000_000)
        p.add_argument('--out')
        p.add_argument('-v', '--verbose', action='store_true')
    g = sub.choices['game']
    g.add_argument('--script')
    g.add_argument('--png')
    g.add_argument('--png-at', action='append', default=[], metavar='N:PATH')
    g.add_argument('--save-snap')
    c = sub.choices['call']
    c.add_argument('routine')
    c.add_argument('args', nargs='*')
    c.add_argument('--snap', required=True)
    c.add_argument('--near', action='store_true')
    c.add_argument('--poke', action='append', default=[])
    c.add_argument('--full-ds', action='store_true')
    a = ap.parse_args()
    {'game': cmd_game, 'call': cmd_call}[a.mode](a)


if __name__ == '__main__':
    main()
