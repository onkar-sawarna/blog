---
title: "I thought a box could only hold 64k connections"
description: "From one machine to one other machine, you often get about 64k TCP connections. That is a source-port limit, not a law of TCP. A server facing many clients is a different table."
pubDate: 2026-08-16
tags: ["networking", "systems"]
---

I used to treat 65,535 as a hard ceiling on TCP.

One machine, 64k connections, full stop. Then you hear a chat service talk about millions of connections on a single box and the number feels like a lie. It is not a lie. It is a different counting problem.

<figure>
  <img src="/blog/64k-loadtest.svg" alt="One laptop looping connect hits 64k and calls the claim fake. The other box: you fixed src, they did not." />
  <figcaption>The load test did not disprove the server. It turned you into one client.</figcaption>
</figure>

## The wrong model

A port is 16 bits. 0 to 65535. You subtract the reserved ones and you get "about 64k." If you think a connection *is* a port, the math is done. A machine has 64k ports, so it has 64k connections.

That model is what you get from looking at a client. You open a socket to `m2:443`. The kernel picks an ephemeral source port, say 49152. The next connection to the same place gets 49153. When the ephemeral range is exhausted, `connect()` fails: cannot assign requested address. From `m1` to that one IP and port on `m2`, you really are in the 64k neighborhood. TIME_WAIT makes it worse. Ports sit occupied after you close.

The mistake is taking that local pain and calling it TCP.

## Where it broke

A TCP connection is not a port. It is a 4-tuple:

`source IP, source port, dest IP, dest port`

The kernel looks up a packet by all four. Two connections can share a dest port. They can share a dest IP. They cannot share the whole tuple.

This whole argument assumes the NIC is fine, the CPU is fine, and the link is not full. No hardware ceiling. No bandwidth ceiling. We are only asking how many distinct TCP 4-tuples the stack will allow. In a real datacenter you often hit RAM, file descriptors, or the wire first. That is a different outage. Here those are off the table on purpose.

So the 64k shows up in a specific shape: **one source IP, one dest IP, one dest port.** The only free variable is the source port. That is `m1` opening a pile of connections to `m2:443`. One client, one peer, one service port. About 64k live tuples, then you are done.

<figure>
  <img src="/blog/64k-one-client.svg" alt="m1 opening many connections to m2 on port 443. Only the source port changes, so the table tops out around 64k." />
  <figcaption>m1 to m2:443. Three fields fixed. Source port is the only knob.</figcaption>
</figure>

You can see the tuple on a Linux machine. Conntrack prints it. That is the point of the tool.

```
conntrack -L
```

Or, if the tool is not installed:

```
cat /proc/net/nf_conntrack
```

Each line is one 4-tuple: `src`, `sport`, `dst`, `dport`, plus a state. Same four fields the kernel uses to look up the packet. Point it at `m2` and port 443. You will watch `sport` climb while the other three stay put. ESTABLISHED lines are sockets you still hold. TIME_WAIT lines are sockets you already closed. Those still count. The port is not free for a new connect to the same dest until that line is gone.

When `sport` has nowhere left to go, the next `connect()` from `m1` to `m2:443` fails. That is the 64k showing up in a command you can type.

A NAT box is the same program, same command, one public source IP. `conntrack -L` on the NAT is how you watch it run out of ports for that dest.

<figure>
  <img src="/blog/conntrack-tuple.svg" alt="A Linux box and a conntrack -L listing. Each line is one 4-tuple: src, sport, dst, dport." />
  <figcaption>conntrack -L is the 4-tuple, one line per flow. src, sport, dst, dport.</figcaption>
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

That is how a box can have millions of connections. Millions of peers, one listen port, one connection table, if the OS and the process can hold the sockets. Still assuming the NIC and the link are not the thing that gives out first.

<figure>
  <img src="/blog/64k-many-clients.svg" alt="Many phones and laptops connecting to one server on port 443. Each client has its own source IP, so the server table can grow past 64k." />
  <figcaption>The server is m2. Each client is a different source IP. 64k is not the cap.</figcaption>
</figure>

The other ceiling is not 64k. It is file descriptors, memory per socket, and how the runtime waits on them. `ulimit -n` at 1024 will stop you long before ports do. A server that keeps a buffer and a timer per connection will run out of RAM. The interesting engineering is that, not the 16-bit myth.

## The WhatsApp number

WhatsApp has talked about millions of connections on a server. Read that as `m2`, not `m1`.

In simple words: the source IP is itself a variable. An IPv4 address is 32 bits. That is `2^32` possible source IPs, not one. Each phone is a different `src`. The dest IP and dest port on the server stay the same. The 64k only appears when `src` is fixed and only `sport` can change. Here `src` keeps changing, so the 64k cap never applies.

Five million users are five million source IPs (or fewer NATs, still many). Each opens one connection. The server holds five million tuples. `conntrack -L` on that box would show many different `src` values, not one host burning through ports.

They still have to win RAM, file descriptors, and the runtime. That is a server problem. It is not a port-space problem. Same assumption as above: we are not talking about the NIC or the link giving out.

That only holds if the phone talks straight to the server. The server sees the phone's own source IP. `src` stays a variable.

Put a load balancer in the middle that SNATs, or a proxy that opens its own sockets to one backend IP:port, and you collapsed `src` again. Every user looks like the LB. Dest IP and dest port on the server are fixed. Only `sport` on the LB-to-server hop can change. That hop is `m1 → m2:443` with a new name. About 64k, then the LB cannot open another connection to that backend. People call it SNAT port exhaustion. It is the same 4-tuple rule.

An LB that does not rewrite the client IP (the server still sees A, B, and C) does not put you back in 64k land. The dangerous LB is the one that makes everyone look like one address.

<figure>
  <img src="/blog/whatsapp-vs-lb.svg" alt="Left: phones A, B, and C connect straight to the server, each with its own source IP. Right: the same phones hit a load balancer that talks to the server as one IP, so that hop is limited to about 64k connections." />
  <figcaption>Straight to the server: src varies. Through a SNAT LB: src is one IP, 64k is back.</figcaption>
</figure>

If you try to reproduce "5 million connections" by looping `connect()` from one host to one host, you will hit ~64k and think the claim is fake. You fixed the source IP. They did not.

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
- TIME_WAIT is a mystery instead of a line that `conntrack -L` still prints after I close.
- A million-connection claim sounds like a protocol violation instead of many peers.

The replacement habit is one sentence: which three fields are fixed. If source IP, dest IP, and dest port are fixed, you are in 64k land. If the source IP keeps changing, you are not.

## What I would still get wrong

Forgetting IPv6 and extra addresses. More local IPs, more tuples.

Forgetting that "one connection per user" is a product choice. Some clients open more. Some share. The server table is still counted in sockets, not in marketing users.

Tuning the ephemeral range and never raising file descriptors. You will hit the smaller wall and blame ports.

Tuning sysctls and never running `conntrack -L` on the box that is failing.

I am not done being wrong about sockets. I am just done treating 64k as a property of TCP itself.

If this is useful, wrong, or incomplete, write to me.
