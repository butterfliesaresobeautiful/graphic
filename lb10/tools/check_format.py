"""Compare the report with the layout of the delivered lab 4 and 6 documents."""
import json
from pathlib import Path

from docx import Document

ROOT = Path(__file__).resolve().parents[1]
BASELINE = ROOT / "reference/report-format.json"


def mm(length):
    return round(length.mm, 2) if length is not None else None


def properties(path):
    doc = Document(path)
    section = doc.sections[0]
    cover = []
    for paragraph in doc.paragraphs[:22]:
        cover.append({
            "style": paragraph.style.name,
            "alignment": paragraph.alignment,
            "blank": not paragraph.text,
            "indent_mm": mm(paragraph.paragraph_format.first_line_indent),
            "bold": any(run.bold for run in paragraph.runs),
        })
    result = {
        "page_mm": [mm(section.page_width), mm(section.page_height)],
        "margins_mm": [
            mm(section.left_margin), mm(section.right_margin),
            mm(section.top_margin), mm(section.bottom_margin),
        ],
        "header_footer_mm": [mm(section.header_distance), mm(section.footer_distance)],
        "styles": {},
        "footer_alignment": int(section.footer.paragraphs[0].alignment),
        "page_number_field": 'w:instr="PAGE"' in section.footer._element.xml,
        "cover": cover,
    }
    for name in ["Normal", "Heading 1", "Heading 2", "Heading 3", "Listing"]:
        style = doc.styles[name]
        fmt = style.paragraph_format
        result["styles"][name] = {
            "font": style.font.name,
            "pt": style.font.size.pt,
            "line_spacing": fmt.line_spacing,
            "first_line_mm": mm(fmt.first_line_indent),
            "space_before_pt": fmt.space_before.pt if fmt.space_before is not None else None,
            "space_after_pt": fmt.space_after.pt if fmt.space_after is not None else None,
        }
    return result


references = [ROOT.parent / lab / "report/report.docx" for lab in ("lb4", "lb6")]
current = properties(ROOT / "report/report.docx")
if all(path.is_file() for path in references):
    reference = properties(references[0])
    assert reference == properties(references[1]), "lab 4 and lab 6 formatting differs"
    assert current == reference, "lab 10 formatting differs from lab 4 and lab 6"
    BASELINE.write_text(json.dumps(reference, ensure_ascii=False, indent=2) + "\n")
else:
    assert current == json.loads(BASELINE.read_text()), "report differs from saved reference"

record = {
    "reference_reports": ["lb4/report/report.docx", "lb6/report/report.docx"],
    "format_matches": True,
    "checked_properties": current,
}
(ROOT / "validation/format-check.json").write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n")
print("Cover, page layout, numbering and styles match lab 4 and lab 6.")
