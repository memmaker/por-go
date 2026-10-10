"""Compare an emulator trace with a Go trace: first divergence of the leaf records.

  compare.py emu.json go.json [--only NAME,NAME] [--context 5]
By default only address-keyed records and INT16 ReadKey are compared
(the Go side has no BIOS/DOS records). Strings compare case-insensitively (LoadPic names differ in case).
"""
import argparse, json, re

def norm(r):
    if r[1] in ('LoadPic', 'FillRect', 'DrawPictureMasked', 'DrawPage'):
        # far pointers are machine specific (Go emits "")
        r = [x for x in r[:2]] + [x for x in r[2:] if not (isinstance(x, str) and (x == '' or re.fullmatch(r'[0-9A-F]{4}:[0-9A-F]{4}', x)))]
    return [x.upper() if isinstance(x, str) else x for x in r]

# far pointers inside the records (machine specific): record prefix -> offsets
PTRS = {'char': (0x7F, 0xC8, 0xCC, 0x104, 0x108), 'sele': (0x7F, 0xC8, 0xCC, 0x104, 0x108)}   # affect list, item list heads, next character, combat block (Go keeps slices)


def collapse(t):
    """Runs of identical KeyPressed polls count once (wait loops spin a machine-dependent number of times)."""
    out = []
    for r in t:
        if r[1] in ('KEYPRESSED', 'TICKS') and out and out[-1] == r:
            continue
        out.append(r)
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('a'); ap.add_argument('b')
    ap.add_argument('--only', default='')
    ap.add_argument('--context', type=int, default=5)
    a = ap.parse_args()
    only = set(a.only.split(',')) - {''}
    ta, tb = (collapse([norm(r) for r in json.load(open(p))['trace'] if (r[1] in only if only else ((':' in r[0] and not r[0].startswith('08AE')) or r[0] == 'INT16'))]) for p in (a.a, a.b))
    for i, (x, y) in enumerate(zip(ta, tb)):
        if x != y:
            print('diverge at record %d' % i)
            for k in range(max(0, i - a.context), min(i + a.context, len(ta), len(tb))):
                print('%s %5d  %-60s %s' % ('>' if k == i else ' ', k, json.dumps(ta[k])[:60], json.dumps(tb[k])))
            return 1
    print('identical over %d records (lengths %d / %d)' % (min(len(ta), len(tb)), len(ta), len(tb)))
    ma, sb = json.load(open(a.a)).get('memory', {}), json.load(open(a.b)).get('state', {})
    bad = 0
    for k in sorted(sb):
        x, y = bytes.fromhex(ma.get(k, {}).get('data', '')), bytes.fromhex(sb[k])
        skip = list(PTRS.get(k[:4], ()))
        if k[:4] in PTRS:
            # ponytail: any word-aligned far pointer into the heap (seg >= 0x1000)
            # where Go keeps 0 is a list link Go holds in a slice; a fixed
            # offset table would be stricter
            skip += [o for o in range(0, len(x) - 3, 2) if y[o:o + 4] == bytes(4) and int.from_bytes(x[o + 2:o + 4], 'little') >= 0x1000]
        diffs = [i for i in range(min(len(x), len(y))) if x[i] != y[i] and not any(o <= i < o + 4 for o in skip)]
        if diffs or len(x) != len(y):
            bad += 1
            print('state %s differs at %d bytes: %s' % (k, len(diffs), ' '.join('%X:%02X/%02X' % (i, x[i], y[i]) for i in diffs[:16])))
    print('state: %d records compared, %d differ' % (len(sb), bad))
    return 1 if bad else 0

if __name__ == '__main__':
    raise SystemExit(main())
