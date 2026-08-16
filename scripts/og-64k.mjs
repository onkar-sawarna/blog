import sharp from 'sharp';

const WIDTH = 1200;
const HEIGHT = 630;
const FACE = 268;

const card = Buffer.from(`<svg xmlns="http://www.w3.org/2000/svg" width="${WIDTH}" height="${HEIGHT}" viewBox="0 0 ${WIDTH} ${HEIGHT}">
  <rect width="${WIDTH}" height="${HEIGHT}" fill="#F7F7F5"/>
  <rect width="18" height="${HEIGHT}" fill="#1A3D32"/>
  <text x="80" y="84" font-family="Georgia, serif" font-size="22" fill="#5C5C59">onkarsawarna.dev</text>
  <rect x="80" y="130" width="380" height="300" rx="10" fill="#FFFFFF" stroke="#B33A1A" stroke-width="3"/>
  <text x="270" y="230" text-anchor="middle" font-family="Georgia, serif" font-size="32" fill="#B33A1A">one laptop</text>
  <text x="270" y="286" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="20" fill="#5C5C59">connect() in a loop</text>
  <text x="270" y="340" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="20" fill="#B33A1A">hits 64k: claim is fake</text>
  <rect x="480" y="130" width="380" height="300" rx="10" fill="#E8EFEA" stroke="#1A3D32" stroke-width="3"/>
  <text x="670" y="230" text-anchor="middle" font-family="Georgia, serif" font-size="32" fill="#1A3D32">you fixed src</text>
  <text x="670" y="286" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="20" fill="#5C5C59">they did not</text>
  <text x="670" y="340" text-anchor="middle" font-family="ui-monospace, Menlo, monospace" font-size="20" fill="#1A3D32">many IPs, one listen port</text>
  <text x="470" y="520" text-anchor="middle" font-family="Georgia, serif" font-size="24" fill="#161616">The 64k showed up because you became m1.</text>
</svg>`);

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

await sharp(card)
  .png()
  .composite([
    { input: face, left: 870, top: 168 },
    { input: ring, left: 870, top: 168 },
  ])
  .toFile('public/og/i-thought-a-box-had-64k-connections.png');
