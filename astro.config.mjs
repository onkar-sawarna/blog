// @ts-check
import { defineConfig } from 'astro/config';
import mdx from '@astrojs/mdx';
import sitemap from '@astrojs/sitemap';
import vercel from '@astrojs/vercel';
import { satteri } from '@astrojs/markdown-satteri';
import { SITE } from './src/config.ts';
import { codeTheme } from './src/lib/shikiTheme.ts';
import { postMedia } from './src/lib/postMedia.ts';
import { headingAnchors } from './src/lib/headingAnchors.ts';

export default defineConfig({
  site: SITE.url,
  output: 'static',
  integrations: [mdx(), sitemap()],
  adapter: vercel(),
  prefetch: {
    prefetchAll: true,
    defaultStrategy: 'viewport',
  },
  markdown: {
    shikiConfig: { theme: codeTheme },
    processor: satteri({ hastPlugins: [postMedia, headingAnchors] }),
  },
  vite: {
    server: {
      proxy: {
        '/api': 'http://127.0.0.1:8080',
      },
    },
  },
});
