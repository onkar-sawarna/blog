---
title: "Redis can hold a key. MySQL holds the seat."
description: "Two people tap Book on seat 12A. Lock the row, check a version, or take a key in Redis. The seat row is still the truth."
pubDate: 2026-09-14
tags: ["systems"]
---

Passenger A taps Book on seat 12A. Passenger B taps Book on the same seat. Two requests hit my API. The row still says empty.

I used to think a lock was one move: grab 12A, write booked, let go. That is one kind. The same two taps need the others.

<figure>
  <object class="figure-svg" data="/blog/lock-two-taps.svg" type="image/svg+xml" width="720" height="260" style="aspect-ratio: 720 / 260" aria-label="Passenger A taps Book on 12A. Then B taps the same seat. Two requests hit one empty row.">
    <img src="/blog/lock-two-taps.svg" alt="Passenger A taps Book on 12A. Then B taps the same seat. Two requests hit one empty row." width="720" height="260" />
  </object>
  <figcaption>Figure 1. One seat. Then a second tap.</figcaption>
</figure>

## Lock 12A first, or check after

Pessimistic locking means I assume a fight. A starts a transaction, locks the row for 12A, sees it empty, writes booked, commits. The lock drops. B has been waiting. B now reads booked and gets "seat taken."

I pay the wait so 12A is not sold twice.

Optimistic locking means I assume they usually do not fight. A and B both read 12A empty. The row has a version, say 3. A writes booked and sets version to 4 where version is still 3. That update changes one row. B runs the same update. Zero rows change. B tells the passenger the seat is gone.

No one waited. The second writer lost at the update, not at a lock.

<figure>
  <img src="/blog/lock-pess-opt.svg" alt="Left: A locks 12A, writes, unlocks. B waits then sees booked. Right: both read version 3. A writes version 4. B update hits zero rows." width="720" height="300" />
  <figcaption>Figure 2. Same seat. Lock before, or check after.</figcaption>
</figure>

There is one 12A. Two taps on it is contention. I pick pessimistic when that fight is the normal path, a cheap fare that just opened. I pick optimistic when collisions are rare, or when I do not want B sitting on a lock while A talks to a payment service.

## Someone has to keep the wait list

When B waits, it is not magic. Inside the database there is a lock manager: which row, who holds it, who is waiting. A asks for an exclusive lock on 12A. The manager writes "holder: A." B asks for the same row. The manager puts B on the wait list. A commits. The manager wakes B.

The lock is not the seat. The seat is the row. The lock is that entry. Optimistic booking often never talks to this table on the read. It only finds out at update time that version 3 is gone.

Shared and exclusive are two modes on that same table. Two check-in screens can take a shared lock to print that 12A is still free. `FOR SHARE` is that mode. A booking needs exclusive: `FOR UPDATE`. The manager will not give exclusive while a share is out, and it will not give a new share while exclusive is out. Read together. Write alone.

<figure>
  <img src="/blog/lock-manager.svg" alt="Transactions A and B talk to a lock manager. Row 12A is exclusive, holder A, waiter B. The seat row is separate." width="720" height="280" />
  <figcaption>Figure 3. The lock is not the seat. It is an entry in that table.</figcaption>
</figure>

## They also want 12B

A is booking a pair: 12A and 12B. A locks 12A. B is booking the same pair from the other side. B locks 12B. A wants 12B. B wants 12A. Each holds what the other needs. That cycle is a deadlock. Nobody can finish.

MySQL looks for that cycle soon. If an edge will close a loop, it kills one transaction. A gets a deadlock error. B books both seats. A retries.

Postgres lets them wait. After a timeout it builds a wait-for graph: who is waiting on whom. If it finds a cycle, it kills one. If it does not, they keep waiting.

Same two passengers. Same pair. MySQL fails fast. Postgres waits, then looks.

