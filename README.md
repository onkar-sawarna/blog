# onkarsawarna.dev

Source for [www.onkarsawarna.dev](https://www.onkarsawarna.dev), my personal site.

I am a software engineer. This repo is the writing: notes on engineering, networking, observability, and distributed systems. Mental models from production work and from DSA, written while the details are still sharp. Not a product blog and not a solutions dump.

Astro builds a static site from Markdown. There is no database and no CMS. Each post is a file in this repo. Vercel deploys on every push to `main`.

## Layout

| Path | What it is |
| :--- | :--------- |
| `src/content/blog/` | Posts. Filename is the URL: `my-post.md` → `/blog/my-post/` |
| `src/config.ts` | Name, domain, social links, newsletter |
| `src/pages/about.astro` | About page |
| `public/blog/` | Original diagrams used in posts |
| `public/og/` | Social preview images |

## Running it locally

Needs Node 22 or newer.

```sh
npm install       # first time only
npm run dev       # http://localhost:4321
```

| Command           | What it does                          |
| :---------------- | :------------------------------------ |
| `npm run dev`     | Dev server at `localhost:4321`        |
| `npm run build`   | Static build into `./dist/`           |
| `npm run preview` | Serve the build to check it first     |

## Writing a post

Add a file under `src/content/blog/`:

```md
---
title: "The title, in quotes"
description: "One or two sentences. Shows on the homepage and in search results."
pubDate: 2026-08-16
tags: ["systems"]
draft: true
---

Your post here, in Markdown.
```

`title`, `description`, and `pubDate` are required. `tags` and `draft` are optional. Keep `draft: true` until the post is ready; drafts stay off the homepage, RSS, and the production build.

RSS is at `/rss.xml`.
