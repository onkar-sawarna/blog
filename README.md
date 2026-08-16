# onkarsawarna.dev

Source for [www.onkarsawarna.dev](https://www.onkarsawarna.dev), my personal site.

I am a software engineer. This repo is the writing: notes on engineering, networking, observability, and distributed systems. Mental models from production work and from DSA, written while the details are still sharp. Not a product blog and not a solutions dump.

Astro builds a static site from Markdown. There is no CMS. Each post is a file in this repo. Vercel deploys on every push to `main`. Post likes are a small Go API plus Redis (Upstash or Vercel KV).

## Layout

| Path | What it is |
| :--- | :--------- |
| `src/content/blog/` | Posts. Filename is the URL: `my-post.md` → `/blog/my-post/` |
| `src/config.ts` | Name, domain, social links, newsletter |
| `src/pages/about.astro` | About page |
| `src/components/LikeButton.astro` | Client `fetch` to the likes API |
| `api/likes.go` | Go handler: GET/POST counts |
| `cmd/likes/main.go` | Local HTTP server that calls that handler |
| `public/blog/` | Original diagrams used in posts |
| `public/og/` | Social preview images |

## Architecture

This is a static site plus one small API. There is no CMS and no database for posts.

**Astro** turns Markdown into HTML at build time (`output: 'static'`). A file `src/content/blog/my-post.md` becomes `/blog/my-post/`. The homepage, About, RSS, and sitemap are also files after `astro build`. Opening a post does not run Node or Go.

**Go** only handles like counts. `Handler` in `api/likes.go` is the entry. The browser never talks to Redis.

```
Markdown ──► Astro build ──► HTML/CSS/JS on Vercel
                                  │
Reader ──► page ──► LikeButton ──► GET/POST /api/likes/:id
                                  │
                           Go Handler
                                  │
                    Upstash Redis (likes:<id>)
                    or .data/likes.json on a laptop
```

**Production.** One push to `main`. Vercel builds the Astro site and compiles `api/likes.go` as a function at `/api/likes`. `vercel.json` rewrites `/api/likes/:id` to `/api/likes?id=:id` so the function sees the post id. Redis env vars must be set on Vercel.

**Local.** `npm run dev` starts `cmd/likes` on `127.0.0.1:8080`, then Astro on `4321`. Vite proxies `/api` to the Go process. Same `Handler`, same JSON: `{ "count": N }` and POST `{ "action": "like" | "unlike" }`.

“Already liked” is `localStorage` in the browser (`liked:<id>`). The server stores a number only.

Theme, subscribe (Buttondown), and analytics do not go through Go.

## Running it locally

Needs Node 22 or newer. Likes also need Go 1.22 or newer.

```sh
npm install       # first time only
npm run dev       # site at localhost:4321, likes API at localhost:8080
```

| Command           | What it does                          |
| :---------------- | :------------------------------------ |
| `npm run dev`     | Site and likes API together           |
| `npm run dev:site`| Astro only, at `localhost:4321`       |
| `npm run dev:api` | Go likes API only, at `localhost:8080`|
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

## Likes

Each post has a like button. The count API is Go (`api/likes.go`). Counts live in Redis, not in the Markdown.

Create a free [Upstash Redis](https://upstash.com) database (or Vercel KV) and set these on Vercel, then redeploy:

```
UPSTASH_REDIS_REST_URL=
UPSTASH_REDIS_REST_TOKEN=
```

Locally, `npm run dev` starts the Go API next to Astro. If Redis env vars are missing, counts are stored in `.data/likes.json` and stay on your machine.
