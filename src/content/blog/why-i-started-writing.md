---
title: "Why I'm starting to write this down"
description: "Four years in one codebase, a move to a new team, and a question I could not answer about a thing I had already fixed twice."
pubDate: 2026-08-15
tags: ["career", "meta"]
---

Early on at the new job, someone asked me why adding a retry to a particular call had made a failure worse rather than better.

I had fixed that exact failure before. Twice, in a previous codebase, confidently, and both times it stayed fixed. I opened my mouth to explain it and found that what I had was a memory of the incident rather than a reason. I knew the shape of the situation and the thing that worked. I could not say why it worked, which meant I could not tell that person whether it applied to what they were looking at.

<figure>
  <img src="/blog/write-head.svg" alt="Two boxes: in my head, I already know this, versus in a sentence, where does it actually start." width="720" height="220" />
  <figcaption>The comfortable version lives in your head. The real one has to survive a sentence.</figcaption>
</figure>

## Four years of learning by incident

I spent four years at Menlo Security, joining as an intern and leaving as a Senior Engineer, which means I learned the craft inside a single codebase: deeply, and narrowly. You absorb an enormous amount that way. You also pick up assumptions you cannot see, because you have never worked anywhere that made different ones.

Then I moved to Acceldata. Different pace, different domain, and a set of habits I did not know I was carrying until they stopped being true.

<figure>
  <img src="/blog/write-teams.svg" alt="One codebase, deep and narrow, then a new team where the old assumptions show." width="720" height="200" />
  <figcaption>The move did not create the gap. It made it visible.</figcaption>
</figure>

Production teaches you a great deal, but it teaches by incident. You learn networking because a tunnel misbehaved at two in the morning. You learn observability because something broke and nobody could see why. The knowledge is real, and it is shaped like a list of things that have already happened to you rather than a set of principles you can point at a problem you have not met yet.

Changing teams did not create that gap. It just put me in rooms where I had to say things out loud, and the gap was not in what I had done. It was in what I could explain.

<figure>
  <img src="/blog/write-gap.svg" alt="Two boxes: used it, from incidents in one codebase, and can explain it, a model you can reuse." width="720" height="200" />
  <figcaption>The gap is not skill. It is a model you can say out loud.</figcaption>
</figure>

Writing is the fix I settled on, for a boring reason. It is much harder to fool yourself in a paragraph than in your own head. A sentence has to commit to something, and once it is written down you can look at it and notice that it does not actually follow.

## What this is going to be

Notes on systems, networking, and observability. Mental models from data structures and algorithms: the patterns, not the problem numbers. Write-ups of things I have actually hit, while the details are still sharp enough to be checked. And occasionally the parts of this job that are not technical, which turn out to matter more than the stack does.

What it will not be is a set of polished takes with the conclusion picked in advance. If a post changes its mind partway through, that is the process showing rather than something I forgot to edit out. I would rather publish something honest and be corrected than publish something safe and learn nothing from it.

More soon.
