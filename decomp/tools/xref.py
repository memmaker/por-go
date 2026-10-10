"""Collect far-call targets from all disassembly listings: who calls what, how often."""
import re,glob,collections,json,os
out=os.path.join(os.path.dirname(__file__),'..','out')
calls=collections.defaultdict(collections.Counter)
ovs={o['n']:o for o in json.load(open(out+'/overlays.json'))}
seg2ovr={o['seg']:o['n'] for o in ovs.values()}
for f in sorted(glob.glob(out+'/asm/*.asm')):
    name=os.path.basename(f)[:-4]
    for line in open(f,errors='replace'):
        m=re.search(r'lcall 0x([0-9a-f]+):0x([0-9a-f]+)',line)
        if m:
            seg,ofs=int(m.group(1),16),int(m.group(2),16)
            if seg in seg2ovr:  # call into an overlay stub: resolve entry index
                o=ovs[seg2ovr[seg]]
                idx=(ofs-0x20)//5 if ofs>=0x20 else -1
                tgt='ovr%02d:%04X'%(o['n'],o['entries'][idx]) if 0<=idx<len(o['entries']) else 'ovr%02d:stub%04X'%(o['n'],ofs)
            else:
                tgt='%04X:%04X'%(seg,ofs)
            calls[tgt][name]+=1
rows=sorted(calls.items(),key=lambda kv:-sum(kv[1].values()))
with open(out+'/xref.tsv','w') as w:
    for tgt,c in rows:
        w.write('%s\t%d\t%s\n'%(tgt,sum(c.values()),' '.join('%s:%d'%kv for kv in sorted(c.items()))))
print(len(rows),'targets'); print(open(out+'/xref.tsv').read()[:6000])
