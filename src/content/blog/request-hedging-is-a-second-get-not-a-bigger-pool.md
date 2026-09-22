---
title: "Request hedging is a second GET, not a bigger pool"
description: "A homepage flood, a pool that reuses TCP, an empty Redis key, and a second GET that sits on the same pool."
pubDate: 2026-09-06
tags: ["systems"]
---

A pair of shoes goes on the homepage. Call it item 42. A thousand people tap it inside a minute. Every tap is one request to my API.

My API reads Redis, then the database. Three things go wrong on that path. The fix for the third one makes the second one worse.

## Opening a socket per page

The first version opened a database connection, read the row, and closed it.

Opening is not free. TCP will not carry a query until a connection exists. That takes three packets: the API sends SYN ("I want to start"), the database replies SYN-ACK ("yes"), the API sends ACK. Only then does the query leave.

Closing is four packets: FIN and ACK both ways. When that is over, the next request starts from nothing.

<figure>
  <object class="figure-svg" data="/blog/hedge-handshake.svg" type="image/svg+xml" width="720" height="320" style="aspect-ratio: 720 / 320" aria-label="Time diagram from the API to the database: SYN, SYN-ACK, ACK, then the query for item 42 and the row, then FIN, ACK and FIN, then ACK.">
    <img src="/blog/hedge-handshake.svg" alt="Time diagram from the API to the database: SYN, SYN-ACK, ACK, then the query for item 42 and the row, then FIN, ACK and FIN, then ACK." width="720" height="320" />
  </object>
  <figcaption>Figure 1. Handshake, then the query, then teardown. Every request.</figcaption>
</figure>

One buyer, one handshake, one query, one teardown. A thousand taps means a thousand of those. Closed sockets also linger in TIME-WAIT so leftover packets do not hit a new connection on the same port. The query was a few bytes. The ceremony was the load.

## Four connections that never hang up

So the process opens four connections at startup and keeps them. Those four pay the handshake once. That set is the connection pool.

A request borrows one, runs the query, and hands it back. No FIN, no new SYN.

A thousand buyers share those four. If all four are busy, the fifth waits. That wait is the point: at most four queries at once, so a spike cannot become a thousand simultaneous reads.

The pool does not make the database faster. It only removes the handshake from the tap.

<figure>
  <object class="figure-svg" data="/blog/hedge-pool-reuse.svg" type="image/svg+xml" width="720" height="300" style="aspect-ratio: 720 / 300" aria-label="At process start the API does four handshakes and holds a pool. Each GET checks out, queries item 42, and returns the connection. No FIN and no new SYN.">
    <img src="/blog/hedge-pool-reuse.svg" alt="At process start the API does four handshakes and holds a pool. Each GET checks out, queries item 42, and returns the connection. No FIN and no new SYN." width="720" height="300" />
  </object>
  <figcaption>Figure 2. A thousand users reuse four handshakes.</figcaption>
</figure>

## Redis has never heard of item 42

The page is still slow if every request hits the database. The API looks in Redis first, under `item:42`. Hit: answer from memory, pool untouched. Miss: Redis returns nil, the API reads the row and writes it back.

The homepage flips. Redis has no `item:42`. A thousand requests arrive.

Every one asks Redis, gets nil, borrows a pool connection, runs the same query. The pool fills with copies of one read. Other products wait behind item 42.

The pool saved the handshake. It did nothing about a thousand identical reads.

<figure>
  <img src="/blog/hedge-redis-miss.svg" alt="Many users send GET 42 to the API. Redis returns nil for item 42. The pool and database run the same row many times and pages get slow." width="720" height="300" />
  <figcaption>Figure 3. Redis had nothing. Every GET became a database read.</figcaption>
</figure>

The fix is a lock. The first request to see the miss claims the right to fill `item:42`. It reads the row, writes Redis, lets go. The others wait, then ask Redis again. One database read. Three pool connections stay free.

<figure>
  <img src="/blog/hedge-lock.svg" alt="A lock for key 42. Worker w1 holds it and fills the cache. Workers w2, w3, and w4 wait and then get a cache hit. The pool has one connection busy on 42 and three free for other keys." width="720" height="280" />
  <figcaption>Figure 4. The lock is for the fill. The pool is for the query.</figcaption>
</figure>

Skip the lock and the page stays slow. A slow page is when the third idea looks like courage.

## Sending the same request twice

Hedging: the user tapped once. The API is waiting on the database. Fifty milliseconds later it fires a second copy of the same read. First answer wins. The loser should be cancelled.

That helps when one path is sick: a bad host, a bad disk. It pulls in the slowest few percent, the tail.

On this flood both copies see the same nil. Both borrow from the same pool. One tap holds two of four connections. A thousand slow taps become two thousand borrowings.

The first request was slow because the key was empty. The hedge doubled the pile.

<figure>
  <object class="figure-svg" data="/blog/hedge-how.svg" type="image/svg+xml" width="720" height="300" style="aspect-ratio: 720 / 300" aria-label="One user click. At t=0 the API GETs item 42, Redis is nil, goes to the database. At 50ms it sends the same GET again. First answer wins. Both copies sit on the pool.">
    <img src="/blog/hedge-how.svg" alt="One user click. At t=0 the API GETs item 42, Redis is nil, goes to the database. At 50ms it sends the same GET again. First answer wins. Both copies sit on the pool." width="720" height="300" />
  </object>
  <figcaption>Figure 5. Hedging is a second GET. Redis is still empty for both.</figcaption>
</figure>

Hedging only helps if the second attempt can land somewhere different, and if the loser actually gives the connection back. Neither was true here. And a hedge never writes Redis, which is what the page needed.

The handshake is why the pool exists. The empty key is why the lock exists. Hedging is a second GET. It does not fill the key.

I still size the pool by user count instead of by how many queries the database can run at once. I still let the losing hedge keep running.

More API boxes do not turn those four sockets into a safe MySQL budget. I wrote that [when the shop outgrew one process](/blog/the-api-should-see-mysql-not-the-topology/). A Redis lock that fills `item:42` is still not the row. I wrote that [when two people booked the same seat](/blog/two-passengers-one-seat/).

If this is useful, wrong, or incomplete, write to me.
