---
title: "I thought Kafka kept the order I published"
description: "A topic is a named log sliced into partitions. The API writes the row, then publishes. Order lives on a partition, not on the topic."
pubDate: 2026-08-23
tags: ["systems"]
---

I used to treat Kafka like a durable queue with a nicer name. I borrowed queue rules for order. Publish 1, then 2, then 3, and they come back that way. If you need more throughput, you add consumers.

That only makes sense if you already know what the cluster is. I did not write that down the first time. Here it is, then the order mistake.

## What Kafka is

Kafka is a cluster that keeps events on disk and lets other processes read them later. You append. You poll. The events stay. Retention is time or size, not "someone already took this."

The names that matter:

**Topic.** A named stream. `orders`. `payments`. `listing-events`. That is the inbox you publish to. It is not one pipe. It is a label on a set of logs.

**Partition.** One of those logs. A topic has n partitions. A partition is append-only and ordered. Two partitions have no order between them. If you needed a total order for the whole topic, you wanted n = 1, and you also wanted the throughput of one log.

**Key.** How a message picks a partition. `hash(key) % n`. Same key, same partition, as long as n does not change. No key, and the producer scatters.

**Consumer group.** A job with a cursor on each partition. Search is a group. Counter is a group. Each group sees every message. Two processes in the same group split partitions. They do not both see every message.

That is the architecture. The rest of this post is what people get wrong about order once they have those words.

## A real request

A buyer hits checkout.

The API does two things on purpose, in this order.

First it writes the order row to the database. That is the fact. If this fails, there is no order. The request returns an error.

Then it publishes an event to a Kafka topic, say `orders`. Someone created that topic with n partitions. The API does not invent n on the request. Ops (or you) set n when the topic was born.

The producer, the client library inside the API, picks the partition. You pass a key. The library does `hash(key) % n` and sends the bytes to that partition. The broker appends. It does not re-decide the lane.

I key by `user_id`. Three partitions.

Buyer u1 checks out. `hash(u1) % 3` is 0. The event is the next line on partition 0.

Buyer u2 checks out a second later. `hash(u2) % 3` is 2. That event is the next line on partition 2.

u1 buys again. Same key, same 0. That second event sits after the first one on partition 0. That is the only order this path promised: u1's checkouts, in the order the API published them, on that one log.

The body is small: order id, user, total, created at. The API does not call search. It does not increment a dashboard. It returns 200.

Now the groups.

Search is a consumer group. Counter is a different consumer group. Kafka assigns every partition of `orders` to each group, independently. Both jobs see every event. They do not share a cursor. Search being behind does not stall the counter.

If search has three live members, Kafka gives each member one partition. Member s0 polls only p0. s1 polls p1. s2 polls p2. s0 sees u1. s2 sees u2. Nobody in search sees both unless one member holds two partitions.

Counter does the same math on its own members. c0 also polls p0. c0 also sees u1. s0 did not hand that event to c0. Two processes read the same line on the same log, because they belong to two groups.

If search has one member, that one process polls p0, p1, and p2. It still sees every order. It just does the three lanes itself. If search has four members and three partitions, the fourth member is assigned nothing.

Search writes the order into an index so support can find it. Counter writes "orders today" back to the database. If search is down for an hour, the events are still on the topic. Search catches up from its cursor. The database row was never waiting on search.

<figure>
  <img src="/blog/kafka-job.svg" alt="A user hits the API. The API writes an order row to the database, then publishes to a Kafka topic named orders with three partitions. A search group and a counter group each poll that topic." width="720" height="300" />
  <figcaption>The row is the fact. The topic is the news. Two groups, two cursors, same topic.</figcaption>
</figure>

<figure>
  <img src="/blog/kafka-groups.svg" alt="The API hashes user ids onto partitions. Search members s0 s1 s2 each take one partition. Counter members c0 c1 c2 take the same three partitions again." width="720" height="300" />
  <figcaption>The API hashes the key. Each group covers every partition. Same event, two jobs.</figcaption>
</figure>

If the API only wrote the row and then HTTP-called search and the counter, checkout is coupled to both. One of them slow, and the buyer waits. One of them down, and checkout fails for a side job. Kafka is the buffer. The API is done when the row is in and the event is on the topic.

Three partitions means three parallel logs. That is how the cluster scales, and that is where my order model died.

## What n is for

n = 1 works. A lot of systems should start there.

One partition is one log. Every checkout appends to the same line. Search has one member that actually works. You get a total order on `orders`: u1, then u2, then u1 again, in the order the API published, for the whole topic. The model I wanted is true. The cost is that one disk path and one reader are the ceiling.

I add partitions when that ceiling shows up on a real Friday, not because three looks more serious.

Checkout is 200 orders a second. Search does a fat write per event. One member cannot keep up. Lag grows. Launching four search processes does nothing. They have one partition to share, so three of them sit idle. I need more lanes so more members can work. That is the significance: **n is how many of this job can run at once, and how many appends can land at once.**

n is three promises at once.

**How hard you can write.** The API can append to three logs at the same time. One partition is one disk path. Checkout traffic that all hashes to one user sits on one partition and the other two sit idle. A bad key wastes n.

**How hard one group can read.** Search can use at most three live members that actually work. The unit of parallelism is the partition. You do not get a fourth pair of hands on `orders` until you add a fourth partition.

**Where order lives.** u1's checkouts stay in order only because they share a partition. n is how many independent ordered logs you asked for.

