---
title: "Redis can hold a key. MySQL holds the seat."
description: "Two people tap Book on the same seat. Lock the row, check a version, skip a locked seat, or take a lock in another box. The seat row is still the truth."
pubDate: 2026-09-14
tags: ["systems"]
draft: false
---

Passenger A taps Book on seat 12A. Passenger B taps Book on the same seat. Two requests hit my API. The row still says empty.

I used to think "a lock" was one thing: grab 12A, write booked, let go. That is one kind. Airlines, check-in desks, and two API boxes need the others.

<figure>
  <img src="/blog/lock-two-taps.svg" alt="Passenger A and passenger B both tap Book on seat 12A. Two API requests hit the same seats row." width="720" height="260" />
  <figcaption>Figure 1. One seat. Two clicks. The lock is the argument.</figcaption>
</figure>

## Lock the seat first

Pessimistic locking means I assume a fight. A starts a transaction, locks the row for 12A, sees it empty, writes booked, commits. The lock drops. B has been waiting. B now reads booked and gets "seat taken."

The name is gloomy on purpose. I pay the wait so I do not double-book.

Optimistic locking means contention is rare. I assume we usually do not fight. A and B both read 12A empty. The row has a version, say 3. A writes booked and sets version to 4 where version is still 3. That update changes one row. B runs the same update. Zero rows change. B retries or tells the passenger the seat is gone.

No one waited. The second writer lost at the update, not at the lock.

<figure>
  <img src="/blog/lock-pess-opt.svg" alt="Left: A locks 12A, writes, unlocks. B waits then sees booked. Right: both read version 3. A writes version 4. B update hits zero rows." width="720" height="300" />
  <figcaption>Figure 2. Same seat. Lock before, or check after.</figcaption>
</figure>

The cabin is fixed inventory: 120 seats, not 120 more when people show up. Two taps on 12A, or 120 desks on the next empty row, is contention. Fixed inventory plus contention is when I lock. I pick pessimistic when that fight is the normal path, like a cheap fare that just opened.

Optimistic is the other side: not much contention. A Tuesday night on a half-empty flight. Collisions are rare, so I check a version at write time and skip the wait list. I also pick it when I do not want B sitting on a lock while A talks to a payment service.

## Someone has to keep the wait list

When B waits, it is not magic. Inside the database there is a lock manager. It is a table of grants: which row, which mode, who holds it, who is waiting. A asks for an exclusive lock on 12A. The manager writes "holder: A." B asks for the same row. The manager puts B on the wait list. A commits. The manager wakes B.

The lock is not the seat. The seat is the row. The lock is that entry. Optimistic booking often never talks to this table on the read. It only finds out at update time that version 3 is gone.

Shared and exclusive are two modes on that same table. Two check-in screens can take a shared lock to print that 12A is still free. `FOR SHARE` is that mode. A booking needs exclusive: `FOR UPDATE`. The manager will not give exclusive while a share is out, and it will not give a new share while exclusive is out. Read together. Write alone.

That is the first difference between kinds. Pessimistic uses the manager. Optimistic uses a version on the row. Shared is "we may all look." Exclusive is "only I may book."

<figure>
  <img src="/blog/lock-manager.svg" alt="Transactions A and B talk to a lock manager. Row 12A is exclusive, holder A, waiter B. The seat row is separate." width="720" height="280" />
  <figcaption>Figure 3. The lock is not the seat. It is an entry in that table.</figcaption>
</figure>

<figure>
  <img src="/blog/lock-kinds.svg" alt="Six kinds: shared, exclusive, optimistic, row lock, remote lock, distributed lock. Who waits, where the lock lives, what decides the seat." width="720" height="320" />
  <figcaption>Figure 4. Who waits, where the lock lives, what decides the seat.</figcaption>
</figure>

The rest of the differences are about where that entry lives and what B does instead of waiting. A row lock lives in the lock manager on the primary, next to the write. A remote lock lives in Redis. A distributed lock is a remote lock that two API boxes can both see. `NOWAIT` is the manager saying busy, with no wait list. `SKIP LOCKED` is the manager skipping that row. Redlock is the same remote idea on several Redis machines.

## Two seats, a cycle

