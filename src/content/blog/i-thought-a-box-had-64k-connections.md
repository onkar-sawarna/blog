---
title: "I thought a box could only hold 64k connections"
description: "A load test from one laptop stops near 64,000. That is a source-port limit, not a law of TCP. A server facing many clients is counting something else entirely."
pubDate: 2026-08-16
tags: ["networking", "systems"]
---

I used to think 65,535 was a hard ceiling on TCP. One machine, about 64,000 connections, and that was the end of the conversation.

Then I read that a chat service was holding millions of connections on a single server, and the number felt like a lie. So I tried to prove it was one.

## The loop that stopped at 64,000

I wrote the simplest test I could think of. A laptop, `m1`, opening connections in a loop to one server, `m2`, on port 443. Hold each one open. Count them.

It climbed fast and then stopped a little short of 64,000. The next `connect()` said it could not assign the requested address.

That looked like proof. A machine has 65,535 ports. I had used them all.

The reasoning was this. A port number is 16 bits, so it holds 0 to 65,535. Take away the reserved ones and you land near 64,000. When my laptop opens a connection, it has to put a source port on it so replies can find the right socket. The kernel picks one from a range set aside for this. That is an ephemeral port: it belongs to that one connection and goes back in the pool afterwards.

So if a connection needs a port, and there are only 64,000 usable ports, a machine gets 64,000 connections. The arithmetic is clean. It is answering a question I had not thought to ask.

<figure>
  <img src="/blog/64k-loadtest.svg" alt="One laptop looping connect hits 64k and calls the claim fake. The other box: you fixed src, they did not." width="720" height="240" />
  <figcaption>Figure 1. The load test did not disprove the server. It turned you into one client.</figcaption>
</figure>

## What the kernel is actually writing down

A connection is not a port. When a packet arrives, the kernel has to work out which socket it belongs to. It looks at four things together:

`source IP, source port, destination IP, destination port`

Those four together are a 4-tuple. The whole set has to be unique. Two connections may share a destination port. They may share a destination IP. They cannot match on all four, or the kernel could not tell their packets apart.

You can watch this on a Linux box. Connection tracking keeps a row per flow:

```
conntrack -L
```

Or, if that tool is missing:

```
cat /proc/net/nf_conntrack
```

Every line is one 4-tuple. Point the loop at `m2` on 443 and watch: destination IP is the same, destination port is 443, my laptop's IP is the same, and the only column moving is the source port. Three of the four fields were nailed down by the test. One field left free, 16 bits wide, so the test ran out after about 64,000 rows.

<figure>
  <object class="figure-svg" data="/blog/64k-one-client.svg" type="image/svg+xml" width="720" height="280" style="aspect-ratio: 720 / 280" aria-label="m1 opening many connections to m2 on port 443. Only the source port changes, so the table tops out around 64k.">
    <img src="/blog/64k-one-client.svg" alt="m1 opening many connections to m2 on port 443. Only the source port changes, so the table tops out around 64k." width="720" height="280" />
  </object>
  <figcaption>Figure 2. m1 to m2:443. Three fields fixed. Source port is the only knob.</figcaption>
</figure>

Some rows say ESTABLISHED: connections I still hold. Others say TIME_WAIT: ones I already closed. TIME_WAIT exists so leftover packets from a closed connection do not hit a brand new one on the same tuple. Closing sockets does not give the ports back immediately.

<figure>
  <img src="/blog/conntrack-tuple.svg" alt="A Linux box and a conntrack -L listing. Each line is one 4-tuple: src, sport, dst, dport." width="720" height="300" />
  <figcaption>Figure 3. conntrack -L is the 4-tuple, one line per flow.</figcaption>
</figure>

Once you can see which column is the bottleneck, the ways out are obvious:

- Give `m1` a second IP and there is another 64,000 to the same destination.
- Have `m2` listen on a second port and there is another 64,000.
- Point at a destination that fans out to several machines and you are no longer talking about one server.

I am only asking how many distinct 4-tuples the stack will permit. On a real machine you usually hit memory, file descriptors, or a saturated link first. Those are real limits. They are different limits.

## Turning the test around

Now put the server on the other side of the same rule.

`m2` is listening on 443. The clients are phones and laptops, each with its own address, each picking its own source port:

- `1.2.3.4:51000 → m2:443`
- `5.6.7.8:51000 → m2:443`
- `9.9.9.9:44321 → m2:443`

Same destination IP. Same destination port. The source IP is different every time, and that is a 32-bit field with billions of values. Two clients even picked source port 51000. It does not matter. The tuples still differ.

The server is not spending its own ephemeral ports to accept these. It never calls `connect()`. It accepts, and holds one row per client. The 16-bit port field that limited my laptop lives on the client's side of each row.

<figure>
  <img src="/blog/64k-many-clients.svg" alt="Many phones and laptops connecting to one server on port 443. Each client has its own source IP, so the server table can grow past 64k." width="720" height="300" />
  <figcaption>Figure 4. The server is m2. Each client is a different source IP. 64k is not the cap.</figcaption>
</figure>

That is how a box holds millions of connections. Millions of separate peers, one listening port, one large table. My loop had five million connections' worth of ambition and one source IP to spend it from.

The real ceiling on the server is somewhere else. Every open socket is a file descriptor, often capped at 1024 by default. Every socket costs memory. Getting to millions is descriptors, memory, and how efficiently the process waits on all of them. None of that is the size of a port field.

## The load balancer that undoes it

There is one arrangement that puts you back in my laptop's position.

The reasoning above only holds while the server sees each client's own address. Put something in the middle that rewrites the source address (source NAT, or a proxy that opens its own connection onward) and every user now arrives from that one machine.

Look at that hop as a 4-tuple. Source IP is the load balancer, fixed. Destination IP is the backend, fixed. Destination port is 443, fixed. The only column left free is the load balancer's source port. That is exactly the shape of my loop. Around 64,000 connections to that backend and it cannot open another. People call it SNAT port exhaustion. Same rule, same number.

A load balancer that preserves the client address does not do this, because the source IP keeps varying.

<figure>
  <img src="/blog/direct-vs-lb.svg" alt="Left: phones A, B, and C connect straight to the server, each with its own source IP. Right: the same phones hit a load balancer that talks to the server as one IP, so that hop is limited to about 64k connections." width="720" height="280" />
  <figcaption>Figure 5. Straight to the server: src varies. Through a SNAT LB: src is one IP, 64k is back.</figcaption>
</figure>

The question is never "how many connections can a machine have." It is which of the four fields are pinned down, and which one is left to vary.

One client to one service: source port is the free column, so expect roughly 64,000. One service to many clients: source IP is the free column, and the table can be huge.

I still write load generators. I just do not use that number to argue about how many users a server can hold. I still raise the ephemeral port range and leave the file descriptor limit where it was, then hit the smaller wall and blame ports.

I am done treating 64,000 as a property of TCP itself.

If this is useful, wrong, or incomplete, write to me.
