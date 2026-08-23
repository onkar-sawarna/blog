// Posts are diagram-heavy and every figure sits below the fold, so nothing in
// the body needs to block first paint.
//
// Figures are hand-written HTML in the Markdown, which reaches the tree as a
// raw node rather than a parsed element, so both shapes need handling.
import type { HastPluginDefinition } from 'satteri';

const LAZY_ATTRS: Record<string, Record<string, string>> = {
  img: { loading: 'lazy', decoding: 'async' },
  iframe: { loading: 'lazy' },
  video: { preload: 'none' },
};

export const postMedia: HastPluginDefinition = {
  name: 'post-media',
  element: {
    filter: Object.keys(LAZY_ATTRS),
    visit(node, ctx) {
      for (const [key, value] of Object.entries(LAZY_ATTRS[node.tagName] ?? {})) {
        if (node.properties?.[key] == null) ctx.setProperty(node, key, value);
      }
    },
  },
  raw(node) {
    const value = addAttrsToRawHtml(node.value);
    if (value !== node.value) return { ...node, value };
  },
};

function addAttrsToRawHtml(html: string) {
  const tags = Object.keys(LAZY_ATTRS).join('|');
  return html.replace(
    new RegExp(`<(${tags})\\b([^>]*?)(\\s*/)?>`, 'gi'),
    (match, tag: string, existing: string, selfClose: string | undefined) => {
      const added = Object.entries(LAZY_ATTRS[tag.toLowerCase()])
        .filter(([key]) => !new RegExp(`\\s${key}\\s*=`, 'i').test(existing))
        .map(([key, value]) => ` ${key}="${value}"`)
        .join('');
      return added ? `<${tag}${existing}${added}${selfClose ?? ''}>` : match;
    },
  );
}
