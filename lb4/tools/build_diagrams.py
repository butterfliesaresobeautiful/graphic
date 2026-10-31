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
def chain(d,pairs,cx=350):
 for a,b in pairs:d.edge([(cx,a),(cx,b)])

d=Diagram(1350)
for args in [
 ('end',230,15,240,45,'Начало'),
 ('io',120,95,460,75,'Ввод ширины w, центра, R;\nчтение фона или белый холст'),
 ('diamond',160,205,380,100,'Данные\nдопустимы?'),
 ('box',120,350,460,75,'A, B, C; три дуги радиуса w\nпо 60° / одна окружность'),
 ('sub',120,465,460,65,'Маска 1: уравнение окружности'),
 ('sub',120,570,460,65,'Маска 2: параметрическое'),
 ('sub',120,675,460,65,'Маска 3: Брезенхем'),
 ('sub',120,780,460,75,'Маска 4: X11 PolyArc\nна отдельном растре сервера'),
 ('box',120,895,460,75,'Попиксельное сравнение масок;\nналожение на исходный фон'),
 ('io',120,1010,460,70,'Показ / запись PNG и метрик'),
 ('io',650,1010,230,70,'Сообщить\nоб ошибке'),
 ('end',230,1250,240,45,'Конец'),
]:d.block(*args)
chain(d,[(60,95),(170,205),(305,350),(425,465),(530,570),(635,675),(740,780),(855,895),(970,1010),(1080,1250)])
d.text(390,333,'Да',20)
d.edge([(540,255),(765,255),(765,1010)],'Нет',595,242)
d.edge([(580,817),(765,817)],'Ошибка',632,805)
d.edge([(580,1045),(650,1045)],'Ошибка',614,1031)
d.edge([(765,1080),(765,1272),(470,1272)])
d.save('algorithm-main')

d=Diagram(1220)
for args in [
 ('end',220,15,260,45,'Начало: одна дуга'),
 ('box',150,100,400,65,'x = 0; L = ceil(R / √2)'),
 ('diamond',220,210,260,100,'x ≤ L?'),
 ('box',130,355,440,75,'y = round(√(R² − x²))'),
 ('diamond',220,475,260,100,'x ≤ y?'),
 ('box',120,620,460,110,'Восемь симметричных точек;\nоставить точки в углах дуги;\nперенести к центру и округлить'),
 ('box',120,775,460,75,'Записать в маску\nс проверкой границ холста'),
 ('box',230,895,240,50,'x = x + 1'),
 ('box',120,1030,460,65,'Добавить концы неполной дуги'),
 ('end',230,1150,240,45,'Конец'),
]:d.block(*args)
chain(d,[(60,100),(165,210),(310,355),(430,475),(575,620),(730,775),(850,895),(1095,1150)])
d.text(390,337,'Да',20);d.text(390,602,'Да',20)
d.edge([(480,260),(790,260),(790,1062),(580,1062)],'Нет',535,246)
d.edge([(480,525),(790,525)],'Нет',535,511)
d.edge([(230,920),(70,920),(70,260),(220,260)])
d.save('algorithm-equation')

d=Diagram(1270)
for args in [
 ('end',210,15,280,45,'Начало: одна дуга'),
 ('box',120,100,460,95,'N = max(1, ceil(R · β · π / 180))\nk = 0'),
 ('diamond',220,240,260,100,'k ≤ N?'),
 ('box',120,385,460,100,'t = (α + β · k / N) · π / 180\nx = round(cx + R cos t)\ny = round(cy + R sin t)'),
 ('diamond',220,530,260,100,'k = 0?'),
 ('box',80,690,340,75,'Записать первый пиксель'),
 ('box',500,690,360,100,'Соединить предыдущий\nи текущий пиксели\nцелочисленным отрезком'),
 ('box',120,865,460,75,'Запомнить текущий пиксель;\nk = k + 1'),
 ('end',230,1160,240,45,'Конец'),
]:d.block(*args)
chain(d,[(60,100),(195,240),(340,385),(485,530)])
d.text(390,367,'Да',20)
d.edge([(220,580),(60,580),(60,727),(80,727)],'Да',175,610)
d.edge([(480,580),(680,580),(680,690)],'Нет',540,565)
d.edge([(250,765),(250,820),(350,820),(350,865)])
d.edge([(680,790),(680,820),(350,820)])
d.edge([(120,902),(40,902),(40,290),(220,290)])
d.edge([(480,290),(880,290),(880,1182),(470,1182)],'Нет',540,275)
d.save('algorithm-parametric')

d=Diagram(1270)
for args in [
 ('end',210,15,280,45,'Начало: одна дуга'),
 ('box',120,100,460,70,'x = 0; y = R; d = 3 − 2R'),
 ('diamond',220,215,260,100,'x ≤ y?'),
 ('box',120,360,460,105,'Восемь симметричных точек;\nотбор по углам дуги; перенос,\nокругление, запись в маску'),
 ('diamond',220,510,260,100,'d < 0?'),
 ('box',70,675,350,75,'d = d + 4x + 6'),
 ('box',500,675,360,100,'d = d + 4(x − y) + 10\ny = y − 1'),
 ('box',230,850,240,50,'x = x + 1'),
 ('box',120,1035,460,65,'Добавить концы неполной дуги'),
 ('end',230,1160,240,45,'Конец'),
]:d.block(*args)
chain(d,[(60,100),(170,215),(315,360),(465,510),(1100,1160)])
d.text(390,342,'Да',20)
d.edge([(220,560),(50,560),(50,712),(70,712)],'Да',175,590)
d.edge([(480,560),(680,560),(680,675)],'Нет',540,545)
d.edge([(245,750),(245,810),(350,810),(350,850)])
d.edge([(680,775),(680,810),(350,810)])
d.edge([(230,875),(40,875),(40,265),(220,265)])
d.edge([(480,265),(880,265),(880,1067),(580,1067)],'Нет',540,250)
d.save('algorithm-bresenham')
