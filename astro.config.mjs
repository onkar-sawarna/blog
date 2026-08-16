// @ts-check
import { defineConfig } from 'astro/config';
import mdx from '@astrojs/mdx';
import sitemap from '@astrojs/sitemap';
import vercel from '@astrojs/vercel';
import { SITE } from './src/config.ts';

export default defineConfig({
  site: SITE.url,
  output: 'static',
  integrations: [mdx(), sitemap()],
  adapter: vercel(),
  vite: {
    server: {
      proxy: {
        '/api': 'http://127.0.0.1:8080',
      },
    },
  },
});
