#!/usr/bin/env python3
import sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(root / "notes" / "_py"))

from fpdf import FPDF  # noqa: E402

stem = sys.argv[1] if len(sys.argv) > 1 else "computer-networks"
md = (root / "notes" / f"{stem}.md").read_text()
pdf_path = root / "notes" / f"{stem}.pdf"
face = root / "public" / "onkar-176.jpg"
footer_label = {
    "computer-networks": "computer networks",
    "objects-as-they-show-up-in-a-request": "low-level design",
    "how-i-walk-a-problem": "how I walk a problem",
}.get(stem, stem.replace("-", " "))

INK = (22, 22, 22)
SOFT = (92, 92, 89)
ACCENT = (26, 61, 50)
TINT = (232, 239, 234)
RULE = (224, 224, 220)
PAPER = (247, 247, 245)


class Note(FPDF):
    def footer(self):
        self.set_y(-12)
        self.set_draw_color(*RULE)
        self.set_line_width(0.2)
        self.line(20, self.get_y(), 190, self.get_y())
        self.set_y(-10)
        self.set_font("Helvetica", "", 8)
        self.set_text_color(*SOFT)
        self.cell(0, 6, f"Onkar Sawarna  ·  onkarsawarna.dev  ·  {footer_label}", align="L")
        self.cell(0, 6, str(self.page_no()), align="R")


def strip_md(s: str) -> str:
    s = s.replace("**", "").replace("`", "")
    for a, b in (
        ("\u201c", '"'),
        ("\u201d", '"'),
        ("\u2018", "'"),
        ("\u2019", "'"),
        ("\u2013", "-"),
        ("\u2014", "-"),
        ("\u2022", "-"),
    ):
        s = s.replace(a, b)
    return s


def cells(row: str) -> list[str]:
    parts = [strip_md(p).strip() for p in row.strip().strip("|").split("|")]
    return parts


def is_table_sep(line: str) -> bool:
    body = line.replace("|", "").replace(":", "").replace("-", "").replace(" ", "")
    return line.startswith("|") and body == ""


pdf = Note(format="A4")
pdf.set_margins(20, 18, 20)
pdf.set_auto_page_break(auto=True, margin=18)
pdf.add_page()
left = 20
width = 170

if face.exists():
    pdf.image(str(face), x=left, y=16, w=16, h=16)
pdf.set_xy(40, 17)
pdf.set_font("Helvetica", "B", 11)
pdf.set_text_color(*INK)
pdf.cell(0, 5, "Onkar Sawarna")
pdf.set_xy(40, 23)
pdf.set_font("Helvetica", "", 8)
pdf.set_text_color(*SOFT)
pdf.cell(0, 5, "software engineer  ·  lecture notes")
pdf.set_text_color(*INK)
pdf.ln(14)
pdf.set_draw_color(*ACCENT)
pdf.set_line_width(0.5)
pdf.line(left, pdf.get_y(), 190, pdf.get_y())
pdf.ln(10)

skip_byline = True
in_code = False
lines = md.splitlines()
i = 0