A is booking a pair: 12A and 12B. A locks 12A. B is booking the same pair from the other side. B locks 12B. A wants 12B. B wants 12A. Each holds what the other needs. That cycle is a deadlock. Nobody can finish.

MySQL is aggressive here. It looks for that cycle soon. If an edge in the wait graph will close a loop, it kills one transaction and lets the other go. A gets a deadlock error. B books both seats. A retries.

Postgres is lazier. It lets them wait. After a timeout it builds a wait-for graph: who is waiting on whom. If it finds a cycle, it kills one. If it does not, they keep waiting. The default timeout is about a second. People talk about it as "every few seconds" because that walk is not free, so the database does not do it on every lock.

Same cycle. MySQL fails fast. Postgres waits, then looks.

<figure>
  <img src="/blog/lock-deadlock.svg" alt="A holds 12A and wants 12B. B holds 12B and wants 12A. MySQL kills one now. Postgres waits then walks the wait-for graph." width="720" height="280" />
  <figcaption>Figure 5. A cycle is a deadlock. Someone has to lose.</figcaption>
</figure>

I still get this wrong when I lock seats in a random order. If every transaction locks 12A then 12B, the cycle does not form. The kill is a safety net, not a booking strategy. The wait-for graph is the lock manager's list of waiters drawn as arrows. MySQL walks it early. Postgres walks it after a timeout.

## Wait, fail, or take the next seat

`FOR UPDATE` is the exclusive row lock. A has 12A. B's `SELECT ... FOR UPDATE` on 12A waits until A commits. That is the check-in path when both passengers want that exact seat.

`NOWAIT` means B does not wait. If 12A is locked, the database returns an error now. The API can say "someone is holding this seat" and show the map again. Useful when waiting would freeze the page.

`SKIP LOCKED` means B does not wait and does not error. It skips 12A and takes the next free row, 12B. That is a check-in desk handing out any seat in a fare class, or a worker taking the next job in a queue. Two desks should not wait on the same row. They should skip it.

<figure>
  <img src="/blog/lock-skip-nowait.svg" alt="FOR UPDATE waits on 12A. NOWAIT errors. SKIP LOCKED takes 12B." width="720" height="280" />
  <figcaption>Figure 6. Wait, fail, or skip. Same lock, three answers from the lock manager.</figcaption>
</figure>

`SKIP LOCKED` is not "ignore the booking." It is "this row is busy, give me another." If B must have 12A, do not skip.

## A full cabin at the desk

Check-in is not two people on 12A. It is a plane: trip 1, 120 empty seats, 120 passengers. `user_id IS NULL` means nobody has that seat yet. Each desk takes the next free row, then writes the passenger.

The 120 requests are not one snapshot. A few desks finish a read and a write while the rest are still in flight.

No lock. Most desks still see the same first empty row and write it. A few start late enough that 1A already has a `user_id`, so they land on 1B, then 1C. Three or four seats get a write. Those rows are overbooked. The rest of the cabin stays empty. Passengers think they boarded.

```
SELECT * FROM seats
WHERE trip_id = 1 AND user_id IS NULL
ORDER BY id
LIMIT 1;

UPDATE seats
SET user_id = 42
WHERE id = ?;
```

`FOR UPDATE`. The first desk locks 1A. The others wait on 1A, because that is the first empty row the scan sees. The first desk commits. Waiters wake and pile onto 1B. Then 1C. One hundred twenty unique seats. One at a time. A convoy on the lock manager.

```
SELECT * FROM seats
WHERE trip_id = 1 AND user_id IS NULL
ORDER BY id
LIMIT 1
FOR UPDATE;

UPDATE seats
SET user_id = 42
WHERE id = ?;
```

`NOWAIT`. A few desks are ahead. They lock 1A, 1B, 1C, maybe 1D. Everyone else hits a locked first-empty row and gets an error now. No skip. Three or four seats booked. The rest of the cabin sits empty while the page says busy. Retry in the same burst hits the same pile.

```
SELECT * FROM seats
WHERE trip_id = 1 AND user_id IS NULL
ORDER BY id
LIMIT 1
FOR UPDATE NOWAIT;

UPDATE seats
SET user_id = 42
WHERE id = ?;
```

`SKIP LOCKED`. The first desk locks 1A. The next desk skips 1A and locks 1B. The next takes 1C. One hundred twenty desks, one hundred twenty seats, in parallel. A 121st passenger gets no row, not a wait.

