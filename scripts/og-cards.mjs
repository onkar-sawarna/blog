import sharp from 'sharp';

const WIDTH = 1200;
const HEIGHT = 630;
const FACE = 268;

const cards = [
  {
    file: 'public/og/i-thought-a-box-had-64k-connections.png',
    leftTitle: 'one laptop',
    leftMid: 'connect() in a loop',
    leftBot: 'hits 64k: claim is fake',
    rightTitle: 'you fixed src',
    rightMid: 'they did not',
    rightBot: 'many IPs, one listen port',
    footer: 'The 64k showed up because you became m1.',
  },
  {
    file: 'public/og/i-thought-logs-were-observability.png',
    leftTitle: 'looks like seeing',
    leftMid: 'timeout  retry  accept',
    leftBot: 'a healthy pile of logs',
    rightTitle: 'is this getting worse',
    rightMid: '(blank)',
    rightBot: 'the only useful question',
    footer: 'Same incident. One of these is a window.',
  },
  {
    file: 'public/og/why-i-started-writing.png',
    leftTitle: 'in my head',
    leftMid: 'I already know this',
    leftBot: 'it makes sense in there',
    rightTitle: 'in a sentence',
    rightMid: 'where does it start',
    rightBot: 'the gap has to survive prose',
    footer: 'Prose is where the gap stops hiding.',
  },
];

const circle = Buffer.from(
  `<svg xmlns="http://www.w3.org/2000/svg" width="${FACE}" height="${FACE}"><circle cx="${FACE / 2}" cy="${FACE / 2}" r="${FACE / 2 - 2}" fill="white"/></svg>`,
);

const face = await sharp('public/onkar.jpg')
  .extract({ left: 200, top: 0, width: 620, height: 620 })
  .resize(FACE, FACE)
  .composite([{ input: circle, blend: 'dest-in' }])
  .png()
  .toBuffer();

const ring = Buffer.from(
  `<svg xmlns="http://www.w3.org/2000/svg" width="${FACE}" height="${FACE}"><circle cx="${FACE / 2}" cy="${FACE / 2}" r="${FACE / 2 - 2}" fill="none" stroke="#E0E0DC" stroke-width="3"/></svg>`,
);

function svgFor(card) {
  return Buffer.from(`<svg xmlns="http://www.w3.org/2000/svg" width="${WIDTH}" height="${HEIGHT}" viewBox="0 0 ${WIDTH} ${HEIGHT}">
  <rect width="${WIDTH}" height="${HEIGHT}" fill="#F7F7F5"/>
  <rect width="18" height="${HEIGHT}" fill="#1A3D32"/>
  <text x="80" y="84" font-family="Georgia, serif" font-size="22" fill="#5C5C59">onkarsawarna.dev</text>
  <rect x="80" y="130" width="380" height="300" rx="10" fill="#FFFFFF" stroke="#B33A1A" stroke-width="3"/>
  <text x="270" y="230" text-anchor="middle" font-family="Georgia, serif" font-size="30" fill="#B33A1A">${card.leftTitle}</text>
  <text x="270" y="286" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="18" fill="#5C5C59">${card.leftMid}</text>
  <text x="270" y="340" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="18" fill="#B33A1A">${card.leftBot}</text>
  <rect x="480" y="130" width="380" height="300" rx="10" fill="#E8EFEA" stroke="#1A3D32" stroke-width="3"/>
  <text x="670" y="230" text-anchor="middle" font-family="Georgia, serif" font-size="30" fill="#1A3D32">${card.rightTitle}</text>
  <text x="670" y="286" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="18" fill="#5C5C59">${card.rightMid}</text>
  <text x="670" y="340" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="18" fill="#1A3D32">${card.rightBot}</text>
  <text x="470" y="520" text-anchor="middle" font-family="Georgia, serif" font-size="22" fill="#161616">${card.footer}</text>
</svg>`);
}

for (const card of cards) {
  await sharp(svgFor(card))
    .png()
    .composite([
      { input: face, left: 870, top: 168 },
      { input: ring, left: 870, top: 168 },
    ])
    .toFile(card.file);
}
