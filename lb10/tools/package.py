"""Package the lab and replay the examples from a fresh extraction."""
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
ARCHIVE = ROOT / "dist/lab10-variant12.zip"
CHECKSUMS = ROOT / "SHA256SUMS"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def pack():
    excluded = {"dist", "output", "__pycache__", ".git"}
    paths = sorted(
        p for p in ROOT.rglob("*")
        if p.is_file() and p != CHECKSUMS
        and not excluded.intersection(p.relative_to(ROOT).parts)
    )
    CHECKSUMS.write_text("".join(
        f"{digest(p)}  {p.relative_to(ROOT).as_posix()}\n" for p in paths
    ))
    ARCHIVE.parent.mkdir(exist_ok=True)
    with zipfile.ZipFile(ARCHIVE, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for path in paths + [CHECKSUMS]:
            rel = path.relative_to(ROOT)
            info = zipfile.ZipInfo.from_file(path, arcname=f"lb10/{rel.as_posix()}")
            mode = 0o755 if rel.parts[0] == "bin" else 0o644
            info.external_attr = (stat.S_IFREG | mode) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, path.read_bytes())
    return len(paths)


def smoke():
    with tempfile.TemporaryDirectory(prefix="lb10 extraction ") as temp:
        directory = Path(temp)
        with zipfile.ZipFile(ARCHIVE) as archive:
            assert archive.testzip() is None
            archive.extractall(temp)
            for item in archive.infolist():
                (directory / item.filename).chmod((item.external_attr >> 16) & 0o777)
        lab = directory / "lb10"
        lines = (lab / CHECKSUMS.name).read_text().splitlines()
        for line in lines:
            sha, rel = line.split("  ", 1)
            assert digest(lab / rel) == sha, rel
        run = subprocess.run(
            [str(lab / "bin/lab10-linux-amd64"), "-mode", "demo"],
            cwd=directory, env={"PATH": "/nonexistent"}, check=True,
            text=True, capture_output=True, timeout=30,
        )
        names = ["quadratic.png", "cubic.png", "surface.png", "metrics.json"]
        for name in names:
            assert (lab / "images" / name).read_bytes() == (lab / "output" / name).read_bytes(), name
        return {
            "verified_checksums": len(lines),
            "zip_crc": "PASS",
            "amd64_exit_code": run.returncode,
            "runtime_path": "/nonexistent",
            "display_present": False,
            "png_byte_matches": 3,
            "json_byte_matches": 1,
        }


pack()
result = smoke()
record = ROOT / "validation/archive-smoke.json"
record.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
count = pack()
result = smoke()
if result["verified_checksums"] != count:
    raise RuntimeError("unexpected number of verified files")
if json.loads(record.read_text()) != result:
    record.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    count = pack()
    assert smoke() == result
print(json.dumps({
    "archive": str(ARCHIVE),
    "size_bytes": ARCHIVE.stat().st_size,
    "sha256": digest(ARCHIVE),
    "verification": result,
}, ensure_ascii=False, indent=2))
