---
title: "I thought Kafka kept the order I published"
description: "One checkout, three lanes, and two jobs reading the same line. Order lives on a partition, not on the topic."
pubDate: 2026-08-23
tags: ["systems"]
---

I used to treat Kafka like a queue with a nicer name. My picture was simple. Messages go in one end, they come out the other end in the same order, and if a job falls behind you start more copies of it.

Both halves of that are wrong. One checkout is enough to show why.

## A buyer taps Buy

A buyer, u1, checks out. The API does two things, in this order, and the order is deliberate.

First it writes the order row to the database. That row is the fact. If the write fails there is no order, and the buyer gets an error.

Then it announces what happened. It sends a small event, something like order id, user, total, and time, to a stream called `orders`. Announcing is not the same as asking. The API does not call the search service, and it does not call the dashboard. It returns a success response and the buyer sees a confirmation page.

That stream is a **topic**. A topic is a name, not a pipe. Sitting under the name is a set of files that Kafka keeps on disk, and someone chose how many of them there would be when the topic was created. This one has three.

Each of those files is a **partition**, and each one is a log in the plain sense of the word: new events only ever get added to the end. Nothing is inserted in the middle. Nothing is removed when somebody reads it. Events sit there until they age out, and how long that takes is a setting measured in days or in disk space. It is not "until a reader picks this up."

Within one partition, the order is exactly the order things were added. Between two partitions there is no order at all. Nothing anywhere records that a line in one file happened before a line in another.

So when the API sends u1's checkout, something has to choose which of the three files it goes in. The API does not choose by hand. It attaches a **key**, and the Kafka client library inside the API turns that key into a lane.

It does this by hashing. A hash function takes any value and produces a number from it, and the same input always produces the same number. Take that number, divide by three, and keep the remainder. The remainder is 0, 1, or 2, which is exactly the set of lanes available. That is what `hash(key) % 3` means.

I key by user id. For u1 the remainder comes out 0, so u1's checkout is appended to the end of partition 0. The Kafka server that holds that file, called a **broker**, writes it where it was told. The broker does not second-guess the lane.

A second later, u2 checks out. For u2 the remainder is 2, so that event goes on the end of a different file entirely.

A moment later, u1 buys again. Same user id, so the same hash, so the same remainder, so the same lane. That second checkout lands directly after the first one on partition 0.

That is the only ordering anything ever promised me: one user's own checkouts, in the order the API sent them, within one file.

<figure>
  <img src="/blog/kafka-key.svg" alt="An API publishes into a hash funnel. u1 and u3 land on lane 0, including a later u1. u2 and u4 land on lane 2. Lane 1 is empty." width="720" height="300" />
  <figcaption>u1 goes to lane 0. Later u1 goes to lane 0 again. That is the only order you were promised.</figcaption>
</figure>

If I leave the key off, the client spreads events across the lanes on its own, and then even one buyer's checkouts can end up in different files with no order between them.

## Two jobs read the same line

Two other jobs care about that checkout. The search service wants the order in its index so support can look it up. A counter wants the running total of orders today written back to the database.

Each of these is a **consumer group**. A consumer group is one job, which may be running as several processes, and it keeps a bookmark on every partition recording how far it has read. That bookmark is called a cursor.

The important part is that the groups are independent. Kafka gives search all three partitions to read, and gives the counter all three partitions to read, and the two never interfere. Both see every checkout. Search reading u1's order does not take it away from the counter, because reading is just moving your own bookmark forward. The event is still sitting in the file afterwards.

Inside a single group, the partitions get divided among the processes. If search is running three processes, each one takes a lane. The process on partition 0 sees u1's checkout. The process on partition 2 sees u2's. Neither sees the other's, unless one process happens to be holding two lanes.

The counter does the same division over its own processes. Its process on partition 0 also sees u1's checkout. Search did not pass it along. Two different processes read the same line out of the same file because they belong to two different groups with two separate bookmarks.

<figure>
  <img src="/blog/kafka-job.svg" alt="A user hits the API. The API writes an order row to the database, then publishes to a Kafka topic named orders with three partitions. A search group and a counter group each poll that topic." width="720" height="300" />
  <figcaption>The row is the fact. The topic is the news. Two groups, two cursors, same topic.</figcaption>
</figure>

<figure>
  <img src="/blog/kafka-groups.svg" alt="The API hashes user ids onto partitions. Search members s0 s1 s2 each take one partition. Counter members c0 c1 c2 take the same three partitions again." width="720" height="300" />
  <figcaption>The API hashes the key. Each group covers every partition. Same event, two jobs.</figcaption>
</figure>

This is also why search can be down for an hour without anybody noticing. The events are still in the files. When search comes back it picks up from its bookmark and works through the backlog. The order row was never waiting on it. Had the API instead written the row and then made direct calls to search and to the counter, the buyer would be waiting on both of them, and a failure in either one would turn into a failed checkout.

## The morning the order looked wrong

The break arrived as a question from support. Why did u2's order appear in the search index before u1's, when u1 checked out first?

