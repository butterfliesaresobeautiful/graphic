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
doc.core_properties.title = "Лабораторная работа № 8. Интерполяция изображений"
doc.core_properties.subject = "Интерполяция изображений"
doc.core_properties.author = "ФИО студента: __________________"
doc.core_properties.keywords = "Go, интерполяция, аффинные преобразования, Лагранж"


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

md.append('# Титульный лист\n')
for text in ['Минобрнауки России','Юго-Западный государственный университет','Кафедра программной инженерии']:
    p=paragraph(text,WD_ALIGN_PARAGRAPH.CENTER);p.paragraph_format.first_line_indent=Mm(0)
for _ in range(4):doc.add_paragraph()
for text in ['ОТЧЁТ','по лабораторной работе № 8','по дисциплине «Компьютерная графика»','«Интерполяция изображений»','Индивидуальный номер 28. Общее задание']:
    p=paragraph(text,WD_ALIGN_PARAGRAPH.CENTER,text=='ОТЧЁТ');p.paragraph_format.first_line_indent=Mm(0)
for _ in range(3):doc.add_paragraph()
for text in ['Выполнил: студент группы __________','ФИО ____________________________','Проверил: _______________________']:
    p=paragraph(text,WD_ALIGN_PARAGRAPH.RIGHT);p.paragraph_format.first_line_indent=Mm(0)
for _ in range(3):doc.add_paragraph()
p=paragraph('Курск 2026 г.',WD_ALIGN_PARAGRAPH.CENTER);p.paragraph_format.first_line_indent=Mm(0)

