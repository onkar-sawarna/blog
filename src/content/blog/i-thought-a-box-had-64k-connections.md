---
title: "I thought a box could only hold 64k connections"
description: "A load test from one laptop stops near 64,000. That is a source-port limit, not a law of TCP. A server facing many clients is counting something else entirely."
pubDate: 2026-08-16
tags: ["networking", "systems"]
---

I used to think 65,535 was a hard ceiling on TCP. One machine, about 64,000 connections, and that was the end of the conversation.

Then I read that a chat service was holding millions of connections on a single server, and the number felt like a lie. So I tried to prove it was one.

## The loop that stopped at 64,000

I wrote the simplest test I could think of. A laptop, call it `m1`, opening connections in a loop to one server, call it `m2`, on port 443. Hold each one open. Count them.

It climbed fast and then stopped a little short of 64,000. The next call to `connect()` came back with an error saying it could not assign the requested address.

That looked like proof. A machine has 65,535 ports, I had used them all, and no amount of marketing was going to get anyone to five million.

The reasoning underneath was this. A port number is a 16-bit field, so it can hold a value from 0 to 65,535. Take away the ones reserved for other purposes and you land near 64,000. When my laptop opens a connection, it has to put a source port on it so replies can find their way back to the right socket, and the kernel picks one automatically from a range set aside for this. That is called an ephemeral port, ephemeral because it belongs to that one connection and goes back in the pool afterwards. First connection gets 49152, next gets 49153, and so on up.

So if a connection needs a port, and there are only 64,000 usable ports, then a machine gets 64,000 connections. The arithmetic is clean. It is also answering a question I had not thought to ask carefully.

<figure>
  <img src="/blog/64k-loadtest.svg" alt="One laptop looping connect hits 64k and calls the claim fake. The other box: you fixed src, they did not." width="720" height="240" />
  <figcaption>The load test did not disprove the server. It turned you into one client.</figcaption>
</figure>

## What the kernel is actually writing down

A connection is not a port. When a packet arrives, the kernel has to work out which open socket it belongs to, and it does that by looking at four things together:

`source IP, source port, destination IP, destination port`

Those four together are called a 4-tuple, and the whole set has to be unique. Two connections are allowed to share a destination port. They are allowed to share a destination IP. They just cannot match on all four at once, because then the kernel would have no way to tell their packets apart.

You can watch this on a Linux box. Connection tracking keeps a row per flow, and `conntrack -L` prints them:

```
conntrack -L
```

If the tool is not installed, the same rows are readable as a file:

```
cat /proc/net/nf_conntrack
```

Every line is one 4-tuple, with a state on the end. Point the loop at `m2` on 443 and watch: the destination IP is the same on every line, the destination port is 443 on every line, my laptop's IP is the same on every line, and the only column moving is the source port. Three of the four fields were nailed down by the test I wrote. There was exactly one field left free, that field is 16 bits wide, and so the test ran out after about 64,000 rows.

<figure>
  <img src="/blog/64k-one-client.svg" alt="m1 opening many connections to m2 on port 443. Only the source port changes, so the table tops out around 64k." width="720" height="280" />
  <figcaption>m1 to m2:443. Three fields fixed. Source port is the only knob.</figcaption>
</figure>

Some of those rows say ESTABLISHED, which are connections I still hold open. Others say TIME_WAIT, which are ones I already closed. TIME_WAIT exists because after a close, stray packets from that connection may still be in flight, and the kernel keeps the tuple reserved for a while so a brand new connection does not receive somebody else's leftovers. The practical effect during a load test is that closing sockets does not immediately give the ports back. They sit unavailable for that same destination until the wait expires.

<figure>
  <img src="/blog/conntrack-tuple.svg" alt="A Linux box and a conntrack -L listing. Each line is one 4-tuple: src, sport, dst, dport." width="720" height="300" />
  <figcaption>conntrack -L is the 4-tuple, one line per flow. src, sport, dst, dport.</figcaption>
</figure>

Once you can see which column is the bottleneck, the ways out are obvious, because any of the other three fields will do:

- Give `m1` a second IP address and there is another 64,000 to the same destination.
- Have `m2` listen on a second port and there is another 64,000.
- Point at a destination address that fans out to several machines and you are no longer talking about one server anyway.

