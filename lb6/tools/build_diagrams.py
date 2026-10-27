"""Generate print-sized technical flowcharts; document tooling only."""
from pathlib import Path
from html import escape
import subprocess
import base64

OUT = Path(__file__).resolve().parents[1] / "report"


class Diagram:
    def __init__(self, height):
        self.parts = [f'''<svg xmlns="http://www.w3.org/2000/svg" width="900" height="{height}" viewBox="0 0 900 {height}">
<defs><marker id="arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M0 0L9 4.5L0 9Z" fill="#243449"/></marker></defs>
<rect width="900" height="{height}" fill="white"/>
<g font-family="DejaVu Sans" font-size="22" stroke="#243449" stroke-width="2" fill="white">''']

    def text(self, x, y, lines, size=22):
        for i, line in enumerate(lines.split("\n")):
            self.parts.append(f'<text x="{x}" y="{y+i*27}" text-anchor="middle" fill="#172333" stroke="none" font-size="{size}">{escape(line)}</text>')

    def block(self, kind, x, y, w, h, label):
        if kind == "diamond":
            s = f'<path d="M{x+w/2} {y}L{x+w} {y+h/2}L{x+w/2} {y+h}L{x} {y+h/2}Z"/>'
        elif kind == "io":
            s = f'<path d="M{x+22} {y}H{x+w}L{x+w-22} {y+h}H{x}Z"/>'
        elif kind == "end":
            s = f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="{h/2}"/>'
        else:
            s = f'<rect x="{x}" y="{y}" width="{w}" height="{h}"/>'
            if kind == "sub":
                s += f'<path d="M{x+9} {y}V{y+h}M{x+w-9} {y}V{y+h}"/>'
        self.parts.append(s)
        self.text(x+w/2, y+h/2+8-(len(label.split("\n"))-1)*13.5, label)

    def edge(self, points, label=None, lx=0, ly=0):
        coords = " ".join(f"{x},{y}" for x, y in points)
        self.parts.append(f'<polyline points="{coords}" fill="none" marker-end="url(#arrow)"/>')
        if label:
            self.text(lx, ly, label, 20)

    def save(self, name):
        path = OUT / (name + ".svg")
        path.write_text("\n".join(self.parts) + "</g></svg>", encoding="utf-8")
        subprocess.run(["inkscape", str(path), "--export-type=png",
                        "--export-filename=" + str(OUT / (name + ".png")),
                        "--export-width=1800"], check=True, capture_output=True)

d = Diagram(1360)
for args in [
 ("end",240,20,220,46,"Начало"),
 ("io",130,100,440,75,"Чтение файла, угла, оси;\nприведение растра к RGB"),
 ("diamond",170,210,360,110,"Входные данные\nкорректны?"),
 ("box",130,360,440,75,"Матрица A: поворот, отражение\nотносительно центра холста"),
 ("sub",130,475,440,75,"Аффинный растр: Warp(S, A)\nвыборка S по A⁻¹"),
 ("sub",130,590,440,75,"Восстановление: Warp(T, A⁻¹)\nвыборка аффинного T по A"),
 ("sub",130,705,440,75,"Нелинейный растр из S:\nвыборка по x′ = i^(3/5), y′ = j"),
 ("io",130,820,440,75,"Запись трёх PNG\nи показ результатов"),
 ("diamond",190,940,320,100,"Запись\nуспешна?"),
 ("io",635,1120,235,85,"Сообщить\nоб ошибке"),
 ("end",240,1270,220,46,"Конец"),
]: d.block(*args)
for a,b in [(66,100),(175,210),(320,360),(435,475),(550,590),(665,705),(780,820),(895,940),(1040,1270)]:
 d.edge([(350,a),(350,b)], "Да" if a in (320,1040) else None,390,(a+b)/2)
d.edge([(530,265),(750,265),(750,1120)],"Нет",600,250)
d.edge([(510,990),(750,990)],"Нет",590,976)
d.edge([(750,1205),(750,1293),(460,1293)])
d.save('algorithm-main')

d = Diagram(1460)
for args in [
 ("end",245,15,240,45,"Начало выборки"),
 ("box",145,95,440,75,"Белый холст W × H; y = 0\nN = 0; K = 0"),
 ("diamond",245,210,240,90,"y < H?"),
 ("box",245,340,240,45,"x = 0"),
 ("diamond",245,425,240,90,"x < W?"),
 ("box",145,555,440,75,"(u, v) = back(x, y)\nu = round(u); v = round(v)"),
 ("diamond",145,680,440,120,"(u, v) внутри\nисходного растра?"),
 ("box",165,850,400,90,"Скопировать RGB(u, v)\nв пиксель (x, y)\nN = N + 1"),
 ("box",655,850,210,90,"Оставить белым\nK = K + 1"),
 ("box",245,1000,240,45,"x = x + 1"),
 ("box",655,1110,210,45,"y = y + 1"),
 ("io",145,1245,440,75,"Вернуть растр,\nN выбранных, K вне границ"),
 ("end",245,1370,240,45,"Конец выборки"),
]: d.block(*args)
for a,b in [(60,95),(170,210),(300,340),(385,425),(515,555),(630,680),(800,850),(940,1000),(1320,1370)]:
 d.edge([(365,a),(365,b)],"Да" if a in (300,515,800) else None,405,(a+b)/2)
d.edge([(245,255),(45,255),(45,1282),(145,1282)],"Нет",200,241)
d.edge([(485,470),(875,470),(875,1080),(760,1080),(760,1110)],"Нет",530,456)
d.edge([(865,1132),(895,1132),(895,255),(485,255)])
d.edge([(585,740),(760,740),(760,850)],"Нет",630,726)
d.edge([(760,940),(760,1022),(485,1022)])
d.edge([(245,1022),(100,1022),(100,470),(245,470)])
d.save('algorithm-sampling')