GOAL='выполнить аффинные преобразования масштабирования, поворота и скоса с разными параметрами, применяя разные методы интерполяции.'
CONCLUSION='В ходе выполненной работы, мы выполнили аффинные преобразования масштабирования, поворота и скоса с разными параметрами, применяя разные методы интерполяции.'
names=['identity','scale-up','scale-down','rotate-positive','rotate-negative','shear-positive','shear-negative','combined']
labels=['Без преобразования','Увеличение','Уменьшение','Поворот +28°','Поворот −17°','Скос +','Скос −','Составное']
metrics={name:json.loads((ROOT/'images'/name/'metrics.json').read_text()) for name in names}
heading('1 Задание',new_page=True)
heading('Цель работы',2);paragraph(GOAL)
paragraph('Источник задания — приложенный документ «ЛР8.docx», тема «Интерполяция изображений». Отдельной формулировки цели в документе нет; приведённая выше цель воспроизводит его предложение «Задание выполнить аффинные преобразования масштабирования, поворота и скоса с разными параметрами, применяя разные методы интерполяции» без вводного слова «Задание». Следующее требование документа — «Сделать выводы».')
paragraph('Индивидуальный номер студента — 28. В документе отсутствуют таблица вариантов и максимальный номер, поэтому выражение 28 % максимальный номер вычислить нельзя. По уточнению пользователя выполняется общее задание документа. Наборы числовых параметров выбраны самостоятельно для сравнения методов; угол 28° является одним из параметров опыта, а не номером варианта.')
paragraph('Требуется реализовать интерполяцию по ближайшему соседу, билинейную и бикубическую интерполяцию, выполнить масштабирование, поворот и скос с несколькими параметрами и сравнить полученные изображения. Программа должна работать как настольное Linux-приложение без установленного Go. Все три результата вычисляются для одной выходной сетки и одной матрицы преобразования.')
paragraph('Оформление отчёта соответствует лабораторным № 4 и № 6 этого репозитория: титульный лист, задание, схемы алгоритмов, полный собственный листинг, примеры работы, анализ и вывод; A4, Times New Roman 14, интервал 1,5, абзац 12,5 мм, поля слева 30 мм, справа 15 мм, сверху и снизу 20 мм. Листинг — DejaVu Sans Mono 9. Контрольные вопросы в ЛР8.docx отсутствуют, поэтому отдельные вопросы не приписываются заданию.')
heading('2 Схемы алгоритмов',new_page=True)
heading('2.1 Геометрия преобразований',2)
paragraph('Начало координат растра находится слева сверху, X направлена вправо, Y — вниз. Геометрические границы источника ширины W и высоты H образуют прямоугольник [0,W] × [0,H]; центр пикселя с индексами (i,j) имеет координаты (i+0,5; j+0,5). Преобразование выполняется относительно центра изображения (W/2; H/2).')
formula('S = [[sx, 0], [0, sy]],   H = [[1, hx], [hy, 1]],')
formula('R = [[cos θ, −sin θ], [sin θ, cos θ]],   A = R · H · S.')
paragraph('Сначала применяется масштабирование, затем одновременный двухкоординатный скос, затем поворот. Это один проход по изображению: промежуточные растры не создаются и не интерполируются повторно. Положительный угол соответствует повороту по часовой стрелке в экранной системе координат. Матрица H задаёт x′ = x + hx·y, y′ = hy·x + y; это одновременный скос, который отличается от последовательного применения двух одномерных скосов.')
formula('A11 = sx(cos θ − hy sin θ),   A12 = sy(hx cos θ − sin θ),')
formula('A21 = sx(sin θ + hy cos θ),   A22 = sy(hx sin θ + cos θ).')
formula('det A = sx · sy · (1 − hx · hy).')
paragraph('Преобразуются четыре угла прямоугольника относительно его центра. Минимальные координаты дают начало выходной области (xmin,ymin), а размеры холста равны округлённым вверх размахам координат. Результат полностью помещается на холсте; лишние области заполняются белым. Для подавления численной погрешности точных поворотов на 90° при округлении размеров вычитается 10^(-9).')
paragraph('Для каждого выходного пикселя вычисляется обратное отображение. Индексы исходной сетки, пригодные для интерполяции, получаются по формуле:')
formula('[x, y]^T = A^(-1) · [u+0,5+xmin, v+0,5+ymin]^T')
formula('+ [W/2−0,5, H/2−0,5]^T.')
paragraph('Прямой перенос мог бы оставлять незаполненные пиксели; обратное отображение задаёт значение каждому выходному элементу. Если точка не принадлежит области −0,5 ≤ x < W−0,5, −0,5 ≤ y < H−0,5, сохраняется белый фон. Общая маска покрытия используется всеми методами и при вычислении различий.')
heading('2.2 Интерполяция по ближайшему соседу',2)
formula('g(x,y) = f(floor(x+0,5), floor(y+0,5)).')
paragraph('Берётся один ближайший пиксель. При равных расстояниях выбирается больший индекс. Опорные индексы ограничиваются границами источника. Метод не создаёт промежуточных цветов, но при увеличении показывает ступеньки и крупные блоки; при уменьшении может пропускать тонкие детали.')
heading('2.3 Билинейная интерполяция',2)
formula('l = floor(x), k = floor(y), a = x−l, b = y−k.')
formula('g = (1−a)(1−b)f(l,k) + a(1−b)f(l+1,k)')
formula('+ (1−a)b f(l,k+1) + ab f(l+1,k+1).')
paragraph('Четыре неотрицательных веса в сумме равны единице. Каждый канал RGB интерполируется независимо. На резких переходах появляются промежуточные значения, уменьшается ступенчатость, но края становятся мягче. Внутри области билинейная интерполяция точно воспроизводит функции, линейные по каждой координате.')
heading('2.4 Бикубическая интерполяция и уточнение формул',2)
paragraph('Документ задаёт сетку 4×4 и полиномиальные коэффициенты. Используется согласованная тензорная форма кубической интерполяции Лагранжа на узлах −1, 0, 1, 2. Это полином степени не выше трёх по каждой координате, то есть суммарной степени не выше шести. Формулировка «полином шестой степени» в документе не означает одномерную интерполяцию шестого порядка.')
formula('w(−1,t) = −t(t−1)(t−2)/6,')
formula('w(0,t) = (t+1)(t−1)(t−2)/2,')
formula('w(1,t) = −(t+1)t(t−2)/2,')
formula('w(2,t) = (t+1)t(t−1)/6,')
formula('g(x,y) = Σ Σ w(i,a) w(j,b) f(l+i,k+j), i,j = −1,0,1,2.')
paragraph('В напечатанном перечне коэффициентов есть несогласованности. Например, при a=0 и b=0,5 коэффициент пикселя f(l,k+1) должен равняться 9/16, тогда как указанный c2 содержит множитель a и обращается в нуль. В c14 встречается xy вместо произведения дробных координат ab. Буквальное копирование такого перечня не даёт согласованной интерполяции. Тензорная запись устраняет перестановки координат и знаковые опечатки; исходный документ приложен без изменений.')
paragraph('Для t=0,5 веса равны (−1/16, 9/16, 9/16, −1/16). Их сумма равна единице, однако крайние веса отрицательны. Поэтому около контрастных границ возможны значения ниже 0 и выше 255. Они учитываются отдельным счётчиком и перед записью ограничиваются диапазоном 0…255 с округлением до ближайшего целого. Это не алгоритм Keys/Catmull–Rom, а кубический полином Лагранжа, согласующийся с полиномиальной постановкой документа.')
figure('report/weights.png','Рисунок 1 — Кубические веса Лагранжа и отрицательные боковые доли',max_h=85)
heading('2.5 Границы, сравнение и ограничения',2)
paragraph('Для точки внутри геометрической области каждый опорный индекс за границей растра заменяется ближайшим допустимым индексом: используется повтор крайнего пикселя. Это правило одинаково для всех методов. Белый фон вне преобразованного изображения не подмешивается в интерполяцию у границы.')
paragraph('Вход PNG/JPEG/GIF нормализуется в непрозрачный 8-битный RGB; прозрачность накладывается на белый фон, для GIF читается первый кадр. Интерполяция проводится непосредственно по кодовым значениям RGB, без линеаризации гаммы, цветовых профилей и предварительного низкочастотного фильтра. Эти ограничения важны: при уменьшении ни один из трёх методов не гарантирует устранение алиасинга.')
paragraph('Различия результатов измеряются только внутри общей маски покрытия. Для N пикселей и трёх каналов: MAE = Σ|I1−I2|/(3N), RMSE = √(Σ(I1−I2)²/(3N)). Дополнительно вычисляются максимальная разность канала и число различающихся пикселей. Это расстояния между методами, а не оценка качества относительно известного непрерывного эталона.')
paragraph('Масштабы ограничены диапазоном 0,05…8, угол — −360…360°, скосы — −3…3. При |1−hx·hy| < 0,01 преобразование отвергается как вырожденное или почти вырожденное. Стороны входа и выхода не превышают 4096, площадь — 16 миллионов пикселей, размер входного файла — 32 МиБ. Проверяется отсутствие пустого покрытия после сильного уменьшения.')
paragraph('Вычислительная сложность всех методов линейна по числу выходных пикселей. На один покрытый пиксель требуется 1, 4 или 16 чтений опорных цветов соответственно. Изображения результатов требуют O(Wвых·Hвых) памяти. Сохранение защищает исходный файл от перезаписи, в том числе через символические и жёсткие ссылки. PNG записывается атомарно; набор файлов сохраняется последовательно.')
page_break();figure('report/algorithm-main.png','Рисунок 2 — Общая схема аффинного преобразования и сравнения',max_h=228)
page_break();figure('report/algorithm-sampling.png','Рисунок 3 — Обработка одного выходного пикселя тремя методами',max_h=228)
heading('3 Листинг программы',new_page=True)
paragraph('Полностью приведены собственные исходники Go и тесты. Аффинное преобразование и интерполяция реализованы в internal/imaging на стандартной библиотеке. X11, диалог и PNG адаптированы из предыдущих лабораторных. Сторонние библиотеки обслуживают только окно, шрифт и превью; их код и лицензии находятся в vendor и third_party.')
source_files=['go.mod','cmd/lab8/main.go','internal/imaging/affine.go','internal/imaging/interpolate.go','internal/imaging/transform.go','internal/imaging/demo.go','internal/imaging/io.go','internal/desktop/window.go','internal/desktop/ui.go','internal/imaging/imaging_test.go','cmd/lab8/main_test.go','internal/desktop/desktop_test.go']
for i,p in enumerate(source_files,1):code(p,i)
heading('4 Пример работы программы',new_page=True)
heading('4.1 Исходное изображение и параметры опытов',2)
paragraph('Испытательное изображение 480×360 создаётся самой программой на Go. Оно содержит плавные градиенты, шахматную сетку, наклонные тонкие линии, цветные частые полосы, окружности с резкими границами и тёмную рамку. Такой источник позволяет наблюдать поведение методов на гладких и высокочастотных участках. Это синтетический рисунок, а не фотография.')
figure('images/combined/source.png','Рисунок 4 — Исходный испытательный рисунок 480×360',max_h=108)
paragraph('Все опыты воспроизводятся командой bin/lab8-linux-amd64 -mode suite -out images. Для каждого набора сохранены исходник, три PNG и metrics.json. Параметры приведены в таблице 1; порядок преобразований во всех строках одинаков: масштаб → скос → поворот.')
page_break()
paragraph('Таблица 1 — Наборы параметров',WD_ALIGN_PARAGRAPH.CENTER)
rows=[]
for name,label in zip(names,labels):
 c=metrics[name]['Config'];g=metrics[name]['Geometry'];rows.append([label,f"{c['ScaleX']}; {c['ScaleY']}",str(c['Angle']),f"{c['ShearX']}; {c['ShearY']}",f"{g['Width']}×{g['Height']}"])
