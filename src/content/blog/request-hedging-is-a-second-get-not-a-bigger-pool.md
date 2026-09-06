---
title: "Request hedging is a second GET, not a bigger pool"
description: "A homepage flood, a pool that reuses TCP, an empty Redis key, and a second GET that sits on the same pool."
pubDate: 2026-09-06
tags: ["systems"]
draft: true
---

A pair of shoes goes on the homepage. Call it item 42. A thousand people tap it inside a minute. Every tap is one request to my API for that one product.

My API reads from Redis and from the database. Three things go wrong on that path, one after another. The fix for the third one makes the second one worse.

## Opening a socket per page

The first version of this API did the obvious thing. A request arrives, it opens a connection to the database, it reads the row, it closes the connection.

Opening is not free, because TCP will not carry a query until a connection exists. Establishing one takes three packets going back and forth, known as the three-way handshake: the API sends a SYN meaning "I would like to start a connection," the database replies SYN-ACK meaning "yes, and likewise," and the API sends an ACK. Only after that round trip completes does the actual query leave the machine.

Closing costs more. A clean shutdown is four packets, a FIN and an ACK in each direction, because each side has to say it is finished sending and have that acknowledged. Some stacks combine two of those. Either way it is a conversation, not a single packet, and when it is over the next request starts from nothing.

<figure>
  <img src="/blog/hedge-handshake.svg" alt="Time diagram from the API to the database: SYN, SYN-ACK, ACK, then the query for item 42 and the row, then FIN, ACK and FIN, then ACK." width="720" height="320" />
  <figcaption>Handshake, then the query, then teardown. Every request. That is the phone-call model.</figcaption>
</figure>

One buyer, one handshake, one query, one teardown. That is fine for a demo. A thousand taps in a minute means a thousand handshakes into the database and a thousand teardowns, plus a heap of sockets on both machines sitting in TIME-WAIT, which is a state a closed connection lingers in for a while so that stray packets from it do not get delivered to some unlucky new connection reusing the same port. The query itself was a few bytes. The ceremony around it was the load.

## Four connections that never hang up

So the API stopped dialling per request.

When the process starts, it opens four connections to the database and simply keeps them. Those four pay for their handshakes once, at startup, and then stay in the state TCP calls ESTABLISHED, meaning open and idle and ready to carry data. That set of four is the connection pool.

A request now borrows one of them, runs its query, and hands it back. No FIN, no new SYN. The buyer still gets a page and the database still sees a query, but the handshake is no longer on the path the user waits on.

A thousand buyers share those four sockets. If all four are busy when a fifth request arrives, that request waits for one to come back rather than opening a fifth connection. That waiting is the point rather than a flaw: a pool of four means four queries may run at once, and it is how you stop a traffic spike from turning into a thousand simultaneous queries against a database that cannot serve them.

What the pool does not do is make the database faster. It removes the handshake and the teardown from every tap, and nothing else.

<figure>
  <img src="/blog/hedge-pool-reuse.svg" alt="At process start the API does four handshakes and holds a pool. Each GET checks out, queries item 42, and returns the connection. No FIN and no new SYN." width="720" height="300" />
  <figcaption>A thousand users reuse four handshakes. Teardown waits until the process dies.</figcaption>
</figure>

## Redis has never heard of item 42

Four connections are still four connections, so the page will be slow if every request reaches the database at all. That is what the cache is for. The API looks in Redis first, under a key like `item:42`. If the value is there, the request is answered from memory and the pool is never touched. If it is not, Redis returns nil, and the API has to read the row and then write it back into Redis so the next request stops at the cache.

The homepage flips. Redis has never seen `item:42`, or the key expired. A thousand requests arrive for the same URL within a few seconds.

Every one of them does exactly the same thing. Ask Redis, get nil, borrow a connection from the pool, run the identical query. The pool I was pleased with fills up entirely with a thousand copies of one read. Requests for other products, which have perfectly good cache entries, now queue behind item 42 waiting for a connection to come free. Pages that should have been instant are slow because of a product they have nothing to do with.

