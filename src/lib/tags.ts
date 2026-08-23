import { getCollection, type CollectionEntry } from 'astro:content';

export function tagSlug(tag: string): string {
  return tag
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function tagPath(tag: string): string {
  return `/tags/${tagSlug(tag)}/`;
}

export async function publishedPosts(): Promise<CollectionEntry<'blog'>[]> {
  const posts = await getCollection('blog', ({ data }) => import.meta.env.DEV || !data.draft);
  return posts.sort((a, b) => b.data.pubDate.valueOf() - a.data.pubDate.valueOf());
}

/** Every tag in use, most-used first, then alphabetical so the order is stable. */
export async function allTags(): Promise<Array<{ tag: string; slug: string; count: number }>> {
  const counts = new Map<string, { tag: string; count: number }>();
  for (const post of await publishedPosts()) {
    for (const tag of post.data.tags) {
      const slug = tagSlug(tag);
      const seen = counts.get(slug);
      if (seen) seen.count += 1;
      else counts.set(slug, { tag, count: 1 });
    }
  }
  return [...counts.entries()]
    .map(([slug, { tag, count }]) => ({ slug, tag, count }))
    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
}
