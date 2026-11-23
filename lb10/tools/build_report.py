"""Document formatting only. Not used by the Go application."""
from pathlib import Path
import json
from PIL import Image
from docx import Document
from docx.shared import Mm, Pt, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.oxml import OxmlElement
from docx.oxml.ns import qn

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "report"
doc = Document()
md = []
sec = doc.sections[0]
sec.page_width, sec.page_height = Mm(210), Mm(297)
sec.left_margin, sec.right_margin = Mm(30), Mm(15)
sec.top_margin, sec.bottom_margin = Mm(20), Mm(20)
sec.header_distance, sec.footer_distance = Mm(8), Mm(10)

normal = doc.styles["Normal"]
normal.font.name, normal.font.size = "Times New Roman", Pt(14)
normal.paragraph_format.line_spacing = 1.5
normal.paragraph_format.first_line_indent = Mm(12.5)
normal.paragraph_format.space_after = Pt(0)
normal.paragraph_format.widow_control = True
normal.element.rPr.rFonts.set(qn("w:eastAsia"), "Times New Roman")
lang = OxmlElement("w:lang")
lang.set(qn("w:val"), "ru-RU")
normal.element.rPr.append(lang)
for name in ("Heading 1", "Heading 2", "Heading 3"):
    st = doc.styles[name]
    st.font.name, st.font.size = "Times New Roman", Pt(14)
    st.font.color.rgb = RGBColor(0, 0, 0)
    st.font.bold = True
    for attr in ("asciiTheme", "hAnsiTheme", "eastAsiaTheme", "cstheme"):
        st.element.rPr.rFonts.attrib.pop(qn("w:" + attr), None)
    st.paragraph_format.first_line_indent = Mm(0)
    st.paragraph_format.line_spacing = 1.5
    st.paragraph_format.space_before = Pt(10)
    st.paragraph_format.space_after = Pt(6)
    st.paragraph_format.keep_with_next = True
code_style = doc.styles.add_style("Listing", 1)
code_style.font.name, code_style.font.size = "DejaVu Sans Mono", Pt(9)
code_style.paragraph_format.line_spacing = 1
code_style.paragraph_format.first_line_indent = Mm(0)
code_style.paragraph_format.space_before = Pt(0)
code_style.paragraph_format.space_after = Pt(0)
code_style.paragraph_format.widow_control = False
code_style.paragraph_format.tab_stops.add_tab_stop(Mm(7))
footer = sec.footer.paragraphs[0]
footer.alignment = WD_ALIGN_PARAGRAPH.CENTER
footer.paragraph_format.first_line_indent = Mm(0)
field = OxmlElement("w:fldSimple")
field.set(qn("w:instr"), "PAGE")
footer._p.append(field)
doc.core_properties.title = "Лабораторная работа № 10. Вариант 12"
doc.core_properties.subject = "Алгоритмы построения кривых и поверхностей"
doc.core_properties.author = "ФИО студента: __________________"
doc.core_properties.keywords = "Go, кривые Безье, поверхность Безье, де Кастельжо"


pending_break=False

def page_break():
    global pending_break
    pending_break=True

def apply_break(p):
    global pending_break
    if pending_break:
        p.paragraph_format.page_break_before=True
        pending_break=False

def paragraph(text, align=None, bold=False):
    p = doc.add_paragraph()
    apply_break(p)
    p.alignment = align or WD_ALIGN_PARAGRAPH.JUSTIFY
    run = p.add_run(text)
    run.bold = bold
    md.append(text + "\n")
    return p


def heading(text, level=1, new_page=False):
    if new_page:
        page_break()
    p=doc.add_heading(text, level)
    apply_break(p)
    md.append("#" * level + " " + text + "\n")


def formula(text):
    p = paragraph(text, WD_ALIGN_PARAGRAPH.CENTER)
    p.paragraph_format.first_line_indent = Mm(0)


