import sys,re,os
out=os.path.join(os.path.dirname(os.path.abspath(__file__)),'..','out')
n=int(sys.argv[1][3:]); b=open(out+'/ovr/ovr%02d.bin'%n,'rb').read()
for m in re.finditer(rb'[\x20-\x7e]{3,}',b):
    o=m.start()
    if o>0 and b[o-1]==len(m.group()): print('%04X %r'%(o-1,m.group().decode()))
