import sharp from 'sharp';

const src = 'public/onkar.jpg';
const sizes = [80, 176, 352];

for (const width of sizes) {
  await sharp(src)
    .resize({ width, withoutEnlargement: true })
    .jpeg({ quality: 78, mozjpeg: true })
    .toFile(`public/onkar-${width}.jpg`);
}
