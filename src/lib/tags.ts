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

/** Other posts, tag overlap first, then recency, so a finished article still has a next click. */
export function relatedPosts(
  current: CollectionEntry<'blog'>,
  all: CollectionEntry<'blog'>[],
  limit = 3,
): CollectionEntry<'blog'>[] {
  const tags = new Set(current.data.tags);
  return all
    .filter((post) => post.id !== current.id)
    .map((post) => ({
      post,
      overlap: post.data.tags.filter((tag) => tags.has(tag)).length,
    }))
    .sort((a, b) => {
      if (b.overlap !== a.overlap) return b.overlap - a.overlap;
      return b.post.data.pubDate.valueOf() - a.post.data.pubDate.valueOf();
    })
    .slice(0, limit)
    .map(({ post }) => post);
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