I should say what this argument is not about. I am assuming the network card, the CPU, and the link all have room to spare, and asking only how many distinct 4-tuples the stack will permit. On a real machine you will usually run into memory or file descriptors or a saturated link first. Those are real limits. They are just different limits.

## Turning the test around

Now put the server on the other side of the same rule.

`m2` is listening on 443. The clients are phones and laptops out on the internet, each with its own address, each picking its own source port. The rows look like this:

- `1.2.3.4:51000 → m2:443`
- `5.6.7.8:51000 → m2:443`
- `9.9.9.9:44321 → m2:443`

Same destination IP on every row. Same destination port on every row. But the source IP is different every time, and that is a 32-bit field with billions of possible values rather than a 16-bit one. Two of those clients even picked the same source port as each other, 51000, and it does not matter, because the tuples still differ.

The server is not spending its own ephemeral ports to accept these. It never calls `connect()`. It accepts, and holds one row per client. The 16-bit port field that limited my laptop lives on the client's side of each row, and each client only needs one of its own.

<figure>
  <img src="/blog/64k-many-clients.svg" alt="Many phones and laptops connecting to one server on port 443. Each client has its own source IP, so the server table can grow past 64k." width="720" height="300" />
  <figcaption>The server is m2. Each client is a different source IP. 64k is not the cap.</figcaption>
</figure>

That is how a box holds millions of connections. Millions of separate peers, one listening port, one large connection table. My loop had five million connections' worth of ambition and one source IP to spend it from.

The real ceiling on the server is somewhere else entirely. Every open socket is a file descriptor, and the per-process limit on those is often 1024 by default, which will stop you long before ports would. Every socket also costs memory for its buffers, and anything keeping a timer per connection costs more. Getting to millions is an exercise in descriptors, memory, and how efficiently the process waits on all of them at once. None of that is about the size of a port field.

## The load balancer that undoes it

There is one arrangement that quietly puts you back in my laptop's position.

The reasoning above only holds while the server sees each client's own address. Put something in the middle that rewrites the source address, such as a load balancer doing source NAT, or a proxy that terminates the client connection and opens its own connection onward to the backend, and every user now arrives from that one machine's address.

Look at that hop as a 4-tuple. Source IP is the load balancer, fixed. Destination IP is the backend, fixed. Destination port is 443, fixed. The only column left free is the load balancer's source port. That is exactly the shape of my loop, with a different name on it. Around 64,000 connections to that backend and the load balancer cannot open another. People call it SNAT port exhaustion, and it is the same rule producing the same number for the same reason.

A load balancer that preserves the client address does not do this, because the source IP keeps varying. The dangerous one is the one that makes everybody look like a single machine.

<figure>
  <img src="/blog/direct-vs-lb.svg" alt="Left: phones A, B, and C connect straight to the server, each with its own source IP. Right: the same phones hit a load balancer that talks to the server as one IP, so that hop is limited to about 64k connections." width="720" height="280" />
  <figcaption>Straight to the server: src varies. Through a SNAT LB: src is one IP, 64k is back.</figcaption>
</figure>

## What the loop taught me

The question is never "how many connections can a machine have." It is which of the four fields are pinned down, and which one is left to vary.

**One client to one service.** Source port is the only free column, so expect roughly 64,000. More local addresses or more destination ports if you genuinely need more.

**One service to many clients.** Source IP is the free column, and it is enormous. The table can be huge, and the limits you meet will be descriptors and memory.

**A connection is the whole 4-tuple.** The port is one column of four, and treating it as the identity of a connection is what caused all of this.

I still write load generators, and I still watch the ephemeral range when I do. I just do not use that number to argue about how many users a server can hold.

I still get pieces of this wrong. I forget that extra addresses and IPv6 change the arithmetic again. I forget that "one connection per user" is a product decision, and that the table is counted in sockets rather than in users. I raise the ephemeral port range and leave the file descriptor limit where it was, then hit the smaller wall and blame ports for it. And I tune settings on a box that is failing without ever running `conntrack -L` on it to see which column actually ran out.

I am not done being wrong about sockets. I am just done treating 64,000 as a property of TCP itself.

If this is useful, wrong, or incomplete, write to me.
