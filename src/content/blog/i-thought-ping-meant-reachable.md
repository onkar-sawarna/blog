---
title: "I thought ping meant reachable"
description: "A reply came back in eleven milliseconds, so I said the network was fine. It was not. Ping answers one small question, and the request you care about is a different path and a different failure."
pubDate: 2026-08-16
tags: ["networking"]
draft: true
---

Someone reported that a client could not connect to a service. I did what I always did. I pinged the host.

The reply came back in eleven milliseconds. So I told them the network was fine and the bug was in their application, and I went and looked at the application, where I found nothing, for about an hour.

## Eleven milliseconds of the wrong information

Ping sends a small message called an ICMP echo request. ICMP is the protocol the network uses to talk about itself, for things like "this host is unreachable" or "your packet was too big." An echo request means, roughly, say something back. If the other machine is willing, it sends an echo reply, and ping prints how long the round trip took.

That is genuinely useful. It is also a much smaller fact than I was treating it as. After enough years of broken boxes and dropped SSH sessions, a reply had come to mean "this host is on the network and healthy," and silence had come to mean "the path is dead." Both of those readings are wrong, and they are wrong in opposite directions.

The echo reply told me that this host, at this address, was willing to answer that particular kind of probe, from where I was standing, at that moment. Nothing in it said anything about port 443.

<figure>
  <img src="/blog/ping-verdict.svg" alt="Ping ok leads to the verdict that the network is fine, while the user is still broken. The other box names the layer: ICMP is not port 443." width="720" height="240" />
  <figcaption>Echo is a small fact. Treating it as the outage is the old model.</figcaption>
</figure>

## What I should have typed second

Eventually I stopped reading application code and tried to open a TCP connection to the port the client actually used. It hung. No refusal, no error, just nothing, until it timed out.

That is a completely different signal from a failed ping, and it points somewhere completely different. A hang usually means a firewall rule is dropping the packets on the floor rather than answering, which is what a security group or a proxy access list does by default. A refusal, where the connection comes back immediately with a reset, means something answered and said no, which usually means nothing is listening on that port. In this case it was a listener bound to the loopback address, so the process was up and serving, but only to things running on that same machine. From outside, the port was a wall. From ping's point of view, the host was perfect.

<figure>
  <img src="/blog/ping-split-anim.svg" alt="A packet completing an ICMP echo to the host while a TCP connection to port 443 stops at a filter." width="720" height="260" />
  <figcaption>Echo comes back. The connect to 443 does not. Same IP.</figcaption>
</figure>

Once I started looking for it, the same gap showed up in both directions.

I have wasted time on a host that was "down" because echo failed, while it was quietly serving production traffic on 443 the whole time. Somebody had blocked ICMP and left the application port alone, which is a very common and entirely reasonable thing to do.

I have also had ping reach a different machine than the request did. When several servers sit behind one address, whether through a load balancer, a NAT device, or an anycast setup where the same address is announced from multiple locations, the thing that replies to echo is not necessarily the thing that will terminate your connection. The reply was honest. It was honest about a different machine.

## Four questions that fail separately

What I had been doing was collapsing a stack of independent questions into a single yes or no. Written out, the questions a user's request has to pass are:

- Can a packet reach this address and get a reply of any kind.
- Can I complete a TCP connection to the specific port.
- Can I complete whatever handshake sits on top, such as a TLS negotiation or a proxy accepting me.
- Can the actual request come back with a useful answer.

Each one can fail while the ones below it succeed. Ping only ever answers the first.

<figure>
  <img src="/blog/ping-layers.svg" alt="Four stacked checks: request, session, TCP port, and ICMP. Ping only lives on the bottom layer." width="720" height="280" />
  <figcaption>Ping lives on L3. The user cares about the top of the stack.</figcaption>
</figure>

The third one catches people more than it should. A TCP connection succeeding means the two machines agreed to talk. It does not mean TLS negotiated a cipher both sides accept, and it does not mean the proxy in the middle decided you were allowed through. A completed handshake is not a login.

Traceroute has the same trap in a longer form. It shows you where its own probes stopped getting answers, which is not necessarily where your connection stopped, and it will not show you a device that passes packets fine but interferes with the application protocol on top of them.

## Saying which layer

The habit that replaced the old one is mostly about language. "The network is down" is a feeling. It does not tell anyone where to look, and it does not commit me to anything I can be wrong about.

So now I try to say one of these instead. ICMP to that address fails from this host. TCP to 443 from this host times out. The connection completes and TLS never finishes. The request returns a 200 but the body is empty. Each of those sentences names a layer and suggests the next command, and each of them is specific enough that somebody can tell me I am wrong.

Ping is still the first thing I type. It is just a first check now, not a verdict, and when it comes back clean I keep going rather than closing the tab.

I still make the smaller versions of this mistake. I ping a name rather than an address and trust whatever DNS handed me, without asking which machine actually answered. I test from my own laptop and declare the path good, when the client that failed sits on a different network, behind a different proxy, with a different maximum packet size. And I still occasionally read "I got a reply" as "I reached the process," when a great deal of infrastructure will happily answer on behalf of a box that is not running the thing I wanted.

I am not done being wrong about this. I am just done treating an echo reply as the whole path.

If this is useful, wrong, or incomplete, write to me.
