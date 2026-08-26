---
title: "I thought two workers on one SQS queue could do two jobs"
description: "SNS-SQS vs Kafka. One SQS inbox and two tasks steal from each other. SNS copies into two queues. Kafka keeps one log and two cursors."
pubDate: 2026-08-23
tags: ["systems"]
draft: true
---

The API publishes. The broker is one SQS queue. Two workers share the same repo. One path updates search. The other updates a count. Both poll that queue.

I called that "two consumers." I thought both tasks would see the event. They will not.

<figure>
  <img src="/blog/kafka-sqs.svg" alt="API publishes to one SQS queue. Worker A got event 1 and deleted it. Worker B never saw it. On the Kafka side, a search group and a counter group both read event 1." width="720" height="300" />
  <figcaption>One queue, two tasks, one of them loses the event.</figcaption>
</figure>

## The wrong model

SQS is a job inbox. A worker polls. It gets one message. The other worker cannot see it. Finish and delete, and the job is gone. Die, and it comes back after a timeout.

That is the right tool when both workers are the same job: do this once. Extra processes drain the queue faster. They fight over who works. They do not fight over meaning.

I used that picture for two jobs. Search and count in one codebase, one queue. I treated "we share the repo" as "we share the event." The broker does not care about your repo. It hands the message to one poller.

## Where it broke

Search takes event 1 and deletes it. Counter never sees event 1. Or both touch the same row because the binary is one process and both think they own the event.

The conflict is not "SQS is broken." The conflict is two tasks sharing one inbox.

SQS can hold a message for a while if nobody deletes it. That is still not a log. Retention there is "this job is not done yet," not "everyone who cares can read this again."

## SNS plus SQS

Need both tasks to see every publish? You do not share that queue.

You fan out. SNS is the usual bus. The API publishes once to SNS. SNS copies the event into two SQS queues. Search has its own inbox. Counter has its own inbox. Two copies. Two deletes. Same repo is still allowed. The inbox is not shared.

That is the queue way to do "many care." It works. You pay for the copy. You operate two queues. A new job means a new subscription and a new inbox.

<figure>
  <img src="/blog/sns-sqs-kafka.svg" alt="Left: API to SNS, then a copy into an SQS search queue and an SQS count queue. Right: API to one Kafka log, search and counter each keep a cursor." width="720" height="300" />
  <figcaption>Same publish. Same two jobs. SNS copies. Kafka lets both read.</figcaption>
</figure>

## Kafka

Kafka is a cluster that keeps a stream of events on disk and lets other processes read that stream later. You append. A consumer polls. The events stay. Retention is time or size, not "someone already took this."

The name on that stream is a topic. Inside it are partitions. That cut is how it scales and how it orders. I wrote that fight in [another post](/blog/i-thought-kafka-kept-the-order-i-published/). The job of the thing is simpler than the cut.

The job: something happened, and more than one other system needs to know, without you wiring them to each other, and without you copying the stream.

Your API writes the row that must be true. Then it publishes "this happened." Search is a consumer group. Counter is a consumer group. Each group gets each message. The log stays. Nobody deletes anybody else.

<figure>
  <img src="/blog/kafka-job.svg" alt="An API writes a row and publishes an event into Kafka. A search job and a counter job each poll the same log with their own cursor." width="720" height="280" />
  <figcaption>One publish. Two jobs. Same events, twice. The log stays.</figcaption>
</figure>

It is not "every process sees every message." Two members inside the search group still split work, like two SQS workers on the search-only queue. The group is the job. The member is a replica of that job.

## SNS-SQS versus Kafka

Same picture, three ways.

**One SQS.** One inbox. One winner per message. Right for one job. Wrong for two tasks.

**SNS plus two SQS.** One publish, two copies, two inboxes, two deletes. Right for two jobs if you want queues. A third job is another copy.

**Kafka.** One log, one copy, a cursor per job. A third job is another group, not another queue. Replay is "move the cursor," not "re-drive a dead letter."

I pick SQS when the message is a job and only one worker should ever run it. I pick SNS plus SQS when I already live in queues and I need a second job. I pick Kafka when several jobs will keep reading the same facts, including later.

## The model that stuck

The inbox belongs to a job, not to a repo.

**1. One queue, one job.** Extra workers are replicas. They share work. They do not share meaning.

**2. Two jobs, two inboxes, or one log with two cursors.** SNS copies. Kafka does not have to.

**3. A Kafka group is the job.** Members of that group are SQS workers. A second group is a second job, not a second worker on the same inbox.

## How I recognize the old model now

I am in the old model when:

- Two tasks poll one SQS queue because they live in one repo.
- I say "we have two consumers" and I have not said whether I mean two jobs or two replicas.
- I add a Kafka process and expect it to see every message, and it is in the same group as the first one.
- I call Kafka a louder SQS.

The replacement habit is one sentence: is this a second job, or a second pair of hands on the same job.

## What I would still get wrong

A filter in SNS that drops the event one job still needed.

A Kafka group id copied into both services. They are now one job again. They steal, just like SQS.

Treating "the log stays" as "I do not need backups." Retention ends.

I am not done being wrong about inboxes. I am just done putting two tasks on one queue and calling it fan-out.

If this is useful, wrong, or incomplete, write to me.
