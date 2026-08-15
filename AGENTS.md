## Development

When starting the dev server, use background mode:

```
astro dev --background
```

Manage the background server with `astro dev stop`, `astro dev status`, and `astro dev logs`.

Site URL: `https://www.onkarsawarna.dev`. Personal config lives in `src/config.ts`. Posts are Markdown in `src/content/blog/`.

## Voice

Write as a working engineer, not a student and not a literary journal.

- First person, short paragraphs, complete sentences. Audience is engineers.
- Normal engineering words are fine: observability, cluster, UI, agent, metrics, traces, host, grep, SSH, DSA. Do not baby-talk ("boss software," "small program," "numbers over time").
- Keep sentences clear. Do not stack unexplained slang or academic filler. Do not paste famous slogans.
- Confident and specific about *ideas*. Broad about *employers*.
- Job title is **software engineer**, never Platform Engineer.
- Acceldata: observability. Menlo: intern → Senior Engineer, cybersecurity and networking.
- Do not say "I am now studying," "masterclass," "course," or "learning in public."
- Do not use cute framing: "Vol. 1," "breaking things and shipping anyway," "working log" as a personality.
- Homepage and About must sound like the same person.
- About lede / site through-line: **engineering, networking, observability, and distributed systems.**
- About heading: "Hey, I am Onkar." Photo only on About (`public/onkar.jpg`), circular crop. Do not retune the portrait CSS unless asked.

## What not to put on the public site

On About and in posts, do not name products, internal tools, customers, or coworkers.

- No work inventory.
- No company-specific install steps, CLIs, or architecture.
- Describe the *shape* instead: an agent on a customer host, a cluster manager, a UI that has to show health.
- Do not name other writers, courses, or companies when explaining an idea. Use original wording. Do not paste slogans (e.g. famous observability one-liners).
- Do not copy other blogs, docs, or course notes. Ideas are fine; someone else's sentences are not.

## Posts

A post must prove a thought, not announce a blog.

- Prefer: wrong model → where it broke → model that stuck → how I notice the old model → what I would still get wrong.
- DSA: patterns and mental models, one example problem as proof. Never a solutions dump or "LeetCode #N."
- Do not lead the site with a "why I started writing" manifesto. About already covers that.
- Filename is the URL: `src/content/blog/my-post.md` → `/blog/my-post/`.
- Required frontmatter: `title`, `description`, `pubDate`. Optional: `tags`, `draft`.
- `draft: true` until it is ready. No product names in titles or descriptions.

## Design

- Palette is off-white / near-black (`src/styles/global.css`). Do not go back to cream paper.
- Do not add pages, nav items, or chrome unless asked. Next leverage is another real post, not more UI.

## Documentation

Full documentation: https://docs.astro.build

Consult these guides before working on related tasks:

- [Adding pages, dynamic routes, or middleware](https://docs.astro.build/en/guides/routing/)
- [Working with Astro components](https://docs.astro.build/en/basics/astro-components/)
- [Using React, Vue, Svelte, or other framework components](https://docs.astro.build/en/guides/framework-components/)
- [Adding or managing content](https://docs.astro.build/en/guides/content-collections/)
- [Adding styles or using Tailwind](https://docs.astro.build/en/guides/styling/)
- [Supporting multiple languages](https://docs.astro.build/en/guides/internationalization/)
