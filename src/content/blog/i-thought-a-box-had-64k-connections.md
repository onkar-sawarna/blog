---
title: "I thought a box could only hold 64k connections"
description: "From one machine to one other machine, you often get about 64k TCP connections. That is a source-port limit, not a law of TCP. A server facing many clients is a different table."
pubDate: 2026-08-16
tags: ["networking", "systems"]
---

I used to treat 65,535 as a hard ceiling on TCP.

One machine, 64k connections, full stop. Then you hear a chat service talk about millions of connections on a single box and the number feels like a lie. It is not a lie. It is a different counting problem.

## The wrong model

A port is 16 bits. 0 to 65535. You subtract the reserved ones and you get "about 64k." If you think a connection *is* a port, the math is done. A machine has 64k ports, so it has 64k connections.

That model is what you get from looking at a client. You open a socket to `m2:443`. The kernel picks an ephemeral source port, say 49152. The next connection to the same place gets 49153. When the ephemeral range is exhausted, `connect()` fails: cannot assign requested address. From `m1` to that one IP and port on `m2`, you really are in the 64k neighborhood. TIME_WAIT makes it worse. Ports sit occupied after you close.

The mistake is taking that local pain and calling it TCP.

## Where it broke

A TCP connection is not a port. It is a 4-tuple:

`source IP, source port, dest IP, dest port`

The kernel looks up a packet by all four. Two connections can share a dest port. They can share a dest IP. They cannot share the whole tuple.

So the 64k shows up in a specific shape: **one source IP, one dest IP, one dest port.** The only free variable is the source port. That is `m1` opening a pile of connections to `m2:443`. One client, one peer, one service port. About 64k live tuples, then you are done.

<figure>
  <img src="/blog/64k-one-client.svg" alt="m1 opening many connections to m2 on port 443. Only the source port changes, so the table tops out around 64k." />
  <figcaption>m1 to m2:443. Three fields fixed. Source port is the only knob.</figcaption>
</figure>

Change any one field and the table grows.

- `m1` has a second address. You get another 64k to the same `m2:443`.
- `m2` listens on a second port. Another 64k.
- The dest is a VIP that fans out. You are no longer talking about one box in the way you thought.

Now flip the roles. `m2` is a server. It listens on `443`. Clients are phones and laptops, each with their own IP, each picking their own source port. The tuples look like this:

- `1.2.3.4:51000 → m2:443`
- `5.6.7.8:51000 → m2:443`
- `9.9.9.9:44321 → m2:443`

Same dest IP. Same dest port. Different source IPs. The server is not spending its own ephemeral ports to accept these. It is holding a row per client tuple. The 16-bit port field lives on the client side of each row. It does not cap the number of rows.

That is how a box can have millions of connections. Millions of peers, one listen port, one connection table, if the OS and the process can hold the sockets.

<figure>
  <img src="/blog/64k-many-clients.svg" alt="Many phones and laptops connecting to one server on port 443. Each client has its own source IP, so the server table can grow past 64k." />
  <figcaption>The server is m2. Each client is a different source IP. 64k is not the cap.</figcaption>
</figure>

The other ceiling is not 64k. It is file descriptors, memory per socket, and how the runtime waits on them. `ulimit -n` at 1024 will stop you long before ports do. A server that keeps a buffer and a timer per connection will run out of RAM. The interesting engineering is that, not the 16-bit myth.

## The WhatsApp number

WhatsApp has talked about millions of connections on a server. Read that as `m2`, not `m1`.

Five million users are not one machine opening five million sockets to one peer. Five million clients each open one (or a few) long-lived connections to a server. The server has five million entries: many source IPs, many source ports, one dest. The 64k rule never applied to that table.

They still have to win the real limits: a kernel that can keep that many sockets, a process that does not wake the world on every packet, and enough RAM that a quiet connection is cheap. That is a server problem. It is not a port-space problem.

If you try to reproduce "5 million connections" by looping `connect()` from one host to one host, you will hit ~64k and think the claim is fake. You built the `m1 → m2:port` case. They published the other case.

## The model that stuck

Ask which side you are on, and which fields are fixed.

**One client to one service.** Source port is the scarce thing. Expect ~64k. Multiple local IPs or multiple dest ports if you actually need more.

**One service to many clients.** Source IP is the thing that multiplies. The listen port is shared. The table can be huge.

**The 4-tuple is the identity of a connection.** A port is just one column.

I still ping. I still watch ephemeral ports when I write a load generator. I do not use that number to argue about how many users a server can hold.

## How I recognize the old model now

I am in the old model when:

- I say "TCP only allows 64k connections" with no 4-tuple.
- A load test from one box to one box is treated as the server's capacity.
- TIME_WAIT is a mystery instead of ports you have not given back yet.
- A million-connection claim sounds like a protocol violation instead of many peers.

The replacement habit is one sentence: which three fields are fixed. If source IP, dest IP, and dest port are fixed, you are in 64k land. If the source IP keeps changing, you are not.

## What I would still get wrong

Forgetting IPv6 and extra addresses. More local IPs, more tuples.

Forgetting that "one connection per user" is a product choice. Some clients open more. Some share. The server table is still counted in sockets, not in marketing users.

Tuning the ephemeral range and never raising file descriptors. You will hit the smaller wall and blame ports.

I am not done being wrong about sockets. I am just done treating 64k as a property of TCP itself.

If this is useful, wrong, or incomplete, write to me.