You can raise n on a live topic. The broker will add empty partitions. Old messages stay on 0, 1, and 2. New publishes use `hash(key) % 6`. u1 that always lived on 0 can start landing on 4. Going forward, same key still sticks. The old log and the new log are not one story. Support looking up "everything u1 did" now has to read two lanes.

You cannot lower n. There is no "make `orders` have one partition" on that topic. The messages already exist on three logs. The broker will not glue them back into one. If checkout is quiet and three search members are a waste, you still have three partitions. The extra members sit idle, or one member holds two lanes. n does not shrink to match the traffic.

The way out is a new topic, `orders-v2`, with the n you actually want. You publish new checkouts there. You replay the old topic into it if you need the history. That is a migration. It is not a setting.

So I pick n for the peak I am willing to operate, not for today's lag graph. Too small, and search cannot catch up no matter how many processes I launch. Too big, and I live with empty lanes and a hash I cannot undo.

<figure>
  <img src="/blog/kafka-n.svg" alt="Three partitions on orders. A shrink to n equals 1 is crossed out. Grow keeps old lines. Fewer lanes means a new topic and a replay." width="720" height="280" />
  <figcaption>You can add lanes. You cannot remove them. A smaller n is a new topic.</figcaption>
</figure>

## The wrong model

A queue hands a message to one worker and the message is gone. Kafka looks close enough that you borrow the word. You say topic when you mean pipe. You say consumer when you mean worker. You assume the cluster remembers the order your API saw.

You publish. Something else polls. The messages come out in the order they went in. If you need more throughput, you add consumers. If that picture were true, most of the surprises would not exist.

<figure>
  <img src="/blog/kafka-pipe.svg" alt="A single queue with messages 1 2 3 4 in one line, next to three lanes where 1 and 3 sit on lane 0 and 2 and 4 sit on lane 2." />
  <figcaption>I called this a queue. It is lanes. 1 can finish after 4.</figcaption>
</figure>

That model is what you get from a diagram with one cylinder and two arrows. Publish on the left. Poll on the right. Nothing in that drawing tells you the cylinder is sliced.

## Where it broke

The useful picture is that cut, used as a promise.

Messages in one partition are ordered. Messages in two partitions are not. There is no global order across `orders`. If you needed that, you wanted one partition, and you also wanted the throughput of one log.

A message lands in a partition by key. You hash the key and take it modulo n. Same key, same lane, every time, as long as n does not change.

Take three partitions and a key that is a user id.

- u1 hashes to 0
- u2 hashes to 2
- u3 hashes to 0
- u4 hashes to 2
- u1 again hashes to 0

u1's events stay in order with each other. u1 and u2 can finish in any wall-clock order. The cluster did not break. You asked it for per-user order and it gave you that. You did not ask for a total order of the topic.

<figure>
  <img src="/blog/kafka-key.svg" alt="An API publishes into a hash funnel. u1 and u3 land on lane 0, including a later u1. u2 and u4 land on lane 2. Lane 1 is empty." />
  <figcaption>u1 goes to lane 0. Later u1 goes to lane 0 again. That is the only order you were promised.</figcaption>
</figure>

Leave the key empty and the producer picks a partition. Then even one user's events can split.

The queue picture dies again on the consumer side.

In one consumer group, a partition is assigned to at most one live member. The unit of parallelism is the partition, not the process you launched. Three partitions, three members, each member has a lane. Three partitions, two members, one member holds two lanes. Three partitions, four members, one member sits idle and receives nothing.

<figure>
  <img src="/blog/kafka-consumers.svg" alt="Three lanes feed c1, c2, and c3. A dashed box for c4 sits aside and polls nothing." />
  <figcaption>You added a consumer. Kafka did not add a lane. The fourth process is unemployed.</figcaption>
</figure>

If you have fewer consumers than partitions, one process reads more than one lane. Those lanes still have no order between them. So even a single process does not give you global order. It gives you two (or n) ordered logs that you interleave however you poll.

## The model that stuck

I keep three facts, in this order.

**1. The partition is the log.** Order, replay, and "who is reading this" all attach here. The topic is how you name a set of those logs. n is how many of those logs you have. You can grow it. You cannot shrink it.

**2. The key is which order you care about.** User, order id, host, whatever must stay in sequence. No key means you accepted scatter.

**3. A consumer group is a cursor per partition, shared by members.** More groups means more independent readers of the same log. More members in one group means more parallelism, up to n, and then you are paying for idle processes.

I still draw the API on the left and two services on the right. I label the middle as n lanes, not as a pipe.

## How I recognize the old model now

I am in the old model when:

- I say "Kafka will keep the order" and I have not named a key or n.
- I add a consumer because a lag graph is red, and I have not counted partitions.
- I treat two consumer groups as two workers fighting over one SQS queue.
- I plan to "just reduce partitions" after a bad launch.

The replacement habit is one sentence: which key, how many lanes, which group. If I cannot say those, I am guessing.

## What I would still get wrong

A key that is too coarse. Everything hashes to a few users and three partitions sit idle while one burns.

A key that is too fine. You wanted per-user order and you keyed on request id. The lane is random again.

Growing n to fix lag and then wondering why a user's history split.

Treating "the fourth consumer is idle" as a broker bug. It is the assignment rule doing what it says.

I am not done being wrong about logs that look like queues. I am just done asking a topic for a total order it never had.

If this is useful, wrong, or incomplete, write to me.