def figure(path, caption, max_h=215):
    path = ROOT / path
    im = Image.open(path)
    # Leave a small safety inset for Word/LibreOffice inline-picture metrics.
    width = min(158, max_h * im.width / im.height)
    p = doc.add_paragraph()
    apply_break(p)
    p.paragraph_format.first_line_indent = Mm(0)
    p.paragraph_format.line_spacing = 1
    p.paragraph_format.keep_with_next = True
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    picture = p.add_run().add_picture(str(path), width=Mm(width))
    # A border also makes the completely white intermediate canvas visible.
    border = OxmlElement("a:ln")
    border.set("w", "6350")
    fill = OxmlElement("a:solidFill")
    colour = OxmlElement("a:srgbClr")
    colour.set("val", "C0C0C0")
    fill.append(colour)
    border.append(fill)
    picture._inline.graphic.graphicData.pic.spPr.append(border)
    p = paragraph(caption, WD_ALIGN_PARAGRAPH.CENTER)
    p.paragraph_format.first_line_indent = Mm(0)
    p.paragraph_format.space_after = Pt(6)
    md.append(f"![{caption}]({path.relative_to(OUT) if path.is_relative_to(OUT) else '../' + str(path.relative_to(ROOT))})\n")


def code(path, index):
    heading(f"Листинг 3.{index} — {path}", 2)
    contents = (ROOT / path).read_text(encoding="utf-8")
    for line in contents.expandtabs(4).splitlines():
        p = doc.add_paragraph(style="Listing")
        p.add_run(line if line else "")
    md.append("```" + ("html" if path.endswith(".html") else "go") + "\n" + contents + "```\n")



def table(headers,rows):
    t=doc.add_table(rows=1,cols=len(headers));t.style='Table Grid'
    for cell,value in zip(t.rows[0].cells,headers):cell.text=str(value)
    for row in rows:
        for cell,value in zip(t.add_row().cells,row):cell.text=str(value)
    for ri,row in enumerate(t.rows):
        for cell in row.cells:
            for p in cell.paragraphs:
                p.paragraph_format.first_line_indent=Mm(0);p.paragraph_format.line_spacing=1
                p.alignment=WD_ALIGN_PARAGRAPH.CENTER
                for run in p.runs:
                    run.font.name='Times New Roman';run.font.size=Pt(12);run.bold=ri==0
    md.append('| '+' | '.join(headers)+' |\n|'+ '|'.join(['---']*len(headers))+'|\n'+''.join('| '+' | '.join(map(str,row))+' |\n' for row in rows))
    doc.add_paragraph()


GOAL = (
    'Изучить математические методы и выполнить программную реализацию '
    'построения кривых и поверхностей.'
)
CONCLUSION = (
    'В ходе выполненной работы мы изучили математические методы и разработали '
    'программу построения кривых и поверхностей Безье для варианта 12.'
)
metrics = json.loads((ROOT / 'images/metrics.json').read_text())

md.append('# Титульный лист\n')
for text in [
    'Минобрнауки России',
    'Юго-Западный государственный университет',
    'Кафедра программной инженерии',
]:
    p = paragraph(text, WD_ALIGN_PARAGRAPH.CENTER)
    p.paragraph_format.first_line_indent = Mm(0)
for _ in range(4):
    doc.add_paragraph()
for text in [
    'ОТЧЁТ',
    'по лабораторной работе № 10',
    'по дисциплине «Компьютерная графика»',
    '«Алгоритмы построения кривых и поверхностей»',
    'Индивидуальный номер 28. Вариант 12',
]:
    p = paragraph(text, WD_ALIGN_PARAGRAPH.CENTER, text == 'ОТЧЁТ')
    p.paragraph_format.first_line_indent = Mm(0)
for _ in range(3):
    doc.add_paragraph()
for text in [
    'Выполнил: студент группы __________',
    'ФИО ____________________________',
    'Проверил: _______________________',
]:
    p = paragraph(text, WD_ALIGN_PARAGRAPH.RIGHT)
    p.paragraph_format.first_line_indent = Mm(0)
for _ in range(3):
    doc.add_paragraph()
p = paragraph('Курск 2026 г.', WD_ALIGN_PARAGRAPH.CENTER)
p.paragraph_format.first_line_indent = Mm(0)