while i < len(lines):
    line = lines[i].rstrip()
    pdf.set_x(left)

    if line.startswith("```"):
        in_code = not in_code
        if in_code:
            pdf.ln(2)
        else:
            pdf.ln(3)
        i += 1
        continue

    if in_code:
        y = pdf.get_y()
        pdf.set_fill_color(*TINT)
        pdf.rect(left, y, width, 4.4, "F")
        pdf.set_font("Courier", "", 8)
        pdf.set_text_color(*ACCENT)
        pdf.set_xy(left + 2, y)
        pdf.cell(width - 4, 4.4, line if line else " ")
        pdf.set_text_color(*INK)
        pdf.ln(4.4)
        i += 1
        continue

    if skip_byline:
        if line.startswith("# "):
            pdf.set_font("Helvetica", "B", 20)
            pdf.set_text_color(*INK)
            pdf.multi_cell(0, 9, strip_md(line[2:]))
            pdf.ln(4)
            i += 1
            continue
        if line.strip() in ("Onkar Sawarna", "onkarsawarna.dev", "software engineer", ""):
            i += 1
            continue
        skip_byline = False
        pdf.ln(2)

    if line.startswith("|") and i + 1 < len(lines) and is_table_sep(lines[i + 1]):
        header = cells(line)
        rows = []
        i += 2
        while i < len(lines) and lines[i].startswith("|"):
            rows.append(cells(lines[i]))
            i += 1
        cols = len(header)
        col_w = width / cols
        row_h = 7

        def draw_row(vals, head=False):
            if pdf.get_y() + row_h > 275:
                pdf.add_page()
                pdf.set_x(left)
            y = pdf.get_y()
            pdf.set_fill_color(*(TINT if head else (255, 255, 255)))
            pdf.set_draw_color(*RULE)
            pdf.set_line_width(0.2)
            pdf.set_font("Helvetica", "B" if head else "", 9)
            pdf.set_text_color(*ACCENT if head else INK)
            x = left
            for c, val in enumerate(vals[:cols]):
                pdf.set_xy(x, y)
                pdf.cell(col_w, row_h, val[:42], border=1, fill=True)
                x += col_w
            pdf.set_y(y + row_h)

        draw_row(header, head=True)
        for r in rows:
            draw_row(r)
        pdf.ln(3)
        pdf.set_text_color(*INK)
        continue

    if line.startswith("## "):
        if pdf.get_y() > 248:
            pdf.add_page()
            pdf.set_x(left)
        pdf.ln(7)
        pdf.set_font("Helvetica", "B", 13)
        pdf.set_text_color(*ACCENT)
        pdf.multi_cell(0, 7, strip_md(line[3:]))
        pdf.set_draw_color(*ACCENT)
        pdf.set_line_width(0.3)
        y = pdf.get_y()
        pdf.line(left, y, left + 36, y)
        pdf.set_text_color(*INK)
        pdf.ln(4)
    elif line.startswith("### "):
        pdf.ln(3)
        pdf.set_font("Helvetica", "B", 10.5)
        pdf.set_text_color(*ACCENT)
        pdf.multi_cell(0, 6, strip_md(line[4:]))
        pdf.set_text_color(*INK)
        pdf.ln(1)
    elif line == "---":
        pdf.ln(3)
        pdf.set_draw_color(*RULE)
        pdf.set_line_width(0.25)
        pdf.line(left, pdf.get_y(), 190, pdf.get_y())
        pdf.ln(4)
    elif line.startswith("> "):
        pdf.set_fill_color(*TINT)
        pdf.set_font("Helvetica", "I", 10.5)
        pdf.set_text_color(*ACCENT)
        pdf.set_x(left)
        pdf.multi_cell(width, 6.2, strip_md(line[2:]), fill=True)
        pdf.set_text_color(*INK)
        pdf.ln(1.5)
    elif line.startswith("**Figure "):
        pdf.ln(1)
        pdf.set_font("Helvetica", "I", 9)
        pdf.set_text_color(*ACCENT)
        pdf.multi_cell(0, 5, strip_md(line))
        pdf.set_text_color(*INK)
        pdf.ln(1.5)
    elif line.startswith("**") and line.endswith("**"):
        pdf.ln(2.5)
        pdf.set_font("Helvetica", "B", 11)
        pdf.set_text_color(*INK)
        pdf.multi_cell(0, 6.2, strip_md(line))
        pdf.ln(1)
    elif line.startswith("- "):
        pdf.set_font("Helvetica", "", 10.5)
        pdf.set_text_color(*INK)
        pdf.set_x(left + 4)
        pdf.multi_cell(0, 5.8, f"-  {strip_md(line[2:])}")
        pdf.ln(0.6)
    elif len(line) >= 3 and line[0].isdigit() and ". " in line[:4]:
        pdf.set_font("Helvetica", "", 10.5)
        pdf.set_x(left + 4)
        pdf.multi_cell(0, 5.8, strip_md(line))
        pdf.ln(0.6)
    elif line.startswith("!["):
        end = line.rfind("](")
        path = line[end + 2 : -1] if end != -1 and line.endswith(")") else ""
        img = (root / "notes" / path).resolve()
        if img.suffix == ".svg":
            png = img.with_suffix(".png")
            if png.exists():
                img = png
        if img.exists() and img.suffix != ".svg":
            pdf.ln(4)
            if pdf.get_y() > 195:
                pdf.add_page()
                pdf.set_x(left)
            pdf.image(str(img), x=left, w=170)
            pdf.ln(2)
    elif line == "":
        pdf.ln(2.4)
    else:
        pdf.set_font("Helvetica", "", 10.5)
        pdf.set_text_color(*INK)
        pdf.multi_cell(0, 6.0, strip_md(line))
        pdf.ln(1.0)

    i += 1

pdf.output(str(pdf_path))
print(pdf_path)
