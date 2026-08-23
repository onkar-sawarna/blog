// Fails the build if a post heading's anchor no longer points at that heading.
//
// Astro assigns heading ids after markdown plugins run, so src/lib/headingAnchors.ts
// cannot read the id and recomputes the slug with the same slugger instead. That
// works until Astro changes how it slugs, at which point every section link on the
// site breaks and nothing complains. This checks the built HTML rather than the
// assumption.
//
//   node scripts/check-heading-anchors.mjs [dir]
import { readdir, readFile } from 'node:fs/promises';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const target = resolve(root, process.argv[2] ?? 'dist/blog');

const HEADING = /<(h2|h3)\b([^>]*)>([\s\S]*?)<\/\1>/g;
const ID = /\bid="([^"]*)"/;
const ANCHOR = /class="heading-anchor"[^>]*href="#([^"]*)"|href="#([^"]*)"[^>]*class="heading-anchor"/;

async function pages(dir) {
  const out = [];
  for (const entry of await readdir(dir, { withFileTypes: true, recursive: true })) {
    if (entry.isFile() && entry.name === 'index.html') out.push(join(entry.parentPath, entry.name));
  }
  return out;
}

const files = await pages(target).catch(() => []);
if (files.length === 0) {
  console.error(`heading anchors: no pages under ${relative(root, target)}. Build first.`);
  process.exit(1);
}

const problems = [];
let headings = 0;

for (const file of files) {
  const html = await readFile(file, 'utf8');
  const where = relative(root, file);
  for (const match of html.matchAll(HEADING)) {
    const [, tag, attrs, inner] = match;
    const id = attrs.match(ID)?.[1];
    if (!id) continue; // a heading Astro chose not to slug carries no link to break
    headings += 1;
    const anchor = inner.match(ANCHOR);
    const href = anchor?.[1] ?? anchor?.[2];
    if (href == null) problems.push(`${where}: <${tag} id="${id}"> has no anchor`);
    else if (href !== id) problems.push(`${where}: <${tag} id="${id}"> links to #${href}`);
  }
}

// A parse that finds nothing would pass silently, which is worse than no check.
if (headings === 0) {
  console.error(`heading anchors: parsed ${files.length} page(s) and found no slugged headings.`);
  process.exit(1);
}

if (problems.length > 0) {
  console.error(`heading anchors: ${problems.length} of ${headings} broken.`);
  for (const problem of problems) console.error(`  ${problem}`);
  console.error('\nAstro likely changed its heading slugs. Update src/lib/headingAnchors.ts to match.');
  process.exit(1);
}

console.log(`heading anchors: ${headings} checked across ${files.length} page(s), all match.`);
