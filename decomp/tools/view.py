"""Annotated listing viewer.
usage: view.py ovr02 [start_hex] [end_hex]   or   view.py start seg:ofs [len]
Annotates: lcall targets with names from names.json, 'mov di,X; push cs; push di' string refs with the literal text,
near calls, and pascal strings in data ranges.
"""
import sys,re,json,os
out=os.path.join(os.path.dirname(os.path.abspath(__file__)),'..','out')
names={}
try: names=json.load(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'names.json')))
except Exception: pass
ovs={o['n']:o for o in json.load(open(out+'/overlays.json'))}
seg2ovr={o['seg']:o['n'] for o in ovs.values()}
def pstr(b,o):
    if o>=len(b): return ''
    n=b[o]; return b[o+1:o+1+n].decode('latin1')
def resolve(seg,ofs):
    if seg in seg2ovr:
        o=ovs[seg2ovr[seg]]; idx=(ofs-0x20)//5
        if 0<=idx<len(o['entries']): return 'ovr%02d:%04X'%(o['n'],o['entries'][idx])
        return 'ovr%02d:stub'%o['n']
    return '%04X:%04X'%(seg,ofs)
def main():
    what=sys.argv[1]
    if what=='start':
        seg,ofs=[int(x,16) for x in sys.argv[2].split(':')]
        length=int(sys.argv[3],16) if len(sys.argv)>3 else 0x200
        b=open(out+'/START.bin','rb').read()
        base=seg*16; lo=base+ofs; hi=lo+length
        # start.asm is a clean linear listing of the whole image (r2 with a seek desyncs)
        lines=[]
        for ln in open(out+'/asm/start.asm',errors='replace'):
            m=re.match(r'\s*0000:([0-9a-f]+)\s+[0-9a-f.]+\s+(.*)',ln)
            if not m: continue
            a=int(m.group(1),16)
            if lo<=a<hi: lines.append((a-base,m.group(2)))
        segbase=base; data=b
        def strat(o): return pstr(data,segbase+o)
    else:
        n=int(what[3:]); o=ovs[n]
        b=open(out+'/ovr/ovr%02d.bin'%n,'rb').read()
        lo=int(sys.argv[2],16) if len(sys.argv)>2 else 0
        hi=int(sys.argv[3],16) if len(sys.argv)>3 else len(b)
        import subprocess
        txt=subprocess.run(['r2','-q','-a','x86','-b','16','-e','scr.color=0','-e','asm.lines=false','-e','asm.bytes=false','-c','s %d; pD %d'%(lo,hi-lo),out+'/ovr/ovr%02d.bin'%n],capture_output=True,text=True).stdout
        lines=[]
        for ln in txt.splitlines():
            m=re.match(r'\s*0000:([0-9a-f]+)\s+(.*)',ln)
            if not m: continue
            lines.append((int(m.group(1),16),m.group(2)))
        ents={e:i for i,e in enumerate(o['entries'])}
        def strat(x): return pstr(b,x)
        segbase=None
    pend=None
    for a,ins in lines:
        ins=re.sub(r'\s{2,};.*','',ins).strip()
        if ins in ('push ss','push cs'): continue
        note=''
        if what!='start' and a in ents: print('; ---- ENTRY %d  %s'%(ents[a], names.get('ovr%02d:%04X'%(n,a),'')))
        if what=='start':
            mj=re.match(r'(j\w+|loop\w*) 0x([0-9a-f]+)$',ins)
            if mj: ins='%s 0x%x'%(mj.group(1),int(mj.group(2),16)-base)
        m=re.search(r'lcall 0x([0-9a-f]+):0x([0-9a-f]+)',ins)
        if m:
            t=resolve(int(m.group(1),16),int(m.group(2),16)); note=t+' '+names.get(t,'')
        m=re.match(r'mov di, 0x([0-9a-f]+)',ins)
        if m: pend=int(m.group(1),16)
        if ins=='push di' and pend is not None:
            s=strat(pend)
            if s: note='"%s"'%s
            pend=None
        m=re.match(r'call 0x([0-9a-f]+)',ins)
        if m and what!='start':
            t='ovr%02d:%04X'%(n,int(m.group(1),16)); note=names.get(t,'')
        if m and what=='start':
            t='%04X:%04X'%(seg,int(m.group(1),16)-base)
            ins=ins.replace(m.group(0),'call %04X'%(int(m.group(1),16)-base)); note=t+' '+names.get(t,'')
        m=re.search(r'\[0x([0-9a-f]{4})\]',ins)
        if m:
            v='DS:'+m.group(1).upper(); 
            if v in names: note+=' '+names[v]
        print('%04X  %-40s %s'%(a,ins,('; '+note) if note else ''))
main()

# strings mode: python3 view.py strings ovrNN  -> list pascal string literals (len byte + printable text)
if __name__=='__main__' and len(sys.argv)>2 and sys.argv[1]=='strings':
    pass
