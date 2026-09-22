---
title: "SNS-SQS and Kafka, are they the same?"
description: "Support could find the order. The dashboard said zero orders today. Both jobs were reading the same checkout, and one queue quietly decided only one of them would get it."
pubDate: 2026-08-28
tags: ["systems"]
---

Support could look up the order. The dashboard said zero orders today. Both jobs were reading the same checkout, and I had written both of them myself.

## A buyer taps Buy

The buyer checks out. The API does two things, in this order.

First it writes the order row, `o1`, to the database. That row is the fact. If that write fails there is no order.

Then it announces what happened: a small event with the order id, the user, and the total. The API does not call search. It does not call the dashboard. It returns success.

Two other jobs care. Search wants `o1` in its index so support can find the order. A counter wants to add one to the orders-today total.

<figure>
  <img src="/blog/checkout-o1.svg" alt="Buyer hits the API. The API writes row o1, publishes event o1, and returns 200. Search and the counter are not on that path." width="720" height="280" />
  <figcaption>Figure 1. The row is the fact. The event is the news. Search and count are later.</figcaption>
</figure>

## The dashboard that stayed at zero

The thing in the middle was a single queue. Both jobs read from it. In my head two consumers meant both would see `o1`.

Only one of them did.

<figure>
  <object class="figure-svg" data="/blog/two-consumers.svg" type="image/svg+xml" width="720" height="300" style="aspect-ratio: 720 / 300" aria-label="Left: I said two consumers, so search and the counter both see o1. Right: the queue gave o1 to search. The counter got nothing.">
    <img src="/blog/two-consumers.svg" alt="Left: I said two consumers, so search and the counter both see o1. Right: the queue gave o1 to search. The counter got nothing." width="720" height="300" />
  </object>
  <figcaption>Figure 2. Two consumers is a sentence. The queue only heard one job.</figcaption>
</figure>

A queue of this kind is a work inbox. A worker asks for work (polling). The queue hands it a message and hides that message from everybody else for a while. The worker does the job and then deletes the message. Then it is gone. If the worker crashes before deleting, the message reappears after a timeout. That reappearance is why delete is a separate step: it is how the queue knows the work finished, not just started.

Search polled first and was handed `o1`. It wrote the index. It deleted the message. The counter polled a moment later and found an empty queue. Support could find the order. The dashboard stayed at zero.

<figure>
  <img src="/blog/dashboard-zero.svg" alt="Support search finds order o1. The orders-today dashboard still shows 0." width="720" height="240" />
  <figcaption>Figure 3. Same checkout. One job ran. One did not.</figcaption>
</figure>

Nothing malfunctioned. The queue behaved as documented. My mistake was what I thought a consumer was.

## What a queue is actually for

Two search processes on that queue would have been fine. They race for whatever arrives next. Each order gets indexed once. That is what a work queue is built for: one job, more hands, and the queue making sure two hands do not do the same task twice.

Search and a counter are not that. They are two different jobs that both need the same event. The queue hands each message to one worker.

The workaround I tried first was to run both functions inside one process: poll once, then index and count. That works until there are two replicas of that process, or until one crashes halfway and the message comes back. Then the counter increments twice. The bug moved.

Leaving the message in the queue does not help either. Not deleting means "this job is not finished yet." It does not mean the counter can come back tomorrow and read the same checkout. A queue holds pending work, not history.

<figure>
  <img src="/blog/kafka-sqs.svg" alt="API publishes to one SQS queue. Worker A got event 1 and deleted it. Worker B never saw it. On the Kafka side, a search group and a counter group both read event 1." width="720" height="300" />
  <figcaption>Figure 4. One queue, two tasks, one of them loses the event.</figcaption>
</figure>

## Giving each job its own inbox

The first real fix is to stop sharing the queue.

SNS is a bus, not an inbox. The API publishes `o1` to a topic, once. SNS copies that event into every queue subscribed to the topic. Search gets its own queue with its own copy. The counter gets its own queue with its own copy.

Now both jobs work. Search polls, indexes `o1`, deletes its copy. The counter polls, writes orders-today as 1, deletes its copy. Neither deletion is visible to the other.

Adding a third job later means another subscription and another queue. Fan-out is done by duplication. The number of copies grows with the number of jobs.