```
SELECT * FROM seats
WHERE trip_id = 1 AND user_id IS NULL
ORDER BY id
LIMIT 1
FOR UPDATE SKIP LOCKED;

UPDATE seats
SET user_id = 42
WHERE id = ?;
```

<figure>
  <img src="/blog/lock-120-checkin.svg" alt="No lock and NOWAIT book a handful of seats. FOR UPDATE queues. SKIP LOCKED assigns 120 seats in parallel." width="720" height="320" />
  <figcaption>Figure 7. Next empty seat. 120 desks. Four answers.</figcaption>
</figure>

<div class="compare-wrap">
<table>
  <thead>
    <tr>
      <th scope="col"></th>
      <th scope="col">What happens</th>
      <th scope="col">Seats filled</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <th scope="row">No lock</th>
      <td class="is-low">Most write the same first empty row. A few late reads move on.</td>
      <td>Three or four seats written. Those rows overbooked. Cabin mostly empty.</td>
    </tr>
    <tr>
      <th scope="row"><code>FOR UPDATE</code></th>
      <td>Wait on the first empty row, then the next.</td>
      <td>120 unique. Serial. Slow.</td>
    </tr>
    <tr>
      <th scope="row"><code>NOWAIT</code></th>
      <td class="is-low">A few lock 1A, 1B, 1C. The rest error. No skip.</td>
      <td>Three or four seats. Error storm. Cabin mostly empty.</td>
    </tr>
    <tr>
      <th scope="row"><code>SKIP LOCKED</code></th>
      <td>Skip the busy row. Take the next free seat.</td>
      <td>120 unique. Parallel.</td>
    </tr>
  </tbody>
</table>
</div>

That is the desk path: any seat in the cabin. If the passenger asked for 12A, do not skip. Wait or fail on that row.

The pile does not grow. The fight is real. That is why the `SELECT` takes a lock instead of hoping the `UPDATE` is enough.

Same two ingredients show up elsewhere. CoWIN: a slot on a day at a center, not more doses because the page is busy. IRCTC: a berth on a train. BookMyShow: a seat in a hall. A flash sale: N units of one SKU. Fixed pile, everyone taps at once. Lock the row, or skip to the next free one. Do not hope the write is enough.

## A lock in Redis is not the seat

A remote lock lives on a central machine, not in the API process. Two workers do not share memory. They ask one lock manager. Redis is the usual choice.

Two consumers pulling booking jobs look like this:

```
acq_lock()
read_msg()
rel_lock()
```

Worker A calls `acq_lock()` on `seat:12A`. Redis grants it. A reads the job, writes the seat, calls `rel_lock()`. Worker B's `acq_lock()` waits or fails until A lets go. Without that, both read the same job and both book 12A.

The call is a `SET` on Redis, then a write on MySQL. The idea is to keep the fight off the database.

That lock and that write are two systems. The Redis key can expire while the MySQL write is still running. A second worker can take the lock and write again. Or the write fails and the lock remains, and 12A looks taken when the row is empty.

A remote lock is a hint plus a timeout. The seat row is still the truth. I still do the version check or the `FOR UPDATE` on the primary when I write booked.

Worse: I take `FOR UPDATE` on a read replica. The replica is a copy. The write goes to the primary. I locked a picture of 12A. The primary never heard. Two bookings land. The lock has to sit where the write happens.

<figure>
  <img src="/blog/lock-remote-replica.svg" alt="Left: Redis lock then MySQL write can disagree. Right: FOR UPDATE on a replica does not stop a primary write." width="720" height="300" />
  <figcaption>Figure 8. The lock has to sit where the write happens.</figcaption>
</figure>

## Two API boxes, one Redis, a Lua script

Pessimistic locking inside one process is a mutex in memory. Two API boxes do not share that memory. They share a key on the central Redis. That is still one lock manager.

The usual Redis move is `SET seat:12A <token> NX EX 30`. NX means set only if the key is missing. EX 30 means the key dies in thirty seconds if the box crashes. A gets the key. B gets nil and does not book.

Unlock is not `DEL seat:12A`. If A's lock expired and C now holds 12A, A's late delete would steal C's lock. A Lua script runs on Redis as one step: read the value, delete only if it is still A's token. Redis runs that script without interleaving another command.

