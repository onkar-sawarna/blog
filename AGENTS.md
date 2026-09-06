## Development

When starting the dev server, use background mode:

```
astro dev --background
```

Manage the background server with `astro dev stop`, `astro dev status`, and `astro dev logs`.

Site URL: `https://www.onkarsawarna.dev`. Personal config lives in `src/config.ts`. Posts are Markdown in `src/content/blog/`.

## Git

Do not name the editor or coding assistant in a commit message, PR title, branch name, comment, or file that ships. Do not add editor config folders to git. Kafka consumer position and CSS pointer style stay as normal engineering words.

## Voice

Write as a working engineer, not a student and not a literary journal.

- First person, short paragraphs, complete sentences. Audience is engineers.
- Normal engineering words are fine: observability, cluster, UI, agent, metrics, traces, host, grep, SSH, DSA. Do not baby-talk ("boss software," "small program," "numbers over time").
- Keep sentences clear. Do not stack unexplained slang or academic filler. Do not paste famous slogans.
- Do not use em dashes (—). Use a comma, colon, period, or parentheses.
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
- Media must be copyright-safe. Only original diagrams, photos you took, or files you have a license to use. No screenshots of docs, no logos, no GIFs or clips from the internet, no stock you did not buy. Prefer SVG or PNG you drew. Put files in `public/blog/`.

## Posts

A post must prove a thought, not announce a blog.

- The story is the spine, not an exhibit inside an argument. Open on one concrete scene with a clock and named actors (9:14, a buyer taps Buy; item 42 is a pair of shoes). Follow that one path the whole way down. State the wrong model in two sentences up top, let the story break it, and close with what the story taught me and what I still get wrong.
- Introduce a term at the moment the story runs into it, never as a glossary up front. The reader should meet "partition" when an event lands on one, not two sections earlier. No vocabulary section before the walk-through.
- Plain language. Use the real name (partition, broker, cursor, hash) and give it one ordinary-language sentence the first time, then use it normally. Say what `hash(key) % 3` actually does. Say a cursor is a bookmark. Say "one single sequence" before leaning on "total order". This is not baby-talk: the rule is no term left unexplained, not simpler words for their own sake.
- Explain the mechanism the concept depends on, not only the concept. Reading not removing the event is why two consumer groups work at all, so it has to be on the page.
- Write full sentences that carry the reader. Clipped fragments read as notes to myself and hide the reasoning.
- Headings are narrative, not a template. "The Friday I added consumers", "Redis has no shoe". Not "The wrong model" / "Where it broke" / "The model that stuck" as fixed section names.
- Still cover the same ground: the model I had, where it broke, the model that replaced it, and what I would still get wrong. Those are beats in the story, not headings to fill in.
- Every systems post needs a real request path, not only definitions. Name the actors (user, API, database, the other job) and one concrete walk-through (checkout, a connect that fails, an agent on a host).
- Do not borrow another post's props for an unrelated idea. A scene may recur across posts that are deliberately about the same system (the checkout runs through both the Kafka and the SNS-SQS posts), as long as they link to each other.
- Every systems post needs original diagrams in `public/blog/` (SVG). Draw the path: boxes, lanes, arrows. Not two caption cards. No screenshots. No broken unicode in SVG text (use ASCII).
- DSA: patterns and mental models, one example problem as proof. Never a solutions dump or "LeetCode #N." Still show the walk (the piles, the `ok` row), plus a diagram.
- Do not ship a post that is only vocabulary (topic, partition, queue) with no scenario and no figure.
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