The pool saved me the handshake. It did nothing about a thousand identical reads.

<figure>
  <img src="/blog/hedge-redis-miss.svg" alt="Many users send GET 42 to the API. Redis returns nil for item 42. The pool and database run the same row many times and pages get slow." width="720" height="300" />
  <figcaption>Redis had nothing. Every GET became a database read.</figcaption>
</figure>

The fix here is a lock, which is worth naming clearly because it is not what the next section is about. When a request finds `item:42` missing, it tries to claim the right to fill it, and only the first one succeeds. That one worker borrows a connection, reads the row, writes it into Redis, and releases the claim. The other requests, having failed to claim it, wait briefly and then ask Redis again, and by that point the value is there. One read of the database instead of a thousand, and the other three pool connections stay available for everything else.

<figure>
  <img src="/blog/hedge-lock.svg" alt="A lock for key 42. Worker w1 holds it and fills the cache. Workers w2, w3, and w4 wait and then get a cache hit. The pool has one connection busy on 42 and three free for other keys." width="720" height="280" />
  <figcaption>The lock is for the fill. The pool is for the query.</figcaption>
</figure>

If I skip the lock, the page stays slow, and a slow page is exactly the situation where the third idea starts to look like courage.

## Sending the same request twice

Hedging is easy to describe. The user tapped once. The API started reading item 42 at time zero, found nothing in Redis, borrowed a connection, and is now waiting on the database. Fifty milliseconds later there is still no answer, so the API fires off a second copy of the same read, hoping this one comes back sooner. Whichever answer arrives first becomes the page, and the loser is supposed to be cancelled.

It is a reasonable technique and it is aimed at a real problem. Sometimes one request out of a hundred is slow for a reason that has nothing to do with the request: a machine that is briefly overloaded, a disk that is having a bad second, a connection that landed on an unhappy replica. Retrying that one early, rather than waiting for it, pulls in the slowest few percent of response times, which is usually what people mean by tail latency.

Now apply it to my morning. Both copies of the request look in the same Redis, and both find the same nil. Both then borrow a connection from the same pool. A single tap is now holding two of my four connections. A thousand slow taps become two thousand borrowings against a pool of four.

The reason the first request was slow was the empty cache key. The hedge did not address that. It doubled the number of requests piling onto it.

<figure>
  <img src="/blog/hedge-how.svg" alt="One user click. At t=0 the API GETs item 42, Redis is nil, goes to the database. At 50ms it sends the same GET again. First answer wins. Both copies sit on the pool." width="720" height="300" />
  <figcaption>Hedging is a second bet on the same GET. Redis is still empty for both.</figcaption>
</figure>

Hedging earns its keep when the second attempt can land somewhere genuinely different, on another host or another replica, and when the losing copy is actually cancelled so it gives its connection back promptly. Neither of those was true here. And in no arrangement does a hedge write a value into Redis, which is what this page needed.

## What I took from this

Three problems, three tools, and they do not substitute for each other.

The handshake and the teardown are why the pool exists. Many users should share a few connections that stay open, rather than each opening and closing their own conversation with the database.

The empty cache key is a different problem entirely, and the pool cannot help with it. A crowd of identical requests should fill that key once, and a lock is what makes "once" true. The lock governs the fill; the pool governs the query.

Hedging is a third thing again. It duplicates a slow request, which is a reasonable answer to a sick path and a terrible answer to a missing cache entry. Hedge into a stampede and you drain the pool you built to avoid the handshake in the first place.

I still get this wrong in the usual ways. Sizing the pool by how many users I expect rather than by how many concurrent queries the database can actually serve. Letting the losing copy of a hedged request keep running after the winner returns, so it holds a connection nobody is waiting on. And reading a nil from Redis as "try again" when it means "somebody should write this key, once."

If this is useful, wrong, or incomplete, write to me.