heading('1 Задание', new_page=True)
heading('Цель работы', 2)
paragraph(GOAL)
heading('Индивидуальное задание', 2)
paragraph('Источник задания — Мельник Е. В. Алгоритмы построения кривых и поверхностей. Курск, ЮЗГУ, 2014. 20 с. Цель приведена на с. 4, условия и вопросы — на с. 18, таблица вариантов — на с. 19. В комплект включён исходный файл m2664.pdf.')
paragraph('Индивидуальный номер 28. Максимальный номер в таблице 16. По схеме выбора варианта остаток от деления вычисляется по формуле')
formula('28 mod 16 = 12.')
paragraph('По строке 12 выполняются пункты 1, 2 и 4. Пункт 6 обязателен для каждого варианта.')
table(
    ['Пункт', 'Что должно делать приложение'],
    [
        ('1', 'Строить квадратичную кривую по заданному списку вершин. Показывать промежуточные вершины по этапам и принимать параметр t.'),
        ('2', 'Строить кубическую кривую с теми же средствами задания вершин, просмотра этапов и выбора t.'),
        ('4', 'Строить поверхность Безье по прямоугольной сетке заданного размера. Принимать параметры u и v.'),
        ('6', 'Показывать по выбору исходные и уточнённые контрольные вершины. Принимать количество интервалов разбиения.'),
    ],
)
paragraph('Приложение написано на Go. Окно работает через X11, в сеансе Wayland используется XWayland. В комплекте два статических исполняемых файла для Linux x86_64 и ARM64. Шрифт встроен в приложение. Установка Go для запуска не требуется.')

heading('2 Схемы алгоритмов', new_page=True)
heading('2.1 Составные кривые Безье', 2)
paragraph('Кривые в приложении лежат в плоскости OXY. Для сегмента степени d нужен список P₀, …, P_d из d + 1 вершин. При d = 2 берутся три точки, при d = 3 — четыре. Для составной кривой начальные индексы сегментов равны 0, d, 2d и далее. Конечная точка каждого сегмента совпадает с начальной вершиной следующего. Так обеспечивается непрерывность положения C⁰. Для равенства касательных в стыке нужны дополнительные условия на соседние вершины.')
paragraph('Если для последнего сегмента осталось меньше d + 1 вершин, последовательно повышаем степень сегмента. Для входного сегмента степени k с точками P₀, …, P_k сохраняем концы и вычисляем новые внутренние вершины по формуле')
formula('P′ᵢ = (i / (k + 1)) · Pᵢ₋₁ + (1 − i / (k + 1)) · Pᵢ,  1 ≤ i ≤ k.')
paragraph('Повышение степени сохраняет значение полинома в каждой точке параметра. Например, в кубическом примере введено восемь точек. Первые два сегмента имеют общий стык в P₃, а отрезок P₆P₇ представляется кубическим сегментом с двумя дополнительными коллинеарными вершинами. Все восемь исходных точек участвуют в построении.')
heading('2.2 Геометрическое разбиение и шаги де Кастельжо', 2)
paragraph('Для выбранного сегмента первое множество Q⁽⁰⁾ равно его управляющим вершинам. При заданном t от 0 до 1 следующий уровень строится линейной интерполяцией соседних точек.')
formula('Qᵢ⁽ʳ⁾(t) = (1 − t) Qᵢ⁽ʳ⁻¹⁾(t) + t Qᵢ₊₁⁽ʳ⁻¹⁾(t).')
paragraph('На уровне r остаётся d + 1 − r точек. Последняя вершина Q₀⁽ᵈ⁾(t) равна B(t). Левые концы уровней сверху вниз формируют контрольную ломаную участка [0, t]. Правые концы в обратном порядке формируют ломаную участка [t, 1]. Обе ломаные имеют общую вершину B(t). При t = 0 или t = 1 одна из них стягивается в конец сегмента.')
paragraph('Сплошную кривую на экране аппроксимирует ломаная из отсчётов B(k/N) для k = 0, …, N. Общий конец соседних сегментов записывается один раз. Для M сегментов получается MN + 1 отсчётов. Поле N меняет точность отрисовки, а t задаёт место точного разбиения и значения промежуточных вершин на выбранном сегменте.')

