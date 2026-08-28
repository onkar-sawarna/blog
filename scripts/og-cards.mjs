// Draws the social card for every post in src/content/blog.
//
// A post with an entry in PANELS gets the hand-written two-panel card (wrong
// model on the left, the one that stuck on the right). Anything else gets a
// title card automatically, so publishing never waits on artwork.
//
//   node scripts/og-cards.mjs           only the posts missing a card
//   node scripts/og-cards.mjs --force   redraw everything
import { readdir, readFile, writeFile, mkdir, access } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const postsDir = join(root, 'src/content/blog');
const outDir = join(root, 'public/og');
const portrait = join(root, 'public/onkar.jpg');

const WIDTH = 1200;
const HEIGHT = 630;
const FACE = 268;
const FACE_LEFT = 870;
const FACE_TOP = 168;

const PAPER = '#F7F7F5';
const PAPER_RAISED = '#FFFFFF';
const PAPER_TINT = '#E8EFEA';
const INK = '#161616';
const INK_SOFT = '#5C5C59';
const RULE = '#E0E0DC';
const ACCENT = '#1A3D32';
const ACCENT_WARM = '#B33A1A';

const SERIF = 'Georgia, serif';
const MONO = 'ui-monospace, Menlo, monospace';

// The editorial cards. Left is the model I had, right is the one that replaced it.
const PANELS = {
  'i-thought-a-box-had-64k-connections': {
    leftTitle: 'one laptop',
    leftMid: 'connect() in a loop',
    leftBot: 'hits 64k: claim is fake',
    rightTitle: 'you fixed src',
    rightMid: 'they did not',
    rightBot: 'many IPs, one listen port',
    footer: 'The 64k showed up because you became m1.',
  },
  'i-thought-logs-were-observability': {
    leftTitle: 'looks like seeing',
    leftMid: 'timeout  retry  accept',
    leftBot: 'a healthy pile of logs',
    rightTitle: 'is this getting worse',
    rightMid: '(blank)',
    rightBot: 'the only useful question',
    footer: 'Same incident. One of these is a window.',
  },
  'binary-search-is-a-predicate': {
    leftTitle: 'sorted array',
    leftMid: 'find this number',
    leftBot: 'that is one use',
    rightTitle: 'predicate',
    rightMid: 'ok(k)?',
    rightBot: 'search the first yes',
    footer: 'The array was never the point. The predicate was.',
  },
  'i-thought-kafka-kept-the-order-i-published': {
    leftTitle: 'I published',
    leftMid: '1 then 2 then 3',
    leftBot: 'so it comes back that way',
    rightTitle: 'it does not',
    rightMid: 'order is a lane',
    rightBot: 'the key picks it',
    footer: 'Publish order is not consume order.',
  },
  'sns-sqs-vs-kafka': {
    leftTitle: 'one SQS',
    leftMid: 'two tasks',
    leftBot: 'one of them loses',
    rightTitle: 'SNS copies',
    rightMid: 'or one Kafka log',
    rightBot: 'two cursors',
    footer: 'The inbox belongs to a job, not to a repo.',
  },
  'why-i-started-writing': {
    leftTitle: 'in my head',
    leftMid: 'I already know this',
    leftBot: 'it makes sense in there',
    rightTitle: 'in a sentence',
    rightMid: 'where does it start',
    rightBot: 'the gap has to survive prose',
    footer: 'Prose is where the gap stops hiding.',
  },
};

