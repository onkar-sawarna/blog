import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const src = join(root, 'notes/computer-networks.md');
const htmlPath = join(root, 'notes/computer-networks.html');
const pdfPath = join(root, 'notes/computer-networks.pdf');
const face = join(root, 'public/onkar-176.jpg');

function inline(text) {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
}

function toHtml(md) {
  const lines = md.replace(/\r\n/g, '\n').split('\n');
  const out = [];
  let i = 0;
  let list = null;

  const closeList = () => {
    if (list) {
      out.push(`</${list}>`);
      list = null;
    }
  };

  while (i < lines.length) {
    const line = lines[i];
    if (line.startsWith('# ')) {
      closeList();
      out.push(`<h1>${inline(line.slice(2))}</h1>`);
    } else if (line.startsWith('## ')) {
      closeList();
      out.push(`<h2>${inline(line.slice(3))}</h2>`);
    } else if (line === '---') {
      closeList();
      out.push('<hr />');
    } else if (line.startsWith('- ')) {
      if (list !== 'ul') {
        closeList();
        list = 'ul';
        out.push('<ul>');
      }
      out.push(`<li>${inline(line.slice(2))}</li>`);
    } else if (/^\d+\.\s/.test(line)) {
      if (list !== 'ol') {
        closeList();
        list = 'ol';
        out.push('<ol>');
      }
      out.push(`<li>${inline(line.replace(/^\d+\.\s/, ''))}</li>`);
    } else if (line.trim() === '') {
      closeList();
    } else {
      closeList();
      out.push(`<p>${inline(line)}</p>`);
    }
    i += 1;
  }
  closeList();
  return out.join('\n');
}

mkdirSync(join(root, 'notes'), { recursive: true });
const body = toHtml(readFileSync(src, 'utf8'));
writeFileSync(
  htmlPath,
  `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <style>
    @page { size: A4; margin: 22mm 20mm 22mm 20mm; }
    body {
      font-family: Georgia, "Source Serif 4", serif;
      color: #161616;
      font-size: 12.5pt;
      line-height: 1.55;
    }
    .mark {
      display: flex;
      align-items: center;
      gap: 14px;
      margin: 0 0 28px;
      padding-bottom: 16px;
      border-bottom: 2px solid #1A3D32;
    }
    .mark img {
      width: 72px;
      height: 72px;
      border-radius: 50%;
      object-fit: cover;
      object-position: 48% 6%;
      border: 1px solid #E0E0DC;
    }
    .mark-name { font-size: 13pt; font-weight: 700; }
    .mark-meta { font-family: ui-monospace, monospace; font-size: 9pt; color: #5C5C59; }
    h1 { font-size: 22pt; letter-spacing: -0.03em; margin: 0 0 0.6em; }
    h2 { font-size: 14.5pt; letter-spacing: -0.02em; margin: 1.6em 0 0.55em; color: #1A3D32; }
    p { margin: 0.7em 0; }
    ul, ol { padding-left: 1.3em; }
    li { margin: 0.25em 0; }
    code { font-family: ui-monospace, monospace; font-size: 0.88em; background: #E8EFEA; padding: 0.05em 0.3em; }
    hr { border: none; border-top: 1px solid #E0E0DC; margin: 1.6em 0; }
    .foot {
      margin-top: 2.5em;
      font-family: ui-monospace, monospace;
      font-size: 8.5pt;
      color: #5C5C59;
    }
  </style>
</head>
<body>
  <div class="mark">
    <img src="${face}" alt="" />
    <div>
      <div class="mark-name">Onkar Sawarna</div>
      <div class="mark-meta">onkarsawarna.dev · software engineer</div>
    </div>
  </div>
  ${body}
  <p class="foot">© ${new Date().getFullYear()} Onkar Sawarna. Original notes. Not a dump of someone else's course.</p>
</body>
</html>`,
);

const chrome = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const result = spawnSync(
  chrome,
  ['--headless=new', '--disable-gpu', `--print-to-pdf=${pdfPath}`, `file://${htmlPath}`],
  { stdio: 'inherit' },
);
if (result.status !== 0) process.exit(result.status ?? 1);
console.log(pdfPath);