heading('2.3 Поверхность Безье и её контрольные сетки', 2)
paragraph('Строка j массива соответствует направлению v, а столбец i — направлению u. Пусть в строке n + 1 точка, а строк m + 1. При u и v от 0 до 1 точка поверхности есть тензорная комбинация контрольных вершин.')
formula('S(u, v) = Σᵢ Σⱼ Pᵢⱼ · bᵢ,ₙ(u) · bⱼ,ₘ(v).')
formula('bᵢ,ₙ(t) = C(n, i) · tⁱ · (1 − t)ⁿ⁻ⁱ.')
paragraph('Суммы берутся по i от 0 до n и по j от 0 до m. Вес по каждому направлению вычисляется заранее для всех значений равномерной сетки. При Nᵤ и Nᵥ интервалах берутся точки S(i/Nᵤ, j/Nᵥ). Каждая ячейка визуализируется двумя треугольниками. Для изображения применяется ортографическая проекция с поворотом камеры, глубина треугольников сравнивается для каждого пикселя.')
paragraph('Для уточнения контрольной сетки каждую строку разбиваем методом де Кастельжо в u. Получаем две сетки. Затем разбиваем каждый столбец обеих сеток в v. Четыре результата управляют участками [0, u] × [0, v], [u, 1] × [0, v], [0, u] × [v, 1] и [u, 1] × [v, 1]. В точных вычислениях их границы совпадают, а общая угловая вершина равна S(u, v). Приложение показывает эту точку маркером и выводит её координаты.')
paragraph('В пользовательском вводе допускается от 2 до 128 вершин кривой, от 2 до 16 строк и столбцов поверхности, от 1 до 64 интервалов по каждому направлению и координаты от −10000 до 10000. При изменении размеров сетки общая часть сохраняется. Координаты каждой вершины редактируются отдельно.')

for path, caption in [
    ('algorithm-main', 'Рисунок 1 — Общая схема работы приложения'),
    ('algorithm-curve', 'Рисунок 2 — Построение и уточнение сегментов кривой'),
    ('algorithm-surface', 'Рисунок 3 — Расчёт и уточнение поверхности Безье'),
]:
    page_break()
    figure(f'report/{path}.png', caption)

heading('3 Листинг программы', new_page=True)
paragraph('Приведены файл модуля и все файлы на Go, написанные для лабораторной работы, включая проверки. Лицензии сторонних библиотек и шрифта включены в комплект.')
sources = ['go.mod'] + [
    path.relative_to(ROOT).as_posix()
    for base in ('cmd', 'internal')
    for path in sorted((ROOT / base).rglob('*.go'))
]
for index, path in enumerate(sources, 1):
    code(path, index)

heading('4 Пример работы программы', new_page=True)
heading('4.1 Квадратичная кривая', 2)
paragraph('Исходные контрольные точки примера (−5, −2), (−4, 4), (−1, 1), (1, −4), (5, 1). При t = 0.4 на первом уровне первого сегмента получены (−4.6, 0.4) и (−2.8, 2.8). Второй уровень даёт B(0.4) = (−3.88, 1.36). Два сегмента по десять интервалов дают 21 отсчёт.')
figure('report/interface-quadratic.png', 'Рисунок 4 — Первый этап построения квадратичной кривой')
heading('4.2 Кубическая кривая', 2, new_page=True)
paragraph('Для восьми исходных вершин получены три сегмента. При t = 0.5 у первого сегмента вычислено B(0.5) = (−2.875, 1.875). Третий сегмент соединяет (4, 1) и (5.5, 2.4). Его степень повышена с первой до третьей. При 16 интервалах на сегмент получено 49 отсчётов.')
figure('report/interface-cubic.png', 'Рисунок 5 — Кубическая кривая на третьем этапе построения')

