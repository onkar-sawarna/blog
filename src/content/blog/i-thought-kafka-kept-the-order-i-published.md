---
title: "I thought Kafka kept the order I published"
description: "Publish order is not consume order. Kafka is a shared log. The key picks a lane. Extra consumers do not invent more lanes."
pubDate: 2026-08-23
tags: ["systems"]
---

I used to treat Kafka like a durable queue with a nicer name. I borrowed queue rules for order. Publish 1, then 2, then 3, and they come back that way. If you need more throughput, you add consumers.

Who sees the event (one inbox versus two jobs) is a different fight. This one is only the order I thought I had bought.

## The wrong model

A queue hands a message to one worker and the message is gone. Kafka looks close enough that you borrow the word. You say topic when you mean pipe. You say consumer when you mean worker. You assume the cluster remembers the order your API saw.

You publish. Something else polls. The messages come out in the order they went in. If you need more throughput, you add consumers. If that picture were true, most of the surprises would not exist.

<figure>
  <img src="/blog/kafka-pipe.svg" alt="A single queue with messages 1 2 3 4 in one line, next to three lanes where 1 and 3 sit on lane 0 and 2 and 4 sit on lane 2." />
  <figcaption>I called this a queue. It is lanes. 1 can finish after 4.</figcaption>
</figure>

That model is what you get from a diagram with one cylinder and two arrows. Publish on the left. Poll on the right. Nothing in that drawing tells you the cylinder is sliced.

## Where it broke

The useful picture is the cut inside the topic.

Messages in one partition are ordered. Messages in two partitions are not. There is no global order across the topic. If you needed that, you wanted one partition, and you also wanted the throughput of one partition.

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