Nothing was broken. u1's checkout was in lane 0 and u2's was in lane 2, and there is no ordering between two lanes. The search process working lane 2 simply got through its backlog faster than the one working lane 0. I had asked for per-user ordering and I had received exactly that. What I had assumed on top of it, that the whole topic came out in the sequence the API sent it, was never on offer, and there is no setting that turns it on.

The drawing in my head had been a single cylinder with an arrow going in and an arrow coming out. Publish on the left, read on the right. Nothing in that picture tells you the cylinder is sliced into three.

<figure>
  <img src="/blog/kafka-pipe.svg" alt="A single queue with messages 1 2 3 4 in one line, next to three lanes where 1 and 3 sit on lane 0 and 2 and 4 sit on lane 2." width="720" height="280" />
  <figcaption>I called this a queue. It is lanes. 1 can finish after 4.</figcaption>
</figure>

If I had genuinely needed every checkout in one single sequence, the way to get it is one partition. One file, one line of events, and the model I had in my head becomes true. The price is that a single file and a single reader are then the ceiling on how fast the whole thing can go. That is a fine place for a lot of systems to start.

## The Friday I added consumers

The other half of the wrong model died on a Friday.

Checkout was running at about two hundred orders a second. Each event turned into a sizeable write into the search index, and one process could not keep up, so it fell further behind all afternoon. I started four more search processes.

Nothing moved.

A partition can be handed to at most one process within a group. With three partitions and three processes already working, there was nothing left to give the new ones. They connected, asked for work, and were assigned no lanes. The partition count is the ceiling on how many processes in a group can do anything at all.

<figure>
  <img src="/blog/kafka-consumers.svg" alt="Three lanes feed c1, c2, and c3. A dashed box for c4 sits aside and polls nothing." width="720" height="300" />
  <figcaption>You added a consumer. Kafka did not add a lane. The fourth process is unemployed.</figcaption>
</figure>

It works the same way in the other direction. Two processes and three partitions means one of them carries two lanes, and those two lanes still have no order between them. So even one process reading everything does not give you a single sequence. It gives you three ordered files that it reads from in whatever order it happens to ask.

Adding lanes was the only thing that would have helped, and that is a decision about the topic itself, not about how many processes I run.

## Three promises, not a tuning knob

So the number of partitions, usually written as n, is not a dial I get to turn on a bad afternoon. Choosing it makes three promises at once.

**How hard you can write.** Three partitions means three files that can be appended to at the same time. If a bad key sends most traffic to one lane, that lane does all the work while the other two sit idle, and the extra lanes bought me nothing.

**How hard one group can read.** Search can put at most three processes to work, because there are three lanes to hand out. There is no fourth pair of hands available until there is a fourth partition.

**Where order lives.** u1's checkouts stay in sequence only because they share a lane. The partition count is really a count of how many independent ordered histories I asked for.

I can raise it later on a live topic. Kafka adds the new empty partitions, everything already written stays where it is, and new events start dividing by the new number instead. A buyer who always landed in lane 0 can start landing in lane 4. From then on that user is consistent again, but support looking up everything that buyer ever did now has to read two lanes and merge them.

I cannot lower it. The events already exist across three files and Kafka will not stitch them back into one. If traffic drops and three search processes are more than I need, the three partitions are still there.

The way out is to create a new topic with the number I actually want, send new checkouts to it, and, if I need the history, read the old topic from the beginning and write it into the new one. That is a migration with a plan, not a setting I change.

<figure>
  <img src="/blog/kafka-n.svg" alt="Three partitions on orders. A shrink to n equals 1 is crossed out. Grow keeps old lines. Fewer lanes means a new topic and a replay." width="720" height="280" />
  <figcaption>You can add lanes. You cannot remove them. A smaller n is a new topic.</figcaption>
</figure>

So I pick the number for the busiest day I am willing to run, not for this afternoon's lag graph. Too few and search can never catch up no matter how many processes I start. Too many and I pay for idle lanes and a division I cannot undo.

## What the checkout taught me

**The partition is the log.** Ordering, history, and how far each reader has got all attach to a single partition. The topic is just a name for a set of them. I can add lanes later and I cannot take them away.

**The key decides which ordering I get.** Whatever I put in the key is the thing whose events stay in sequence. Key by user and I get per-user order. Leave it off and I have agreed to let events scatter.

**A group is a set of bookmarks, one per partition.** Another group is another independent reader of the same events. More processes inside one group buys more parallelism, but only up to the number of lanes, and after that they sit idle.

I still draw the API on the left and the two services on the right. I just draw three lanes in the middle now instead of one pipe. Before I say Kafka will keep the order, I have to be able to answer three questions: which key, how many lanes, and which group. If I cannot answer all three, I am guessing.

I still get this wrong. I pick a key so coarse that one lane burns while two idle. I pick one so fine that the ordering I wanted is gone. I add lanes to fix a lag graph and then wonder why one buyer's history is split across two of them. And I still catch myself treating an idle fourth process as a bug in Kafka, when it is the assignment rule doing exactly what it says.

I am not done being wrong about logs that look like queues. I am just done asking a topic for an order it never had.

The other half of this checkout is what happens when search and the counter share a queue instead of a log. I wrote that [separately](/blog/sns-sqs-vs-kafka/).

If this is useful, wrong, or incomplete, write to me.
