---
title: "SNS-SQS and Kafka, are they the same?"
description: "Support could find the order. The dashboard said zero orders today. Both jobs were reading the same checkout, and one queue quietly decided only one of them would get it."
pubDate: 2026-08-28
tags: ["systems"]
---

Support could look up the order. The dashboard said zero orders today. Both of those were reading the same checkout, and I had written both of them myself, in the same repository, on the same afternoon.

## A buyer taps Buy

The buyer checks out and the API does two things in a deliberate order.

First it writes the order row, call it `o1`, to the database. That row is the fact. If that write fails there is no order and the buyer gets an error.

Then it announces what happened, by publishing a small event carrying the order id, the user, and the total. Announcing is not asking. The API does not call the search service and it does not call the dashboard. It returns a success response and the buyer sees a confirmation page.

Two other jobs care about that announcement. Search wants `o1` written into its index so support can find the order later. A counter wants to add one to the orders-today total in the database. Two jobs, entirely separate concerns, which happened to live in the same repository and get deployed together.

<figure>
  <img src="/blog/checkout-o1.svg" alt="Buyer hits the API. The API writes row o1, publishes event o1, and returns 200. Search and the counter are not on that path." width="720" height="280" />
  <figcaption>The row is the fact. The event is the news. Search and count are later.</figcaption>
</figure>

## The dashboard that stayed at zero

The thing in the middle was a single queue: SQS, the Simple Queue Service. Both jobs read from it. In my head that made them two consumers of the event, and two consumers meant both would see `o1`.

Only one of them did.

<figure>
  <img src="/blog/two-consumers.svg" alt="Left: I said two consumers, so search and the counter both see o1. Right: the queue gave o1 to search. The counter got nothing." width="720" height="300" />
  <figcaption>Two consumers is a sentence. The queue only heard one job.</figcaption>
</figure>

A queue of this kind is a work inbox, and the rules it plays by are worth stating plainly, because everything that went wrong follows from them. A worker asks the queue for work, a step called polling. The queue hands it a message and hides that message from everybody else for a while. The worker does the job and then explicitly deletes the message, and at that point the message is gone for good. If the worker crashes before deleting, the hidden message reappears after a timeout and somebody else picks it up. That reappearance is why the deletion is a separate step: it is how the queue knows the work finished rather than just started.

Read those rules back and the outcome is forced. Search polled first and was handed `o1`. It wrote the index. It deleted the message. The counter polled a moment later and found an empty queue. Support could find the order, because search did its job. The dashboard stayed at zero, because the counter never learned there was anything to count.

<figure>
  <img src="/blog/dashboard-zero.svg" alt="Support search finds order o1. The orders-today dashboard still shows 0." width="720" height="240" />
  <figcaption>Same checkout. One job ran. One did not.</figcaption>
</figure>

Nothing malfunctioned. The queue behaved exactly as documented. My mistake was in what I thought a consumer was.

## What a queue is actually for

Two search processes on that queue would have been completely fine. They would race each other for whatever arrives next, and each order would get indexed exactly once by whichever one got there first. That is the case a work queue is built for: one job, more hands, and the queue making sure two hands do not do the same task twice.

Search and a counter are not that. They are two different jobs that both need to know about the same event. The queue has no way to tell the difference, and no reason to care that they were written by the same person. It hands each message to one worker.

The workaround I reached for first was to run both functions inside one process: poll once, then call the indexer and the counter on the same message. That works right up until there are two replicas of that process, or until one crashes halfway and the message comes back. Then the counter increments twice for one checkout, and the dashboard says 2. The bug moved rather than went away.

It is also worth being clear that leaving the message in the queue does not help. Not deleting means "this job is not finished yet." It does not mean the counter can come back tomorrow and read the same checkout. A queue holds work that is pending, not history.

<figure>
  <img src="/blog/kafka-sqs.svg" alt="API publishes to one SQS queue. Worker A got event 1 and deleted it. Worker B never saw it. On the Kafka side, a search group and a counter group both read event 1." width="720" height="300" />
  <figcaption>One queue, two tasks, one of them loses the event.</figcaption>
</figure>

## Giving each job its own inbox

The first real fix is to stop sharing the queue.

SNS, the Simple Notification Service, is a bus rather than an inbox. The API publishes `o1` to a topic there, once, exactly as before. SNS then copies that event into every queue subscribed to the topic. So search gets its own queue with its own copy of `o1`, and the counter gets its own queue with its own copy.

Now both jobs work the way each of them expects. Search polls its queue, indexes `o1`, deletes its copy. The counter polls its queue, writes orders-today as 1, deletes its copy. Neither deletion is visible to the other, because they were never holding the same message. Sharing a repository is fine again, because they are no longer sharing an inbox.

