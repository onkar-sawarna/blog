---
title: "Caching is easy until the second write"
description: "Notes on cache invalidation strategies, written while working through system design fundamentals."
pubDate: 2026-08-10
tags: ["system-design", "caching"]
---

Everyone can explain a cache in one sentence: store the expensive result, serve it fast next time. The part that actually separates a working system from a broken one is what happens on the *second* write — when the source of truth changes and the cache doesn't know yet.

## The three strategies that actually matter

**Cache-aside (lazy loading).** The application checks the cache first; on a miss, it reads from the database and populates the cache. Simple, and the default most people reach for. The cost: the first request after any invalidation always pays the full latency.

**Write-through.** Every write goes to the cache and the database together, synchronously. Reads are always fresh, but writes get slower, and you're paying cache-write cost even for data that might never be read again.

**Write-behind (write-back).** Writes land in the cache first and get flushed to the database asynchronously. Fast writes, but a real risk: if the cache node dies before the flush, that write is gone.

## The question that actually matters

Not "which strategy is best" — it's "what happens to a stale read in this specific system." A stale product price for three seconds is a non-event. A stale account balance for three seconds is a support ticket, or worse, an incident.

That's the framing I keep coming back to when a system design problem mentions caching: start from the cost of staleness, then pick the strategy, not the other way around.
