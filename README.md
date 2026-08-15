# Personal blog

An Astro blog: Markdown posts, no database, no CMS. Every post is a file in this repo.

## Running it locally

```sh
npm install       # first time only
npm run dev       # http://localhost:4321
```

The dev server reloads as you save, so you can write a post with it open in a second window.

| Command           | What it does                             |
| :---------------- | :--------------------------------------- |
| `npm run dev`     | Local dev server at `localhost:4321`     |
| `npm run build`   | Builds the static site into `./dist/`    |
| `npm run preview` | Serves the built site, to check it first |

## Making it yours

Everything personal — site title, your name, social links, domain — lives in `src/config.ts`.
Nothing else needs editing to change those. The long-form bio is in `src/pages/about.astro`.

## Writing a post

Create a file in `src/content/blog/`, e.g. `src/content/blog/my-post.md`. The filename becomes
the URL, so `my-post.md` is served at `/blog/my-post/`.

```md
---
title: "The title, in quotes"
description: "One or two sentences. Shows on the homepage and in search results."
pubDate: 2026-08-15
tags: ["systems", "career"]
draft: false
---

Your post here, in Markdown.

## A section heading

Regular paragraphs, `inline code`, [links](https://example.com), and lists all work.
```

`title`, `description`, and `pubDate` are required. `tags` and `draft` are optional —
set `draft: true` and the post stays out of the homepage, RSS feed, and build until you flip it.

Posts are sorted newest-first automatically. RSS lives at `/rss.xml` and updates itself.

## Deploying

Push this repo to GitHub, then import it at [vercel.com/new](https://vercel.com/new) or
[app.netlify.com](https://app.netlify.com/start). Both detect Astro and need no configuration.
Every push to `main` redeploys.

After you attach a custom domain, set `url` in `src/config.ts` to that domain so RSS,
the sitemap, and social previews point at the right place.
