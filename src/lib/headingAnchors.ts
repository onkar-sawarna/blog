// Astro slugs headings itself, but only after this plugin runs, so the href is
// recomputed here with the same slugger. A factory keeps the duplicate-heading
// counter per document. The anchor is left empty (the marker comes from CSS) so
// it cannot change the text Astro slugs from.
import GithubSlugger from 'github-slugger';
import type { HastPluginDefinition } from 'satteri';

export function headingAnchors(): HastPluginDefinition {
  const slugger = new GithubSlugger();
  return {
    name: 'heading-anchors',
    element: {
      filter: ['h2', 'h3'],
      visit(node, ctx) {
        const text = ctx.textContent(node);
        if (!text.trim()) return;
        ctx.appendChild(node, {
          type: 'element',
          tagName: 'a',
          properties: {
            className: ['heading-anchor'],
            href: `#${slugger.slug(text)}`,
            'aria-label': `Link to section: ${text}`,
          },
          children: [],
        });
      },
    },
  };
}
