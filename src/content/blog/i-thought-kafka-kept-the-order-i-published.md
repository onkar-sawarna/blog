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

Then it publishes an event to a Kafka topic, say `orders`, keyed by `user_id` or `order_id`. The body is small: order id, user, total, created at. The API does not call search. It does not increment a dashboard. It returns 200.

A search group polls `orders` and writes the order into an index so support can find it. A counter group polls the same topic and writes "orders today" back to the database. They do not call each other. They do not steal. Each group has its own cursor. If search is down for an hour, the events are still on the topic. Search catches up. The database row was never waiting on search.

<figure>
  <img src="/blog/kafka-job.svg" alt="A user hits the API. The API writes an order row to the database, then publishes to a Kafka topic named orders with three partitions. A search group and a counter group each poll that topic." width="720" height="300" />
  <figcaption>The row is the fact. The topic is the news. Two groups, two cursors, same topic.</figcaption>
</figure>

If the API only wrote the row and then HTTP-called search and the counter, checkout is coupled to both. One of them slow, and the buyer waits. One of them down, and checkout fails for a side job. Kafka is the buffer. The API is done when the row is in and the event is on the topic.

The topic `orders` is still sliced. Three partitions means three parallel logs. That is how the cluster scales, and that is where my order model died.

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

n is not a knob you twist for free.

If you increase the partition count, the old messages stay where they were written. New messages hash into the new n. u1 that always lived on 0 can start landing on 4. Per-key order still holds going forward. It does not stitch the old log and the new log into one story.

You also cannot shrink n on a topic. The way out of a bad n is a new topic with the count you actually want, then you process from the old one into the new one. That is a migration. It is not a setting.

## The model that stuck

I keep three facts, in this order.

**1. The partition is the log.** Order, replay, and "who is reading this" all attach here. The topic is how you name a set of those logs.

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
