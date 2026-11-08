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

d=Diagram(1530)
for args in [
 ('end',230,20,240,45,'Начало'),
 ('io',110,100,480,75,'Прочитать RGB-изображение;\nввести Sx, Sy, θ, hx, hy'),
 ('diamond',160,215,380,110,'Параметры\nдопустимы?'),
 ('box',110,365,480,75,'Составить A = R · H · S;\nвычислить A⁻¹'),
 ('box',110,480,480,75,'Преобразовать 4 угла;\nопределить размеры холста'),
 ('diamond',160,595,380,110,'Размеры\nдопустимы?'),
 ('box',110,745,480,75,'Создать 3 белых изображения;\nперебрать выходные пиксели'),
 ('sub',110,860,480,85,'Обратное отображение и\n3 метода интерполяции\n(рисунок 2)'),
 ('box',110,985,480,75,'Вычислить различия методов;\nпосчитать ограниченные каналы'),
 ('io',110,1100,480,75,'Вывести результаты в окно\nили сохранить PNG и JSON'),
 ('diamond',190,1215,320,100,'Успешно?'),
 ('io',650,1215,230,100,'Сообщить\nоб ошибке'),
 ('end',230,1445,240,45,'Конец'),
]:d.block(*args)
for a,b in [(65,100),(175,215),(325,365),(440,480),(555,595),(705,745),(820,860),(945,985),(1060,1100),(1175,1215),(1315,1445)]:
 d.edge([(350,a),(350,b)],'Да' if a in [325,705,1315] else None,390,(a+b)/2)
d.edge([(540,270),(765,270),(765,1215)],'Нет',590,255)
d.edge([(540,650),(765,650)],'Нет',590,635)
d.edge([(510,1265),(650,1265)],'Нет',580,1250)
d.edge([(765,1315),(765,1467),(470,1467)])
d.save('algorithm-main')

d=Diagram(1570)
for args in [
 ('end',230,20,240,45,'Пиксель (u, v)'),
 ('box',110,110,480,90,'Учесть центр пикселя и начало;\nприменить A⁻¹ → (x, y)'),
 ('diamond',150,250,400,120,'Точка внутри\nисточника?'),
 ('box',110,420,480,80,'Отметить общую маску;\nвыбрать метод'),
 ('box',110,550,480,90,'Во всех методах: индексы\nза границей ограничивать краем'),
 ('sub',110,690,480,90,'Сосед: округлить x и y;\nвзять один пиксель'),
 ('sub',110,830,480,90,'Билинейный: четыре пикселя;\nвеса (1−a), a, (1−b), b'),
 ('sub',110,970,480,90,'Бикубический: 16 пикселей;\nвеса wᵢ(a) · wⱼ(b)'),
 ('box',110,1110,480,90,'Ограничить RGB до 0…255,\nокруглить и записать 3 результата'),
 ('box',650,420,230,80,'Оставить\nбелый фон'),
 ('end',200,1430,300,65,'Следующий пиксель'),
]:d.block(*args)
for a,b in [(65,110),(200,250),(370,420),(500,550),(640,690),(780,830),(920,970),(1060,1110),(1200,1430)]:
 d.edge([(350,a),(350,b)],'Да' if a==370 else None,390,(a+b)/2)
d.edge([(550,310),(765,310),(765,420)],'Нет',610,295)
d.edge([(765,500),(765,1462),(500,1462)])
d.save('algorithm-sampling')
