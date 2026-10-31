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
d=Diagram(1380)
for args in [
 ('end',230,20,240,45,'Начало'),
 ('io',120,105,460,75,'Ввод размеров, R и позиций;\nчтение изображения → RGB'),
 ('diamond',160,225,380,100,'Данные\nдопустимы?'),
 ('box',120,370,460,65,'Создать белый холст W × H'),
 ('sub',120,480,460,75,'Перенести круг попиксельно\n(рисунок 2)'),
 ('diamond',170,600,360,100,'Copied > 0?'),
 ('box',120,745,460,75,'Рассчитать область графика,\nмасштабы; добавить оси'),
 ('sub',120,865,460,70,'Построить y = x³\nна x ∈ [−2; 2]'),
 ('io',120,980,460,70,'Сохранить PNG\nчерез временный файл'),
 ('diamond',200,1095,300,90,'Запись успешна?'),
 ('io',650,1095,230,90,'Сообщить\nоб ошибке'),
 ('end',230,1300,240,45,'Конец'),
]:d.block(*args)
for a,b in [(65,105),(180,225),(325,370),(435,480),(555,600),(700,745),(820,865),(935,980),(1050,1095),(1185,1300)]:
 d.edge([(350,a),(350,b)],'Да' if a in [325,700,1185] else None,390,(a+b)/2)
d.edge([(540,275),(765,275),(765,1095)],'Нет',595,260)
d.edge([(530,650),(765,650)],'Нет',590,635)
d.edge([(580,782),(765,782)],'Ошибка',632,769)
d.edge([(500,1140),(650,1140)],'Нет',575,1125)
d.edge([(765,1185),(765,1322),(470,1322)])
d.save('algorithm-main')

d=Diagram(1450)
for args in [
 ('end',220,15,260,45,'Начало переноса'),
 ('box',150,95,400,70,'oy = −R; N = 0; K = 0'),
 ('diamond',230,205,240,90,'oy ≤ R?'),
 ('box',230,335,240,45,'ox = −R'),
 ('diamond',230,420,240,90,'ox ≤ R?'),
 ('diamond',150,550,400,100,'ox² + oy² ≤ R²?'),
 ('box',120,690,460,90,'Pист = Cист + (ox; oy)\nPрез = Cрез + (ox; oy)'),
 ('diamond',120,820,460,110,'Точки внутри\nрастров?'),
 ('box',150,970,400,90,'Считать R, G, B источника;\nзаписать в результат; N++'),
 ('box',650,970,200,90,'Не записывать;\nK++'),
 ('box',230,1105,240,45,'ox = ox + 1'),
 ('box',660,1195,190,45,'oy = oy + 1'),
 ('io',120,1280,460,70,'Вернуть N и K;\nесли N = 0 — ошибка'),
 ('end',230,1390,240,45,'Конец переноса'),
]:d.block(*args)
for a,b in [(60,95),(165,205),(295,335),(380,420),(510,550),(650,690),(780,820),(930,970),(1060,1105),(1350,1390)]:
 d.edge([(350,a),(350,b)],'Да' if a in [295,510,650,930] else None,390,(a+b)/2)
d.edge([(230,250),(40,250),(40,1315),(120,1315)],'Нет',185,235)
d.edge([(470,465),(865,465),(865,1165),(755,1165),(755,1195)],'Нет',520,450)
d.edge([(850,1217),(890,1217),(890,250),(470,250)])
d.edge([(550,600),(610,600),(610,1127),(470,1127)],'Нет',585,585)
d.edge([(580,875),(750,875),(750,970)],'Нет',632,860)
d.edge([(750,1060),(750,1127),(610,1127)])
d.edge([(230,1127),(90,1127),(90,465),(230,465)])
d.save('algorithm-transfer')