table(['Опыт','sx; sy','θ, °','hx; hy','Размер'],rows)
paragraph('Составное преобразование имеет det A = 1,153125, выходной размер 639×553 и 199256 покрытых пикселей. Геометрическая площадь преобразованного прямоугольника составляет 199260; небольшое отличие числа покрытых центров пикселей связано с дискретизацией границы, а не с обрезкой холста.')
page_break();figure('report/comparison-scale.png','Рисунок 5 — Масштабирование: увеличение и уменьшение, три метода',max_h=180)
paragraph('При увеличении ближайший сосед делает видимыми блоки исходных пикселей. Билинейная интерполяция смягчает границы; бикубическая сохраняет более выраженный локальный контраст, но может создавать ореолы. При уменьшении шахматная сетка и частые полосы дают различающиеся узоры: без предварительного фильтра подавление наложения частот не гарантируется.')
page_break();figure('report/comparison-rotation.png','Рисунок 6 — Повороты +28° и −17°, одинаковая геометрия для трёх методов',max_h=180)
paragraph('Холст автоматически расширяется до полного охвата повернутого прямоугольника. Белые углы лежат вне источника и исключены из численного сравнения. Различия ступенчатости наклонных границ удобнее рассматривать на увеличенных фрагментах, поскольку при показе целого растра в отчёте тонкие детали уменьшаются.')
page_break();figure('report/comparison-shear.png','Рисунок 7 — Два набора коэффициентов скоса',max_h=180)
paragraph('При скосе изменяются направления линий и форма исходной рамки. Для hx=0,45, hy=0,15 определитель H равен 0,9325; для hx=−0,35, hy=0,10 — 1,035. Следовательно, одновременный двухкоординатный скос в этих опытах изменяет также площадь изображения.')
page_break();figure('report/comparison-combined.png','Рисунок 8 — Однопроходное составное преобразование',max_h=115)
figure('report/comparison-zoom.png','Рисунок 9 — Одинаковые фрагменты результатов, увеличение без сглаживания',max_h=105)
page_break();figure('report/profile.png','Рисунок 10 — Профиль красного канала по сохранённым PNG',max_h=90)
paragraph('На рисунке 10 показан красный канал сохранённых PNG: строка v=84, столбцы 30…85. Видны ступенчатые переходы NN, промежуточные уровни BL и выбросы BC. Значения ограничены диапазоном 0…255. Профиль получен из результатов программы, а не из отдельной модели.')
heading('4.2 Численные различия',2)
paragraph('Таблица 2 — Различия каналов внутри общей маски покрытия',WD_ALIGN_PARAGRAPH.CENTER)
rows=[]
for name,label in zip(names,labels):
 m=metrics[name];d=m['Differences'];rows.append([label,f"{d[0]['MAE']:.4f}",f"{d[1]['MAE']:.4f}",f"{d[2]['RMSE']:.4f}",str(m['ClippedChannels'][2])])