heading('4.3 Поверхность и переключение контрольных сеток', 2, new_page=True)
paragraph('В примере использована сетка из трёх строк и четырёх столбцов. Координаты X столбцов определены выражением −4 + 8i/3, координаты Y строк равны −3, 0 и 3. Значения Z приведены в таблице.')
table(['Строка j', 'Z₀ⱼ', 'Z₁ⱼ', 'Z₂ⱼ', 'Z₃ⱼ'], [
    [j] + [point['z'] for point in row]
    for j, row in enumerate(metrics['surface']['control_rows'])
])
paragraph('Степень поверхности равна (3, 2). При u = 0.42 и v = 0.60 получена точка S = (−0.64, 0.6, 1.870919968). Для Nᵤ = 16 и Nᵥ = 12 вычислены 221 точка, 192 ячейки и 384 треугольника.')
figure('report/interface-surface.png', 'Рисунок 6 — Поверхность с исходными и уточнёнными вершинами')
page_break()
figure('report/interface-surface-original.png', 'Рисунок 7 — Тот же вид с исходной контрольной сеткой')
paragraph('На рисунке 6 исходные вершины показаны тёмно-синим, а уточнённые — оранжевым. На рисунке 7 отключён слой уточнённых вершин. При переключении слоя значения S(u, v) и количество отсчётов сохраняются.')

heading('4.4 Проверка и запуск', 2)
table(['Режим', 'Вершины', 'Разбиений', 'Отсчётов'], [
    ['Квадратичная'] + [len(metrics['curves'][0]['control']), metrics['curves'][0]['divisions_per_segment'], metrics['curves'][0]['samples']],
    ['Кубическая'] + [len(metrics['curves'][1]['control']), metrics['curves'][1]['divisions_per_segment'], metrics['curves'][1]['samples']],
    ['Поверхность', '4 × 3', '16 × 12', metrics['surface']['sample_count']],
])
paragraph('Для запуска из папки работы на Linux x86_64 используется файл bin/lab10-linux-amd64. Для ARM64 предусмотрен bin/lab10-linux-arm64. В оконном тесте готовый AMD64-файл запущен с путём поиска программ /nonexistent. Тест вводит координаты и параметры через X11, строит кривую из шести точек, меняет размеры поверхности на 5 × 4 и сохраняет PNG. Оба сохранённых изображения совпали с результатами прямого вызова вычислительного ядра и отрисовки.')
paragraph('Проверка формул сравнивает значения поверхности с независимым вычислением методом де Кастельжо в двух направлениях. Проверено совпадение уточнённых участков с исходной поверхностью при переводе локальных параметров в общие. Для повторения трёх примеров без экрана предусмотрен режим demo. Файлы PNG и числовые расчёты хранятся в папке images.')