<figure>
  <object class="figure-svg" data="/blog/lock-deadlock.svg" type="image/svg+xml" width="720" height="280" style="aspect-ratio: 720 / 280" aria-label="A holds 12A and wants 12B. B holds 12B and wants 12A. MySQL kills one now. Postgres waits then walks the wait-for graph.">
    <img src="/blog/lock-deadlock.svg" alt="A holds 12A and wants 12B. B holds 12B and wants 12A. MySQL kills one now. Postgres waits then walks the wait-for graph." width="720" height="280" />
  </object>
  <figcaption>Figure 4. A cycle is a deadlock. Someone has to lose.</figcaption>
</figure>

If every booking locks 12A then 12B, the cycle does not form. The kill is a safety net, not a booking strategy.

## Wait, fail, or take 12B

`FOR UPDATE` is the exclusive row lock. A has 12A. B's `SELECT ... FOR UPDATE` on 12A waits until A commits.

`NOWAIT` means B does not wait. If 12A is locked, the database returns an error now.

`SKIP LOCKED` means B does not wait and does not error. It skips 12A and takes the next free row, 12B. That is "any window," not "I asked for 12A."

<figure>
  <img src="/blog/lock-skip-nowait.svg" alt="FOR UPDATE waits on 12A. NOWAIT errors. SKIP LOCKED takes 12B." width="720" height="280" />
  <figcaption>Figure 5. Wait, fail, or skip.</figcaption>
</figure>

If B must have 12A, do not skip.

## A lock in Redis is not 12A

The next version puts A and B on two API boxes. Those boxes do not share memory. They ask Redis for `seat:12A`.

A calls `SET seat:12A <token> NX EX 30`. NX means set only if the key is missing. EX 30 means the key dies in thirty seconds if the box crashes. A gets the key. B gets nil and does not book. A writes the MySQL row, then unlocks.

Unlock is not `DEL seat:12A`. If A's lock expired and C now holds the key, A's late delete would steal C's lock. A Lua script runs on Redis as one step: read the value, delete only if it is still A's token.

```
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
end
return 0
```

That lock and that write are two systems. The Redis key can expire while the MySQL write is still running. B can take the key and write again. Or the write fails and the key remains, and 12A looks taken when the row is empty.

A remote lock is a hint plus a timeout. The seat row is still the truth. I still do the version check or the `FOR UPDATE` on the primary when I write booked.

Worse: I take `FOR UPDATE` on a read replica. The replica is a copy. The write goes to the primary. I locked a picture of 12A. Two bookings land. The lock has to sit where the write happens.

<figure>
  <img src="/blog/lock-remote-replica.svg" alt="Left: Redis lock then MySQL write can disagree. Right: FOR UPDATE on a replica does not stop a primary write." width="720" height="300" />
  <figcaption>Figure 6. The lock has to sit where the write happens.</figcaption>
</figure>

<figure>
  <img src="/blog/lock-dist-lua.svg" alt="Two API boxes talk to Redis with SET NX and a Lua unlock. Redlock uses three Redis boxes and a majority." width="720" height="300" />
  <figcaption>Figure 7. Two boxes cannot share a mutex in memory. The seat row is still the truth.</figcaption>
</figure>

One Redis can die, and then nobody can take `seat:12A`. Redlock is the same `SET` on several Redis machines. A needs a majority before a deadline, then writes the row. I treat that as harder to lose the key, not as "12A cannot be double-booked." The row decides the winner.

A Redis key is a hint. A lock on a replica is a picture. Two API boxes still have to write 12A on the primary.

I still lock in Redis and skip the version on the row. I still `FOR UPDATE` a replica. I still `SKIP LOCKED` when the passenger asked for 12A.

The same Redis-is-not-the-row mistake showed up when a homepage click found `item:42` missing. I wrote that [when a second GET sat on the pool](/blog/request-hedging-is-a-second-get-not-a-bigger-pool/). The shop that grew more API boxes around that pool is [the proxy post](/blog/the-api-should-see-mysql-not-the-topology/).

If this is useful, wrong, or incomplete, write to me.
