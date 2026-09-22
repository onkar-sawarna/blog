---
title: "I thought Kafka kept the order I published"
description: "One checkout, three lanes, and two jobs reading the same line. Order lives on a partition, not on the topic."
pubDate: 2026-08-23
tags: ["systems"]
---

I used to treat Kafka like a queue with a nicer name. Messages go in one end, they come out the other end in the same order, and if a job falls behind you start more copies of it.

Both halves of that are wrong. One checkout is enough to show why.

## A buyer taps Buy

A buyer, u1, checks out. The API does two things, in this order.

First it writes the order row to the database. That row is the fact. If the write fails there is no order.

Then it announces what happened. It sends a small event to a stream called `orders`. The API does not call search, and it does not call the dashboard. It returns success and the buyer sees a confirmation page.

That stream is a **topic**. A topic is a name, not a pipe. Under the name sit files on disk. This one has three.

Each file is a **partition**: a log. New events only get added to the end. Nothing is inserted in the middle. Nothing is removed when somebody reads it. Events sit there until they age out.

Within one partition, the order is the order things were added. Between two partitions there is no order at all.

So when the API sends u1's checkout, something has to pick a file. The API attaches a **key**. The client hashes that key and keeps the remainder after dividing by three. That is what `hash(key) % 3` means: pick lane 0, 1, or 2.

I key by user id. For u1 the remainder is 0, so the checkout is appended to partition 0. The Kafka server that holds that file is a **broker**. It writes where it was told.

A second later, u2 checks out. For u2 the remainder is 2, so that event goes on a different file.

A moment later, u1 buys again. Same user id, same hash, same lane. That second checkout lands directly after the first one on partition 0.

That is the only ordering anything ever promised me: one user's own checkouts, in the order the API sent them, within one file.

<figure>
  <object class="figure-svg" data="/blog/kafka-key.svg" type="image/svg+xml" width="720" height="300" style="aspect-ratio: 720 / 300" aria-label="An API publishes into a hash funnel. u1 and u3 land on lane 0, including a later u1. u2 and u4 land on lane 2. Lane 1 is empty.">
    <img src="/blog/kafka-key.svg" alt="An API publishes into a hash funnel. u1 and u3 land on lane 0, including a later u1. u2 and u4 land on lane 2. Lane 1 is empty." width="720" height="300" />
  </object>
  <figcaption>Figure 1. u1 goes to lane 0. Later u1 goes to lane 0 again. That is the only order you were promised.</figcaption>
</figure>

If I leave the key off, even one buyer's checkouts can land in different files with no order between them.

## Two jobs read the same line

Two other jobs care about that checkout. Search wants the order in its index. A counter wants the running total of orders today.

Each of these is a **consumer group**: one job, which may be several processes, with a bookmark on every partition. That bookmark is a cursor.

The groups are independent. Kafka gives search all three partitions, and gives the counter all three partitions. Both see every checkout. Search reading u1's order does not take it away from the counter, because reading is just moving your own bookmark. The event is still in the file afterwards.

Inside a single group, the partitions get divided among the processes. If search is running three processes, each one takes a lane. The process on partition 0 sees u1. The process on partition 2 sees u2.

The counter does the same split over its own processes. Its process on partition 0 also sees u1. Two different processes read the same line because they belong to two different groups.

<figure>
  <img src="/blog/kafka-job.svg" alt="A user hits the API. The API writes an order row to the database, then publishes to a Kafka topic named orders with three partitions. A search group and a counter group each poll that topic." width="720" height="300" />
  <figcaption>Figure 2. The row is the fact. The topic is the news. Two groups, two cursors.</figcaption>
</figure>

<figure>
  <img src="/blog/kafka-groups.svg" alt="The API hashes user ids onto partitions. Search members s0 s1 s2 each take one partition. Counter members c0 c1 c2 take the same three partitions again." width="720" height="300" />
  <figcaption>Figure 3. The API hashes the key. Each group covers every partition. Same event, two jobs.</figcaption>
