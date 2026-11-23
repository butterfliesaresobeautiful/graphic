"""Check the delivered DOCX/PDF against the actual app sources and report manifest."""
from pathlib import Path
import json,re
import fitz
from docx import Document

root=Path(__file__).resolve().parents[1]
m=json.loads((root/'report/report-manifest.json').read_text())
d=Document(root/'report/report.docx')
pdf=fitz.open(root/'report/report.pdf')
norm=lambda s:re.sub(r'\s+','',s)
full='\n'.join(p.text for p in d.paragraphs)
assert m['goal'] in full and m['conclusion'] in full
for q in m['questions']: assert q in full,q
expected=''.join((root/f).read_text() for f in m['full_source_files'])
listing=''.join(p.text for p in d.paragraphs if p.style.name=='Listing')
assert norm(expected)==norm(listing),'DOCX listing differs from source'
assert len(d.inline_shapes)==m['figures']
s=d.sections[0]
assert abs(s.page_width.mm-210)<.1 and abs(s.page_height.mm-297)<.1
assert all(abs(a-b)<.1 for a,b in zip([s.left_margin.mm,s.right_margin.mm,s.top_margin.mm,s.bottom_margin.mm],m['margins_mm']))
assert d.styles['Normal'].font.name=='Times New Roman'
assert d.styles['Normal'].font.size.pt==14
assert d.styles['Normal'].paragraph_format.line_spacing==1.5
text=''.join(p.get_text() for p in pdf)
assert norm(m['goal']) in norm(text) and norm(m['conclusion']) in norm(text)
for q in m['questions']: assert norm(q) in norm(text),q
pdfcode=[]; outside=[]; fonts=set(); image_count=0
for n,p in enumerate(pdf,1):
    assert p.get_text().strip().strip('0123456789') or p.get_image_info(), f'Blank page {n}'
    assert abs(p.rect.width-595.3)<1 and abs(p.rect.height-841.9)<1
    image_count+=len(p.get_image_info())
    for block in p.get_text('dict')['blocks']:
        if 'lines' not in block: continue
        for line in block['lines']:
            for span in line['spans']:
                fonts.add(span['font'])
                if 'DejaVuSansMono' in span['font']: pdfcode.append(span['text'])
    for block in p.get_text('rawdict')['blocks']:
        if 'lines' not in block: continue
        for line in block['lines']:
            for span in line['spans']:
                for ch in span['chars']:
                    if not ch['c'].strip(): continue
                    x0,y0,x1,y1=ch['bbox']
                    if y0>=790: # footer
                        assert ch['c'].isdigit() and y1<=825
                    elif x0<84 or x1>555 or y0<54 or y1>787:
                        outside.append({'page':n,'char':ch['c'],'bbox':ch['bbox']})
    for im in p.get_image_info():
        x0,y0,x1,y1=im['bbox']
        if x0<84 or x1>555 or y0<54 or y1>787:
            outside.append({'page':n,'image_bbox':im['bbox']})
assert norm(expected)==norm(''.join(pdfcode)),'PDF code missing or changed'
assert not outside,outside[:10]
assert image_count==m['figures'],image_count
assert any('TimesNewRoman' in f for f in fonts),fonts
report={'pages':len(pdf),'figures':image_count,'questions':len(m['questions']),'source_files':m['full_source_files'],'full_code_matches_docx':True,'full_code_matches_pdf_ignoring_whitespace':True,'goal_and_conclusion_exact':True,'body_font':'Times New Roman','font_size_pt':14,'fonts_in_pdf':sorted(fonts),'margins_mm':m['margins_mm'],'out_of_margin_content':outside,'result':'PASS'}
(root/'validation/report-check.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps(report,ensure_ascii=False,indent=2))
