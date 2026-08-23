import rss from '@astrojs/rss';
import { getCollection } from 'astro:content';
import type { APIContext } from 'astro';
import MarkdownIt from 'markdown-it';
import sanitizeHtml from 'sanitize-html';
import { SITE, SOCIALS } from '../config';

const parser = new MarkdownIt({ html: true, linkify: true, typographer: true });

const mailto = SOCIALS.find((s) => s.href.startsWith('mailto:'))?.href.replace('mailto:', '');
const author = mailto ? `${mailto} (${SITE.author})` : undefined;

/** Feed readers resolve nothing, so every in-post link and diagram needs a full URL. */
function absolutize(html: string, site: string) {
  return html.replace(/(\s(?:src|href))="\/(?!\/)([^"]*)"/g, (_m, attr, path) => {
    return `${attr}="${new URL(`/${path}`, site).href}"`;
  });
}

function toFeedHtml(body: string, site: string) {
  const rendered = sanitizeHtml(parser.render(body), {
    allowedTags: [...sanitizeHtml.defaults.allowedTags, 'img', 'figure', 'figcaption'],
    allowedAttributes: {
      ...sanitizeHtml.defaults.allowedAttributes,
      img: ['src', 'alt', 'title', 'width', 'height', 'loading', 'decoding'],
    },
  });
  return absolutize(rendered, site);
}

export async function GET(context: APIContext) {
  const site = (context.site ?? new URL(SITE.url)).toString();
  const posts = await getCollection('blog', ({ data }) => !data.draft);

  return rss({
    title: SITE.title,
    description: SITE.description,
    site,
    xmlns: { content: 'http://purl.org/rss/1.0/modules/content/' },
    customData: `<language>en</language><copyright>© ${new Date().getFullYear()} ${SITE.author}</copyright>`,
    items: posts
      .sort((a, b) => b.data.pubDate.valueOf() - a.data.pubDate.valueOf())
      .map((post) => ({
        title: post.data.title,
        description: post.data.description,
        pubDate: post.data.updatedDate ?? post.data.pubDate,
        link: `/blog/${post.id}/`,
        categories: post.data.tags,
        author,
        content: toFeedHtml(post.body ?? '', site),
      })),
  });
}
