"""Compare the report's formatting with the delivered lab 4 and lab 6 reports."""
from pathlib import Path
from docx import Document
import json
ROOT=Path(__file__).resolve().parents[1]
def properties(path):
 d=Document(path);s=d.sections[0]
 out={'page_mm':[round(s.page_width.mm,2),round(s.page_height.mm,2)],'margins_mm':[round(x.mm,2) for x in [s.left_margin,s.right_margin,s.top_margin,s.bottom_margin]],'styles':{}}
 for name in ['Normal','Heading 1','Heading 2','Listing']:
  st=d.styles[name];p=st.paragraph_format
  out['styles'][name]={'font':st.font.name,'pt':st.font.size.pt,'line_spacing':p.line_spacing,'first_line_mm':round(p.first_line_indent.mm,2) if p.first_line_indent is not None else None}
 out['footer_alignment']=int(s.footer.paragraphs[0].alignment)
 out['page_number_field']='PAGE' in s.footer._element.xml
 return out
baseline=ROOT/'reference/report-format.json'
actual=properties(ROOT/'report/report.docx')
refs=[ROOT.parent/n/'report/report.docx' for n in ['lb4','lb6']]
if all(p.exists() for p in refs):
 reference=[properties(p) for p in refs]
 assert reference[0]==reference[1],'lab 4 and 6 styles differ'
 assert actual==reference[0],'new report formatting differs'
 baseline.write_text(json.dumps(reference[0],ensure_ascii=False,indent=2)+'\n')
else:
 assert actual==json.loads(baseline.read_text()),'new report differs from saved baseline'
result={'reference_reports':['lb4/report/report.docx','lb6/report/report.docx'],'format_matches':True,'checked_properties':actual}
(ROOT/'validation/format-check.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print('Format matches lab 4 and lab 6.')
