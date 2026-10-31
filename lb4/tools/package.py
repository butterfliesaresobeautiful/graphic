"""Create the distributable and verify a fresh extraction with no Go in PATH."""
from pathlib import Path
import hashlib,json,os,stat,subprocess,tempfile,zipfile

ROOT=Path(__file__).resolve().parents[1]
ZIP=ROOT/'dist/lab4-variant8.zip'

def files():
    return sorted(p for p in ROOT.rglob('*') if p.is_file() and
                  not any(x in {'dist','output','__pycache__','.git'} for x in p.relative_to(ROOT).parts)
                  and p.name!='SHA256SUMS')

def digest(p): return hashlib.sha256(p.read_bytes()).hexdigest()

def pack():
    fs=files()
    (ROOT/'SHA256SUMS').write_text(''.join(f'{digest(p)}  {p.relative_to(ROOT).as_posix()}\n' for p in fs))
    ZIP.parent.mkdir(exist_ok=True)
    with zipfile.ZipFile(ZIP,'w',zipfile.ZIP_DEFLATED,compresslevel=9) as z:
        for p in fs+[ROOT/'SHA256SUMS']:
            rel=p.relative_to(ROOT)
            info=zipfile.ZipInfo.from_file(p,arcname='lab4-variant8/'+rel.as_posix())
            mode=0o755 if rel.parts[0]=='bin' else 0o644
            info.external_attr=(stat.S_IFREG|mode)<<16
            info.compress_type=zipfile.ZIP_DEFLATED
            z.writestr(info,p.read_bytes())
    return len(fs)

def smoke():
    with tempfile.TemporaryDirectory(prefix='lb4-archive-') as td:
        with zipfile.ZipFile(ZIP) as z:
            assert z.testzip() is None
            z.extractall(td)
            for item in z.infolist():
                os.chmod(Path(td)/item.filename, (item.external_attr>>16)&0o777)
        base=Path(td)/'lab4-variant8'
        checks=(base/'SHA256SUMS').read_text().splitlines()
        for line in checks:
            sha,rel=line.split('  ',1)
            assert digest(base/rel)==sha,rel
        run=subprocess.run([str(base/'bin/lab4-linux-amd64'),'-mode','batch',
                            '-out',str(base/'output')],
                            cwd=base,env={'PATH':'/nonexistent','DISPLAY':os.environ.get('DISPLAY',''),'HOME':os.environ.get('HOME',''),'XAUTHORITY':os.environ.get('XAUTHORITY','')},check=True,text=True,capture_output=True)
        matches={}
        for name in ['equation.png','parametric.png','bresenham.png','native.png','overlay.png','difference.png','zoom.png']:
            matches[name]=(base/'images'/name).read_bytes()==(base/'output'/name).read_bytes()
            assert matches[name],name
        return {'verified_checksums':len(checks),'zip_crc':'PASS','amd64_exit_code':run.returncode,
                'runtime_path':'/nonexistent','png_byte_matches':matches,
                'runtime_stdout':run.stdout,'arm64_executed':False,'test':'fresh ZIP extraction'}

pack()
result=smoke()
(ROOT/'validation/archive-smoke.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
count=pack()
final=smoke()
assert final['verified_checksums']==count
print(json.dumps({'archive':str(ZIP),'size_bytes':ZIP.stat().st_size,'sha256':digest(ZIP),
                  'final_archive_check':final},ensure_ascii=False,indent=2))