<figure>
  <img src="/blog/sns-sqs-kafka.svg" alt="Left: API to SNS, then a copy into an SQS search queue and an SQS count queue. Right: API to one Kafka log, search and counter each keep a cursor." width="720" height="300" />
  <figcaption>Figure 5. Same publish. Same two jobs. SNS copies. Kafka lets both read.</figcaption>
</figure>

## Keeping one copy and two bookmarks

The other fix keeps a single copy of the event and changes who is allowed to read it.

The API publishes `o1` to a Kafka topic called `orders`. Kafka writes the event to a file and leaves it there. Reading does not remove it. Search is one consumer group and the counter is a different group. A group is one job, which may be several processes, and each group keeps its own bookmark. That bookmark is a cursor.

Both groups read `o1`, because reading is only moving your own bookmark. The event is still in the file afterwards, available to whatever job gets written next month.

If search is down for an hour, checkout still returns success. The row is in the database and the event is on the topic. When search comes back, it reads on from its cursor. The counter never waited for it.

<figure>
  <object class="figure-svg" data="/blog/kafka-search-down.svg" type="image/svg+xml" width="720" height="300" style="aspect-ratio: 720 / 300" aria-label="At t=0 checkout returns 200 and search is down. In the same hour the counter reads o1. After an hour search is back and indexes o1 from its cursor.">
    <img src="/blog/kafka-search-down.svg" alt="At t=0 checkout returns 200 and search is down. In the same hour the counter reads o1. After an hour search is back and indexes o1 from its cursor." width="720" height="300" />
  </object>
  <figcaption>Figure 6. Checkout did not wait on search. The log held o1.</figcaption>
</figure>

How long `o1` survives is set by time or disk space, not by whether anybody has read it. Inside a group, several processes still split the work: only one of them indexes this particular order. The group is the job. A process inside it is another pair of hands.

Kafka also slices a topic into partitions. That is how it scales and where order actually lives. I wrote that [separately](/blog/i-thought-kafka-kept-the-order-i-published/). None of it changes the choice here.

<figure>
  <img src="/blog/kafka-job.svg" alt="A user hits the API. The API writes an order row, then publishes. A search job and a counter job each poll the same log with their own cursor." width="720" height="300" />
  <figcaption>Figure 7. One publish. Two jobs. Same checkout, twice. The log stays.</figcaption>
</figure>

## Which one I reach for

<figure>
  <img src="/blog/which-inbox.svg" alt="Three paths for o1: one SQS with one winner, SNS copying into two queues, Kafka with one log and two cursors." width="720" height="300" />
  <figcaption>Figure 8. Same o1. The inbox is what changes.</figcaption>
</figure>

A queue on its own when `o1` is a task and exactly one worker should perform it. Extra workers just share the load.

SNS with a queue per job when queues are already the shape of the system and a second job needs the same events. One publish, two copies, two deletions.

Kafka when several jobs will keep reading the same checkouts, including jobs that do not exist yet. One log and a cursor per job. A new job is a new group, and replaying history is just moving a cursor back.

An inbox belongs to a job, not to a repository. Search and the counter shared a queue because they shared a codebase. The queue had no idea it was picking a winner between two jobs.

A filter on an SNS subscription that drops `o1` from the counter's queue leaves search healthy and the dashboard at zero. Same bug from the outside.

<figure>
  <img src="/blog/sns-filter-drop.svg" alt="API publishes to SNS. A filter drops o1 from the count queue. Search still gets a copy. The dashboard stays 0." width="720" height="260" />
  <figcaption>Figure 9. Fan-out with a filter is still a delete for one job.</figcaption>
</figure>

Giving two Kafka services the same group id does it too. Same group means one job, so they divide the partitions and each sees roughly half the events. That is the single-queue bug rebuilt on a log.

<figure>
  <img src="/blog/shared-group-id.svg" alt="Topic orders holds o1. Search and the counter share group id search. Search got o1. The counter is split away." width="720" height="260" />
  <figcaption>Figure 10. Same group id. Same steal as one queue.</figcaption>
</figure>

"The log keeps everything" only holds until retention expires. Then `o1` is gone for every job at once.

I am done putting search and a counter on one queue and calling it fan-out.

If this is useful, wrong, or incomplete, write to me.