heading('5 Ответы на контрольные вопросы', new_page=True)
answers = [
    (
        '1. Выполните геометрическое построение квадратичной кривой Безье.',
        'Соединим P₀ = (−5, −2), P₁ = (−4, 4) и P₂ = (−1, 1). При t = 0.4 отметим точки (−4.6, 0.4) на P₀P₁ и (−2.8, 2.8) на P₁P₂. Разделим соединяющий их отрезок в той же доле и получим (−3.88, 1.36). Изменяя t от 0 до 1, получаем точки всей дуги. Промежуточная ломаная показана на рисунке 4.',
    ),
    (
        '2. Запишите аналитическое выражение и сформулируйте свойства кривой Безье второй степени.',
        'B(t) = (1 − t)²P₀ + 2t(1 − t)P₁ + t²P₂. При t = 0 и t = 1 получаются P₀ и P₂. Производные на концах равны 2(P₁ − P₀) и 2(P₂ − P₁). Для неколлинеарной тройки кривая является дугой параболы внутри контрольного треугольника.',
    ),
    (
        '3. Выполните геометрическое построение кубической кривой Безье.',
        'Возьмём первые четыре точки кубического примера (−5, −2), (−4, 3), (−2, 3), (0, −1). При t = 0.5 первый ряд середин равен (−4.5, 0.5), (−3, 3), (−1, 1). Второй ряд даёт (−3.75, 1.75) и (−2, 2). Середина последнего отрезка (−2.875, 1.875) лежит на кривой. Аналогичные три шага выполняются для каждого значения t.',
    ),
    (
        '4. Запишите аналитическое выражение и сформулируйте свойства кривой Безье n-й степени.',
        'B(t) = Σᵢ Pᵢ bᵢ,ₙ(t), где i = 0, …, n, а bᵢ,ₙ(t) = C(n, i)tⁱ(1 − t)ⁿ⁻ⁱ. По биному Ньютона сумма весов равна (t + (1 − t))ⁿ = 1. Для t от 0 до 1 веса неотрицательны, поэтому точка B(t) принадлежит выпуклой оболочке контрольных вершин.',
    ),
    (
        '5. Выполните построение кривой методом Чайкина.',
        'Для каждого ребра берём точки на расстояниях 1/4 и 3/4 его длины от начала. Для открытой ломаной (0, 0), (4, 0), (4, 4) получаем (1, 0), (3, 0), (4, 1), (4, 3). Если закрепить исходные концы, добавим по краям (0, 0) и (4, 4). При повторении шага ребро (3, 0)–(4, 1) даёт (3.25, 0.25) и (3.75, 0.75). Соединение новых точек по порядку отсекает исходный угол.',
    ),
    (
        '6. Запишите выражение для точек на поверхности Безье n-й степени.',
        'Для степеней (n, m) имеем S(u, v) = Σᵢ Σⱼ Pᵢⱼ bᵢ,ₙ(u)bⱼ,ₘ(v). Если степени равны, подставляем m = n. При v = 0 сохраняется только член с j = 0. Поэтому S(u, 0) есть кривая Безье по первой строке сетки. При u = 1 и v = 1 сохраняется только Pₙ,ₘ.',
    ),
    (
        '7. Сформулируйте свойства кривой Безье n-й степени.',
        'Крайние контрольные точки совпадают с концами кривой. Производные B′(0) = n(P₁ − P₀) и B′(1) = n(Pₙ − Pₙ₋₁) задают касательные, если соответствующие векторы ненулевые. Аффинное преобразование контрольных точек даёт то же аффинное преобразование кривой, поскольку сумма весов равна единице. Обращение порядка вершин заменяет параметр t на 1 − t.',
    ),
    (
        '8. Выполните поэтапное построение поверхности методами Ду-Сабина и Кэтмулла-Кларка.',
        'Ду–Сабин. Возьмём единичный куб. Для каждой четырёхугольной грани усредним исходную вершину, середины её двух рёбер и центр грани. Например, на грани z = 0 для вершины (0, 0, 0) получаем (0.25, 0.25, 0). Новые точки каждой грани соединяем в четырёхугольник. Для каждого старого ребра соединяем четыре соответствующие точки двух соседних граней. Вокруг каждой старой вершины куба соединяем три точки в треугольник. После шага получаются 6 граней исходного типа, 12 четырёхугольников рёбер и 8 треугольников вершин.',
    ),
]
for question, answer in answers:
    heading(question, 2)
    paragraph(answer)
paragraph('Кэтмулл–Кларк. На том же кубе вычислим центры граней. Для ребра от начала координат до (1, 0, 0) усредним его концы и центры двух смежных граней. Получим Eₓ = (0.5, 0.125, 0.125). Для старой вершины S средней точкой граней является Q, а средней точкой середин рёбер — R. При валентности n новая вершина равна (Q + 2R + (n − 3)S)/n. Для S = (0, 0, 0) при n = 3 получаем S′ = (2/9, 2/9, 2/9). На нижней грани у этой вершины формируем четырёхугольник [S′, Eₓ, (0.5, 0.5, 0), Eᵧ], где Eᵧ = (0.125, 0.5, 0.125). Повторяем для каждой пары грани и её угла. Получаем 24 четырёхугольника.')

heading('Вывод', new_page=True)
paragraph(CONCLUSION)
paragraph('Метод де Кастельжо позволил получить координаты промежуточных вершин и разложить сегмент на два участка, сохранив его геометрию. Для поверхности аналогичное разбиение по двум параметрам дало четыре контрольные сетки. Увеличение количества интервалов увеличивало число отсчётов, а переключение слоёв помогало сопоставить форму с исходными и уточнёнными вершинами. Работа приложения проверена в графическом сеансе Linux без Go в пути поиска программ.')

doc.save(OUT / 'report.docx')
(OUT / 'report.md').write_text('\n'.join(md), encoding='utf-8')
manifest = {
    'goal': GOAL,
    'conclusion': CONCLUSION,
    'full_source_files': sources,
    'figures': len(doc.inline_shapes),
    'questions': [question for question, _ in answers],
    'margins_mm': [30, 15, 20, 20],
}
(OUT / 'report-manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n')