function esc(text) {
  return String(text)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

/** Enough of YAML to read the frontmatter this blog actually writes. */
function frontmatter(source) {
  const match = source.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match) return {};
  const data = {};
  for (const line of match[1].split(/\r?\n/)) {
    const pair = line.match(/^(\w+):\s*(.*)$/);
    if (!pair) continue;
    const [, key, rawValue] = pair;
    let value = rawValue.trim();
    if (value.startsWith('[') && value.endsWith(']')) {
      data[key] = value
        .slice(1, -1)
        .split(',')
        .map((item) => item.trim().replace(/^["']|["']$/g, ''))
        .filter(Boolean);
      continue;
    }
    value = value.replace(/^["']|["']$/g, '');
    data[key] = value === 'true' ? true : value === 'false' ? false : value;
  }
  return data;
}

// librsvg gives us no text metrics, so estimate: serif glyphs land near half the
// font size, and narrow letters pull the average down.
const NARROW = new Set([...'ijltfrI.,;:\'"!|()[]{} ']);
const WIDE = new Set([...'mwMW@']);

function textWidth(text, size) {
  let units = 0;
  for (const char of text) {
    units += NARROW.has(char) ? 0.34 : WIDE.has(char) ? 0.82 : 0.53;
  }
  return units * size;
}

function wrap(text, size, maxWidth) {
  const lines = [];
  let line = '';
  for (const word of text.split(/\s+/).filter(Boolean)) {
    const candidate = line ? `${line} ${word}` : word;
    if (line && textWidth(candidate, size) > maxWidth) {
      lines.push(line);
      line = word;
    } else {
      line = candidate;
    }
  }
  if (line) lines.push(line);
  return lines;
}

/** Cut to `limit` lines, marking the cut so a clipped sentence does not read as the whole one. */
function ellipsize(lines, limit) {
  if (lines.length <= limit) return lines;
  const kept = lines.slice(0, limit);
  kept[limit - 1] = `${kept[limit - 1].replace(/[\s,;:.]+$/, '')}...`;
  return kept;
}

function panelSvg(card) {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${WIDTH}" height="${HEIGHT}" viewBox="0 0 ${WIDTH} ${HEIGHT}">
  <rect width="${WIDTH}" height="${HEIGHT}" fill="${PAPER}"/>
  <rect width="18" height="${HEIGHT}" fill="${ACCENT}"/>
  <text x="80" y="84" font-family="${SERIF}" font-size="22" fill="${INK_SOFT}">onkarsawarna.dev</text>
  <rect x="80" y="130" width="380" height="300" rx="10" fill="${PAPER_RAISED}" stroke="${ACCENT_WARM}" stroke-width="3"/>
  <text x="270" y="230" text-anchor="middle" font-family="${SERIF}" font-size="30" fill="${ACCENT_WARM}">${esc(card.leftTitle)}</text>
  <text x="270" y="286" text-anchor="middle" font-family="${MONO}" font-size="18" fill="${INK_SOFT}">${esc(card.leftMid)}</text>
  <text x="270" y="340" text-anchor="middle" font-family="${MONO}" font-size="18" fill="${ACCENT_WARM}">${esc(card.leftBot)}</text>
  <rect x="480" y="130" width="380" height="300" rx="10" fill="${PAPER_TINT}" stroke="${ACCENT}" stroke-width="3"/>
  <text x="670" y="230" text-anchor="middle" font-family="${SERIF}" font-size="30" fill="${ACCENT}">${esc(card.rightTitle)}</text>
  <text x="670" y="286" text-anchor="middle" font-family="${MONO}" font-size="18" fill="${INK_SOFT}">${esc(card.rightMid)}</text>
  <text x="670" y="340" text-anchor="middle" font-family="${MONO}" font-size="18" fill="${ACCENT}">${esc(card.rightBot)}</text>
  <text x="470" y="520" text-anchor="middle" font-family="${SERIF}" font-size="22" fill="${INK}">${esc(card.footer)}</text>
</svg>`;
}

function titleSvg({ title, description, tags, pubDate }) {
  const maxWidth = 730;
  let size = 54;
  let lines = wrap(title, size, maxWidth);
  while (lines.length > 3 && size > 34) {
    size -= 4;
    lines = wrap(title, size, maxWidth);
  }
  lines = lines.slice(0, 3);

  const lineHeight = Math.round(size * 1.2);
  const titleTop = 190;
  const titleRows = lines
    .map(
      (line, index) =>
        `<text x="80" y="${titleTop + index * lineHeight}" font-family="${SERIF}" font-size="${size}" font-weight="700" fill="${INK}">${esc(line)}</text>`,
    )
    .join('\n  ');

  const dekTop = titleTop + lines.length * lineHeight + 26;
  const dekLines = ellipsize(wrap(description, 23, maxWidth), 2);
  const dekRows = dekLines
    .map(
      (line, index) =>
        `<text x="80" y="${dekTop + index * 32}" font-family="${SERIF}" font-size="23" fill="${INK_SOFT}">${esc(line)}</text>`,
    )
    .join('\n  ');

  const meta = [tags[0], pubDate].filter(Boolean).join('   ·   ');

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${WIDTH}" height="${HEIGHT}" viewBox="0 0 ${WIDTH} ${HEIGHT}">
  <rect width="${WIDTH}" height="${HEIGHT}" fill="${PAPER}"/>
  <rect width="18" height="${HEIGHT}" fill="${ACCENT}"/>
  <text x="80" y="84" font-family="${SERIF}" font-size="22" fill="${INK_SOFT}">onkarsawarna.dev</text>
  <line x1="80" y1="118" x2="200" y2="118" stroke="${ACCENT}" stroke-width="2"/>
  ${titleRows}
  ${dekRows}
  <line x1="80" y1="524" x2="${maxWidth + 80}" y2="524" stroke="${RULE}" stroke-width="1"/>
  <text x="80" y="562" font-family="${MONO}" font-size="19" fill="${ACCENT}">${esc(meta.toUpperCase())}</text>
</svg>`;
}

function monthYear(value) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return '';
  return date.toLocaleDateString('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' });
}

async function exists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

const force = process.argv.includes('--force');

await mkdir(outDir, { recursive: true });

const circle = Buffer.from(
  `<svg xmlns="http://www.w3.org/2000/svg" width="${FACE}" height="${FACE}"><circle cx="${FACE / 2}" cy="${FACE / 2}" r="${FACE / 2 - 2}" fill="white"/></svg>`,
);
const ring = Buffer.from(
  `<svg xmlns="http://www.w3.org/2000/svg" width="${FACE}" height="${FACE}"><circle cx="${FACE / 2}" cy="${FACE / 2}" r="${FACE / 2 - 2}" fill="none" stroke="${RULE}" stroke-width="3"/></svg>`,
);
const face = await sharp(portrait)
  .extract({ left: 200, top: 0, width: 620, height: 620 })
  .resize(FACE, FACE)
  .composite([{ input: circle, blend: 'dest-in' }])
  .png()
  .toBuffer();

const files = (await readdir(postsDir)).filter((name) => /\.mdx?$/.test(name));
let drawn = 0;

for (const file of files) {
  const slug = file.replace(/\.mdx?$/, '');
  const target = join(outDir, `${slug}.png`);
  if (!force && (await exists(target))) continue;

  const data = frontmatter(await readFile(join(postsDir, file), 'utf8'));
  const panel = PANELS[slug];
  const svg = panel
    ? panelSvg(panel)
    : titleSvg({
        title: data.title ?? slug,
        description: data.description ?? '',
        tags: data.tags ?? [],
        pubDate: monthYear(data.pubDate),
      });

  await sharp(Buffer.from(svg))
    .png()
    .composite([
      { input: face, left: FACE_LEFT, top: FACE_TOP },
      { input: ring, left: FACE_LEFT, top: FACE_TOP },
    ])
    .toFile(target);

  drawn += 1;
  console.log(`og: ${slug}.png${panel ? '' : ' (title card)'}`);
}

console.log(drawn ? `og: drew ${drawn} card(s)` : 'og: all cards present');