table(['Опыт','MAE: NN–BL','MAE: NN–BC','RMSE: BL–BC','Ограничено каналов BC'],rows)
paragraph('NN — ближайший сосед, BL — билинейная интерполяция, BC — бикубическая. При тождественном преобразовании все результаты совпали с источником. Для составного опыта MAE NN–BL = 6,4572, MAE NN–BC = 7,5041, RMSE BL–BC = 4,6049.')
paragraph('Ограничены 5470 каналов BC в составном опыте и 11919 при увеличении; один пиксель может дать три срабатывания. У NN и BL ограничений нет. Эти различия не определяют победителя: ни один метод не является эталоном непрерывной сцены.')
heading('4.3 Настольное приложение',2,new_page=True)
figure('report/interface.png','Рисунок 11 — Реальное окно Linux-приложения',max_h=105)
paragraph('Кнопка «Открыть файл…» показывает файловый диалог; «Демо» устанавливает испытательный рисунок. Поля Sx, Sy, угол, hx и hy задают параметры. Кнопка набора последовательно выбирает восемь опытов, а «Преобразовать» применяет текущие поля. На экране одновременно показаны источник и три результата. При ошибке ввода сохраняется предыдущий корректный результат; загрузка нового источника сбрасывает старые результаты.')
paragraph('«Сохранить всё» записывает источник, три PNG и JSON в выбранную папку. Не применённые изменения полей не меняют сохраняемые параметры: сохраняется последний завершённый расчёт. Кнопка «Пиксели» переключает вид на центральные фрагменты 96×72 с увеличением без сглаживания. Для трёх результатов используются одни и те же координаты фрагмента; центральный фрагмент источника служит отдельной иллюстрацией и не является обратным отображением выходного фрагмента.')
page_break();figure('report/interface-zoom.png','Рисунок 12 — Увеличенные пиксели трёх методов в настоящем окне',max_h=95)
paragraph('Снимки получены чтением пикселей окна через X11 GetImage. Интеграционный тест запустил готовый бинарник с PATH=/nonexistent, ввёл масштаб 1,25 вместо 0,9, открыл файл через диалог, нажал преобразование и сохранение. Пиксели каждого из трёх сохранённых результатов совпали с вычислительным ядром. Снимки не являются макетом интерфейса.')
heading('4.4 Проверки и воспроизведение',2)
paragraph('Пройдены 12 тестов: веса Лагранжа и воспроизведение полиномов, аналитические RGB, тождественное преобразование, поворот на 90°, матрицы и обратное отображение, границы и выбросы, маска сравнения, прозрачность и PNG, защита источника, ошибочные параметры, CLI и настоящее окно. go vet завершился без замечаний.')
paragraph('AMD64 без DISPLAY и Go в PATH воспроизвёл все PNG и JSON побайтово; дочерних программ и сетевых соединений нет. Оба Linux ELF статические, шрифт встроен. Для окна нужен X11/XWayland. ARM64 собран, но на ARM не исполнялся. ZIP проверен после распаковки: контрольные суммы и все восемь опытов.')
heading('5 Анализ результатов',new_page=True)
paragraph('Три метода решают одну задачу восстановления значения между узлами, но имеют разные свойства. Ближайший сосед сохраняет исходные уровни и подходит для категориальных изображений, где смешение меток недопустимо; для фотографий при увеличении заметна пикселизация. Билинейный метод использует выпуклую комбинацию четырёх значений и даёт мягкие переходы без выбросов. Кубический Лагранж использует 16 значений и отрицательные веса, повышая локальную резкость ценой возможных ореолов и ограничения значений.')
paragraph('Опыты с положительными и отрицательными поворотами и скосами подтверждают, что разница связана с интерполяцией, а не с разными размерами холста или направлением преобразования: геометрия и маска покрытия общие. Тождественный опыт и точный поворот на 90° служат проверками правильности координат. Однопроходная композиция уменьшает число повторных интерполяций по сравнению с последовательным преобразованием уже округлённых растров.')
paragraph('При уменьшении наблюдается проблема дискретизации высоких частот. Для качественного уменьшения в практических системах нужен предварительный низкочастотный фильтр с полосой, согласованной с масштабом и направлением преобразования, либо интегрирование по площади пикселя. В этой работе дополнительный фильтр намеренно не включён, чтобы сравнивать именно три метода из задания на одинаковых условиях. Для фотометрически корректной обработки также может потребоваться переход из кодового RGB в линейное световое пространство.')
heading('Вывод',new_page=True)
paragraph(CONCLUSION)
paragraph('Создано десктопное приложение, которое выполняет все три преобразования и сравнивает ближайшего соседа, билинейный и бикубический методы. На восьми наборах параметров установлены различия ступенчатости, сглаживания и выбросов около контрастных границ. Подтверждены точность базовых координатных случаев, воспроизведение кубических полиномов и автономный запуск Linux-бинарника. Универсальный «лучший» метод по разности изображений не определяется: выбор зависит от содержания изображения, требуемой резкости и допустимых артефактов.')
heading('Источники')
paragraph('1. ЛР8.docx. Интерполяция изображений: документ задания, приложенный пользователем. Оригинал включён в reference/ЛР8.docx без изменений.')
paragraph('2. Отчёты лабораторных № 4 и № 6 репозитория graphiclabs — образцы оформления, структуры титульного листа, листинга, таблиц и иллюстраций.')
paragraph('3. Документация стандартной библиотеки Go: image, image/color, image/png, math, testing. https://pkg.go.dev/std')
paragraph('4. X Window System. X11 Protocol. https://www.x.org/releases/current/doc/xproto/x11protocol.html')
manifest={'lab':8,'individual':28,'maximum_variant':None,'variant':None,'variant_note':'Общее задание подтверждено пользователем; таблица вариантов отсутствует','goal':GOAL,'goal_source':'Предложение задания без вводного слова; отдельной цели нет','conclusion':CONCLUSION,'questions':[],'full_source_files':source_files,'figures':12,'body_font':'Times New Roman','body_size_pt':14,'line_spacing':1.5,'margins_mm':[30,15,20,20]}
doc.save(OUT/'report.docx')
(OUT/'report.md').write_text('\n'.join(md),encoding='utf-8')
(OUT/'report-manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
print('Created report.docx, report.md and manifest')
