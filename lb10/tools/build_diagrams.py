"""Generate print-sized technical flowcharts; document tooling only."""
from pathlib import Path
from html import escape
import subprocess

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

d = Diagram(1030)
for args in [
    ('end', 225, 20, 250, 50, 'Начало'),
    ('io', 95, 110, 510, 75, 'Выбрать режим, ввести вершины\nи параметры разбиения'),
    ('diamond', 172, 230, 356, 108, 'Данные\nдопустимы?'),
    ('io', 625, 244, 232, 78, 'Сообщить\nоб ошибке'),
    ('sub', 95, 390, 510, 80, 'Вычислить кривую или поверхность\nи контрольные вершины разбиения'),
    ('io', 95, 526, 510, 85, 'Показать результат, этапы и\nвыбранные контрольные сетки'),
    ('diamond', 172, 670, 356, 106, 'Пользователь\nзакрыл окно?'),
    ('end', 225, 875, 250, 50, 'Конец'),
]:
    d.block(*args)
chain(d, [(70, 110), (185, 230), (338, 390), (470, 526), (611, 670), (776, 875)])
d.text(390, 365, 'Да', 20)
d.text(390, 815, 'Да', 20)
d.edge([(528, 283), (625, 283)], 'Нет', 572, 268)
d.edge([(741, 244), (741, 146), (595, 146)])
d.edge([(172, 723), (48, 723), (48, 146), (106, 146)], 'Нет', 105, 707)
d.save('algorithm-main')

d = Diagram(1110)
for args in [
    ('end', 225, 20, 250, 50, 'Начало'),
    ('io', 90, 110, 520, 80, 'Список вершин P, степень d,\nпараметр t, число интервалов N'),
    ('box', 90, 235, 520, 75, 'Сформировать сегменты по d + 1\nвершин с общей конечной точкой'),
    ('diamond', 150, 357, 400, 105, 'Последний сегмент\nнеполный?'),
    ('box', 606, 368, 252, 80, 'Повысить степень\nпоследнего сегмента'),
    ('box', 90, 512, 520, 84, 'Для каждого сегмента вычислить\nB(k/N) методом де Кастельжо'),
    ('box', 90, 642, 520, 80, 'Для выбранного сегмента вычислить\nвсе уровни для параметра t'),
    ('box', 90, 765, 520, 80, 'Построить две контрольные ломаные\nиз крайних точек уровней'),
    ('io', 90, 892, 520, 70, 'Вывести ломаную отсчётов и этапы'),
    ('end', 225, 1010, 250, 50, 'Конец'),
]:
    d.block(*args)
chain(d, [(70, 110), (190, 235), (310, 357), (462, 512), (596, 642),
          (722, 765), (845, 892), (962, 1010)])
d.text(320, 491, 'Нет', 20)
d.edge([(550, 409), (606, 409)], 'Да', 576, 393)
d.edge([(732, 448), (732, 486), (350, 486)])
d.save('algorithm-curve')

d = Diagram(1060)
for args in [
    ('end', 225, 20, 250, 50, 'Начало'),
    ('io', 84, 110, 532, 80, 'Контрольная сетка P,\nпараметры u, v и числа Nᵤ, Nᵥ'),
    ('box', 84, 235, 532, 88, 'Вычислить базис Бернштейна\nдля каждого направления'),
    ('box', 84, 370, 532, 80, 'Суммированием получить S(u, v)\nи сетку (Nᵤ + 1) × (Nᵥ + 1)'),
    ('box', 84, 496, 532, 78, 'Разделить каждую строку в u\nи собрать две сетки'),
    ('box', 84, 620, 532, 78, 'Разделить столбцы этих сеток в v\nи собрать четыре участка'),
    ('io', 84, 748, 532, 84, 'Показать поверхность, точку S(u, v)\nи выбранные контрольные сетки'),
    ('end', 225, 899, 250, 50, 'Конец'),
]:
    d.block(*args)
chain(d, [(70, 110), (190, 235), (323, 370), (450, 496), (574, 620),
          (698, 748), (832, 899)])
d.save('algorithm-surface')
