import struct, zlib, gzip, glob, os
def _r(b,i,t):
    if t==1: return struct.unpack_from('>b',b,i)[0],i+1
    if t==2: return struct.unpack_from('>h',b,i)[0],i+2
    if t==3: return struct.unpack_from('>i',b,i)[0],i+4
    if t==4: return struct.unpack_from('>q',b,i)[0],i+8
    if t==5: return struct.unpack_from('>f',b,i)[0],i+4
    if t==6: return struct.unpack_from('>d',b,i)[0],i+8
    if t==7:
        n=struct.unpack_from('>i',b,i)[0]; return b[i+4:i+4+n],i+4+n
    if t==8:
        n=struct.unpack_from('>H',b,i)[0]; return b[i+2:i+2+n].decode('utf8','replace'),i+2+n
    if t==9:
        et=b[i]; n=struct.unpack_from('>i',b,i+1)[0]; i+=5; out=[]
        for _ in range(n): v,i=_r(b,i,et); out.append(v)
        return out,i
    if t==10:
        d={}
        while True:
            tt=b[i]; i+=1
            if tt==0: return d,i
            n=struct.unpack_from('>H',b,i)[0]; name=b[i+2:i+2+n].decode('utf8','replace'); i+=2+n
            d[name],i=_r(b,i,tt)
    if t==11:
        n=struct.unpack_from('>i',b,i)[0]; return list(struct.unpack_from('>%di'%n,b,i+4)),i+4+4*n
    if t==12:
        n=struct.unpack_from('>i',b,i)[0]; return list(struct.unpack_from('>%dq'%n,b,i+4)),i+4+8*n
    raise ValueError(t)
def parse(b):
    t=b[0]; n=struct.unpack_from('>H',b,1)[0]; return _r(b,3+n,t)[0]
def region_chunks(path):
    data=open(path,'rb').read()
    if len(data)<8192: return
    for k in range(1024):
        off=int.from_bytes(data[k*4:k*4+3],'big'); 
        if not off: continue
        p=off*4096; ln=struct.unpack_from('>i',data,p)[0]; comp=data[p+4]
        raw=data[p+5:p+4+ln]
        try:
            raw=zlib.decompress(raw) if comp==2 else gzip.decompress(raw) if comp==1 else raw
            yield parse(raw)
        except Exception: pass