</figure>

This is why search can be down for an hour without anybody noticing. The events are still in the files. When search comes back it picks up from its bookmark. The order row was never waiting on it. Had the API called search and the counter directly, the buyer would be waiting on both.

## The morning the order looked wrong

Support asked why u2's order appeared in search before u1's, when u1 checked out first.

Nothing was broken. u1 was in lane 0 and u2 was in lane 2. There is no ordering between two lanes. The search process on lane 2 simply got through its backlog faster. I had asked for per-user ordering and I had received exactly that. Topic-wide order was never on offer.

<figure>
  <img src="/blog/kafka-pipe.svg" alt="A single queue with messages 1 2 3 4 in one line, next to three lanes where 1 and 3 sit on lane 0 and 2 and 4 sit on lane 2." width="720" height="280" />
  <figcaption>Figure 4. I called this a queue. It is lanes. 1 can finish after 4.</figcaption>
</figure>

If I genuinely needed every checkout in one single sequence, the way to get it is one partition. One file, one line of events. The price is that a single file and a single reader are then the ceiling.

## The Friday I added consumers

Checkout was running at about two hundred orders a second. One search process could not keep up. I started four more.

Nothing moved.

A partition can be handed to at most one process within a group. With three partitions and three processes already working, there was nothing left to give. They connected, asked for work, and were assigned no lanes. The partition count is the ceiling on how many processes in a group can do anything at all.

<figure>
  <object class="figure-svg" data="/blog/kafka-consumers.svg" type="image/svg+xml" width="720" height="300" style="aspect-ratio: 720 / 300" aria-label="Three lanes feed c1, c2, and c3. A dashed box for c4 sits aside and polls nothing.">
    <img src="/blog/kafka-consumers.svg" alt="Three lanes feed c1, c2, and c3. A dashed box for c4 sits aside and polls nothing." width="720" height="300" />
  </object>
  <figcaption>Figure 5. You added a consumer. Kafka did not add a lane. The fourth process is unemployed.</figcaption>
</figure>

Two processes and three partitions means one of them carries two lanes, and those two lanes still have no order between them. Even one process reading everything does not give you a single sequence.

Adding lanes was the only thing that would have helped, and that is a decision about the topic, not about how many processes I run.

## Three promises, not a tuning knob

The number of partitions, n, is not a dial I get to turn on a bad afternoon. Choosing it makes three promises at once.

**How hard you can write.** Three partitions means three files that can be appended to at the same time. If a bad key sends most traffic to one lane, the extra lanes bought me nothing.

**How hard one group can read.** Search can put at most three processes to work. There is no fourth pair of hands until there is a fourth partition.

**Where order lives.** u1's checkouts stay in sequence only because they share a lane.

I can raise n later. Kafka adds empty partitions. Old events stay where they are. New events start dividing by the new number. A buyer who always landed in lane 0 can start landing in lane 4.

I cannot lower it. The events already exist across three files. Kafka will not stitch them back into one.

The way out is a new topic with the number I actually want, then a replay if I need the history. That is a migration, not a setting.

<figure>
  <img src="/blog/kafka-n.svg" alt="Three partitions on orders. A shrink to n equals 1 is crossed out. Grow keeps old lines. Fewer lanes means a new topic and a replay." width="720" height="280" />
  <figcaption>Figure 6. You can add lanes. You cannot remove them. A smaller n is a new topic.</figcaption>
</figure>

I pick the number for the busiest day I am willing to run, not for this afternoon's lag graph.

The partition is the log. The key decides whose events stay in sequence. A group is a set of bookmarks, one per partition. Another group is another independent reader of the same events.

I still pick a key so coarse that one lane burns while two idle. I still add a fourth process and treat the idle one as a Kafka bug.

The other half of this checkout is what happens when search and the counter share a queue instead of a log. I wrote that [separately](/blog/sns-sqs-vs-kafka/).

If this is useful, wrong, or incomplete, write to me.