Adding a third job later, email say, means another subscription and another queue. You pay for another copy of every event and you operate another inbox. That is the honest cost of this shape: fan-out is done by duplication, and the number of duplicates grows with the number of jobs.

<figure>
  <img src="/blog/sns-sqs-kafka.svg" alt="Left: API to SNS, then a copy into an SQS search queue and an SQS count queue. Right: API to one Kafka log, search and counter each keep a cursor." width="720" height="300" />
  <figcaption>Same publish. Same two jobs. SNS copies. Kafka lets both read.</figcaption>
</figure>

## Keeping one copy and two bookmarks

The other fix keeps a single copy of the event and changes who is allowed to read it.

The API publishes `o1` to a Kafka topic called `orders`. Kafka writes the event to a file on disk and leaves it there. Reading does not remove it. Search is one consumer group and the counter is a different consumer group, where a group is one job that may be several processes, and each group keeps its own bookmark recording how far it has read. That bookmark is called a cursor.

Both groups read `o1`, because reading is only moving your own bookmark forward. Search indexes the order and the counter writes orders-today as 1, and the event is still sitting in the file afterwards, unchanged, available to whatever job gets written next month.

That last part buys something the copies do not. If search is down for an hour, checkout keeps returning success, because the API was never calling search in the first place. The row is in the database and the event is on the topic. When search comes back, it reads on from its cursor and works through everything it missed. The counter, meanwhile, never waited for it.

<figure>
  <img src="/blog/kafka-search-down.svg" alt="At t=0 checkout returns 200 and search is down. In the same hour the counter reads o1. After an hour search is back and indexes o1 from its cursor." width="720" height="300" />
  <figcaption>Checkout did not wait on search. The log held o1.</figcaption>
</figure>

How long `o1` survives is set by time or by disk space, not by whether anybody has read it. And within a group, several processes still split the work between them, exactly as two search workers would on a queue of their own: only one of them indexes this particular order. The group is the job. A process inside it is another pair of hands on that job.

Kafka also slices a topic into partitions, which is how it scales and where its ordering guarantees actually live. That has its own way of surprising people and I wrote it up [separately](/blog/i-thought-kafka-kept-the-order-i-published/). None of it changes the choice being made here.

<figure>
  <img src="/blog/kafka-job.svg" alt="A user hits the API. The API writes an order row, then publishes. A search job and a counter job each poll the same log with their own cursor." width="720" height="300" />
  <figcaption>One publish. Two jobs. Same checkout, twice. The log stays.</figcaption>
</figure>

## Which one I reach for

<figure>
  <img src="/blog/which-inbox.svg" alt="Three paths for o1: one SQS with one winner, SNS copying into two queues, Kafka with one log and two cursors." width="720" height="300" />
  <figcaption>Same o1. The inbox is what changes.</figcaption>
</figure>

A queue on its own when `o1` is a task and exactly one worker should perform it. Index this order, once. Extra workers just share the load.

SNS with a queue per job when queues are already the shape of the system and a second job needs the same events. One publish, two copies, two deletions. Each new job is another copy.

Kafka when several jobs will keep reading the same checkouts, including jobs that do not exist yet. One log and a cursor per job. A new job is a new group rather than a new queue, and replaying history is just moving a cursor back to where `o1` sits.

The sentence I did not have when I started is that an inbox belongs to a job, not to a repository. Search and the counter shared a queue because they shared a codebase, and the queue had no idea it was arbitrating between two unrelated jobs.

I still find new ways to make the same mistake. A filter on an SNS subscription that quietly drops `o1` from the counter's queue leaves search perfectly healthy and the dashboard at zero, which looks identical to the original bug from the outside.

<figure>
  <img src="/blog/sns-filter-drop.svg" alt="API publishes to SNS. A filter drops o1 from the count queue. Search still gets a copy. The dashboard stays 0." width="720" height="260" />
  <figcaption>Fan-out with a filter is still a delete for one job.</figcaption>
</figure>

Giving two Kafka services the same group id does it too. Same group means one job, so the two of them divide the partitions between themselves and each sees roughly half the events. That is the single-queue bug rebuilt on top of a log.

<figure>
  <img src="/blog/shared-group-id.svg" alt="Topic orders holds o1. Search and the counter share group id search. Search got o1. The counter is split away." width="720" height="260" />
  <figcaption>Same group id. Same steal as one SQS queue.</figcaption>
</figure>

And reading "the log keeps everything" as "we do not need backups" only holds until retention expires, at which point `o1` is gone for every job at once.

I am not done being wrong about this. I am just done putting search and a counter on one queue and calling it fan-out.

If this is useful, wrong, or incomplete, write to me.
