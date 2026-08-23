// Social scrapers cache aggressively, so the query string has to change when
// the card changes. Hashing the file does that without a version to remember.
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const FALLBACK = '/og.png';
const publicDir = join(process.cwd(), 'public');
const cache = new Map<string, string | null>();

function fingerprint(path: string): string | null {
  if (!cache.has(path)) {
    try {
      const bytes = readFileSync(join(publicDir, path));
      cache.set(path, createHash('sha256').update(bytes).digest('hex').slice(0, 8));
    } catch {
      cache.set(path, null);
    }
  }
  return cache.get(path) ?? null;
}

/**
 * Resolve a public-relative OG image to an absolute, cache-busted URL, falling
 * back to the site card when a post has no generated image yet.
 */
export function ogImageUrl(image: string, site: URL | string): string {
  let version = fingerprint(image);
  let path = image;
  if (version === null) {
    path = FALLBACK;
    version = fingerprint(FALLBACK);
  }
  const url = new URL(path, site);
  if (version) url.searchParams.set('v', version);
  return url.href;
}
