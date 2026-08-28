---
title: "SNS-SQS vs Kafka"
description: "Same checkout, three brokers. One Simple Queue Service (SQS) queue and search steals the order from the counter. Simple Notification Service (SNS) copies into two queues. Kafka keeps one log and two cursors."
pubDate: 2026-08-28
tags: ["systems"]
---

## Checkout

A buyer hits checkout.

The API writes order `o1` to the database. That row is the fact. Then it publishes a small event: order id, user, total. It does not call search. It does not increment a dashboard. It returns 200.

Search should write `o1` into an index so support can find it. The counter should write "orders today" back to the database. Those are two jobs. They happen to live in one repo.

<figure>
  <img src="/blog/checkout-o1.svg" alt="Buyer hits the API. The API writes row o1, publishes event o1, and returns 200. Search and the counter are not on that path." width="720" height="280" />
  <figcaption>The row is the fact. The event is the news. Search and count are later.</figcaption>
</figure>

## One SQS

The broker is one Simple Queue Service (SQS) queue. Both jobs poll it.

I called that "two consumers." I thought both would see `o1`. They will not.

<figure>
  <img src="/blog/two-consumers.svg" alt="Left: I said two consumers, so search and the counter both see o1. Right: the queue gave o1 to search. The counter got nothing." width="720" height="280" />
  <figcaption>Two consumers is a sentence. The queue only heard one job.</figcaption>
</figure>

SQS is a job inbox. A worker asks for work. It gets one message. The other worker cannot see it. When the work is done, the worker deletes the message. The job is gone. If the worker dies first, the message comes back after a timeout.

Search polls first. It gets `o1`. It writes the index. It deletes the message.

The counter polls next. The queue is empty. "Orders today" stays 0. Support can find the order. The dashboard cannot.

<figure>
  <img src="/blog/dashboard-zero.svg" alt="Support search finds order o1. The orders-today dashboard still shows 0." width="720" height="240" />
  <figcaption>Same checkout. One job ran. One did not.</figcaption>
</figure>

Two search processes on that queue would have been fine. They race for the next order. Only one of them should index `o1`. Search and a count are not that. The queue does not care that they share a repo. It gives `o1` to one worker.

Or both jobs run in one process. One poll, two functions. Both think they own `o1`. Then a second replica retries and the counter increments again. Now the dashboard says 2.

SQS can keep `o1` if nobody deletes it. That is still not a log. It means "this job is not done yet." It does not mean the counter can come back later and read the same checkout.

<figure>
  <img src="/blog/kafka-sqs.svg" alt="API publishes to one SQS queue. Worker A got event 1 and deleted it. Worker B never saw it. On the Kafka side, a search group and a counter group both read event 1." width="720" height="300" />
  <figcaption>One queue, two tasks, one of them loses the event.</figcaption>
</figure>

## SNS plus two queues

Need search and the counter to both see `o1`? Do not share that queue.

Fan out. Simple Notification Service (SNS) is the bus. The API still publishes once. It publishes to SNS, not to a queue. SNS copies `o1` into two SQS queues. Search has its own inbox. The counter has its own inbox.

Search polls the search queue. It indexes `o1`. It deletes its copy. The counter polls the count queue. It writes "orders today = 1." It deletes its copy. Same repo is fine. The inbox is not shared.

A third job, say email, needs another subscription and another queue. You pay for the extra copy. You run the extra inbox.

<figure>
  <img src="/blog/sns-sqs-kafka.svg" alt="Left: API to SNS, then a copy into an SQS search queue and an SQS count queue. Right: API to one Kafka log, search and counter each keep a cursor." width="720" height="300" />
  <figcaption>Same publish. Same two jobs. SNS copies. Kafka lets both read.</figcaption>
</figure>

## Kafka

Now the API publishes `o1` to a Kafka topic, say `orders`. Kafka keeps the event on disk. Search is a consumer group. The counter is a different consumer group. Each group has its own cursor. Both read `o1`. Nobody deletes it for the other job.

Search writes the index. The counter writes "orders today = 1." The line is still on the topic.

Search is down for an hour. Checkout still returns 200. The row is in the database. The event is on `orders`. Search comes back, reads from its cursor, and indexes `o1`. The counter did not wait.

<figure>
  <img src="/blog/kafka-search-down.svg" alt="At t=0 checkout returns 200 and search is down. In the same hour the counter reads o1. After an hour search is back and indexes o1 from its cursor." width="720" height="300" />
  <figcaption>Checkout did not wait on search. The log held o1.</figcaption>
</figure>

How long `o1` stays is time or size, not "someone already took this." Partitions are how that topic scales and how it orders. I wrote that fight in [another post](/blog/i-thought-kafka-kept-the-order-i-published/). You do not need the cut to pick the tool here.

Two workers in the search group still split the work, like two SQS workers on the search-only queue. Only one of them indexes this order. The group is the job. A member of the group is another pair of hands on that job.

<figure>
  <img src="/blog/kafka-job.svg" alt="A user hits the API. The API writes an order row, then publishes. A search job and a counter job each poll the same log with their own cursor." width="720" height="280" />
  <figcaption>One publish. Two jobs. Same checkout, twice. The log stays.</figcaption>
</figure>

## Which one

<figure>
  <img src="/blog/which-inbox.svg" alt="Three paths for o1: one SQS with one winner, SNS copying into two queues, Kafka with one log and two cursors." width="720" height="300" />
  <figcaption>Same o1. The inbox is what changes.</figcaption>
</figure>

I pick SQS when `o1` is a job and only one worker should run it. Index this order once. Extra search workers share the work.

I pick SNS plus SQS when I already use queues and I need the counter too. One publish, two copies, two deletes. A third job is another copy.

I pick Kafka when search, the counter, and the next job will keep reading the same checkouts, including later. One log, a cursor per job. A third job is another group, not another queue. Replay is "move the search cursor back to `o1`."

The inbox belongs to a job, not to a repo. I am in the old model when search and the counter poll one SQS queue because they share a repo, or when I put the counter in the search Kafka group and the dashboard stays 0.

I still get the next bits wrong.

An SNS filter that drops `o1` from the count queue: search is fine, dashboard is 0.

<figure>
  <img src="/blog/sns-filter-drop.svg" alt="API publishes to SNS. A filter drops o1 from the count queue. Search still gets a copy. The dashboard stays 0." width="720" height="260" />
  <figcaption>Fan-out with a filter is still a delete for one job.</figcaption>
</figure>

The same Kafka group id on both services: one of them steals `o1`, just like SQS.

<figure>
  <img src="/blog/shared-group-id.svg" alt="Topic orders holds o1. Search and the counter share group id search. Search got o1. The counter is split away." width="720" height="260" />
  <figcaption>Same group id. Same steal as one SQS queue.</figcaption>
</figure>

Thinking "the log stays" means "I do not need backups": retention ends, then `o1` is gone for both.

I am just done putting search and a count on one queue and calling it fan-out.

If this is useful, wrong, or incomplete, write to me.