```
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
end
return 0
```

That is compare-and-delete. It is still not a seat assignment. After A holds the key, A writes the row. If the write fails, A unlocks. If A dies, the expiry frees the key.

<figure>
  <img src="/blog/lock-dist-lua.svg" alt="Two API boxes talk to Redis with SET NX and a Lua unlock. Redlock uses three Redis boxes and a majority." width="720" height="300" />
  <figcaption>Figure 9. Two boxes cannot share a mutex in memory. The seat row is still the source of truth.</figcaption>
</figure>

## A majority, then the work

One Redis can die. The whole lock manager is gone. Redlock is a distributed lock on Redis: the same `acq_lock` on several independent Redis machines, not one.

A tries `SET seat:12A <token> NX EX` on each box, one after another, with a short timeout. If A gets a majority before a deadline, A holds the lock and does the work. Then A `rel_lock`s every box, with the same token check in Lua. B needs that majority too. One Redis restart does not hand 12A to B.

```
n = 0
for each redis:
  if SET seat:12A token NX EX 30: n = n + 1
if n > N/2 and time left: work()
for each redis: rel_lock()
```

I treat Redlock as "harder to lose the key when one Redis restarts," not as "12A cannot be double-booked." Clocks and pauses can still lie. A process can sleep past the expiry, wake up, and think it holds the lock. I still write the seat with a version or `FOR UPDATE` on the primary. Redlock reduces the pile-up. The row decides the winner.

## Remote, replica, or two boxes

Same two taps on 12A. The cheap remote lock dies with Redis. The replica lock is the wrong copy. Redlock is the distributed lock on Redis. It pays in QPS.

<div class="compare-wrap">
<table>
  <thead>
    <tr>
      <th scope="col"></th>
      <th scope="col">Correctness</th>
      <th scope="col">Throughput</th>
      <th scope="col">Availability</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <th scope="row">Remote lock</th>
      <td>Ok. Hint plus timeout. Still write the row.</td>
      <td>High. Fight sits on Redis, not the primary.</td>
      <td class="is-low">Low. One Redis dies, I cannot take the lock.</td>
    </tr>
    <tr>
      <th scope="row">Remote lock with a read replica</th>
      <td class="is-low">Low. Locked a copy. Primary never heard. Double-book.</td>
      <td>High. Reads leave the primary.</td>
      <td>Ok. Replica can stay up. The booking is still wrong.</td>
    </tr>
    <tr>
      <th scope="row">Redlock</th>
      <td>Ok. Majority of Redis boxes. Still not the seat.</td>
      <td class="is-low">Low. One holder per seat. Several Redis calls.</td>
      <td>High. Majority can survive one Redis dying.</td>
    </tr>
  </tbody>
</table>
</div>

None of the three replace `FOR UPDATE` or a version check on the primary when I write booked.

## What I took from this

Two passengers on 12A is one story. The lock manager is the wait list. The kinds differ on who waits, where the lock lives, and what decides the seat.

Fixed inventory plus contention: lock. A vaccine slot, a berth, a cinema seat, a flash-sale SKU. Same pile, same fight as 12A. Optimistic locking when contention is rare: check a version, do not take the wait list. Shared for two desks reading. Exclusive for one booking. MySQL will kill a deadlock quickly. Postgres will wait, then walk the wait-for graph. One hundred twenty desks on the next empty seat: no lock and `NOWAIT` fill three or four seats, `FOR UPDATE` makes a convoy, `SKIP LOCKED` fills the plane. If they asked for 12A, wait or fail on that row.

A Redis lock is remote: one central lock manager, `acq_lock`, work, `rel_lock`. A lock on a replica is on the wrong copy. Two API boxes on one Redis still share that one manager: a token, an expiry, and a Lua unlock so they do not delete someone else's key. Redlock is the distributed lock on Redis. Same key, several machines, a majority. Correctness, throughput, and availability trade in that table. None of them replace the write to seat 12A.

I still get this wrong. I lock in Redis and skip the version on the row. I `FOR UPDATE` a replica. I `DEL` a lock without checking the token. I `SKIP LOCKED` when the passenger asked for 12A, not "any window." I treat a deadlock error as a bug instead of "retry, and lock seats in one order."
