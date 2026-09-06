---
title: "I thought wrapping the packet was the hard part"
description: "My first traffic simulator unwrapped every packet perfectly and forwarded all of them. That is not steering, it is a hole. The work is what you prove after you open the envelope."
pubDate: 2026-08-16
tags: ["networking", "systems"]
draft: true
---

I needed a simulator for a steered-traffic path: many clients at once, each one's traffic sent down a chosen route, running as an ordinary process with no root and no kernel network device to lean on.

The first version worked on the second afternoon. Packets went in wrapped, came out unwrapped, and got forwarded. I was pleased with it for about a day, until I realised that what I had built was not a simulator. It was a machine that would forward anything.

## Putting a packet inside a packet

The mechanism is called encapsulation, and the specific flavour here is IP-in-IP. You take a finished IP packet, the one the client meant to send, and you treat the whole thing as payload inside a second IP packet with its own header on the front.

The header is the small block of fields at the start of a packet that says where it came from and where it is going. So the outer header is addressed to wherever you decided this traffic should go, and the inner packet, untouched, is carried along inside. At the far end something strips the outer header off and you are left holding the original.

Alongside the inner packet there is an identifier. Call it a traffic id. It says which session this flow belongs to, which client, and which path it was assigned. It has to ride inside the envelope, because the outer IP header has nowhere to put any of that. Addresses are all it knows about.

My mental model of the whole system was: the wrap is the mechanism, the id is a label I might log somewhere, and forwarding is what IP does anyway. Get the encapsulation right and the network handles the rest.

<figure>
  <img src="/blog/steering-stranger.svg" alt="Tunnel is up, outer header fine, id stale or missing. After the unwrap you prove the session or it is a stranger." width="720" height="240" />
  <figcaption>The wrap succeeded. The packet was still not allowed to continue.</figcaption>
</figure>

## The version that forwarded everything

Here is what my working simulator did with an arriving packet. Parse the outer header. Pull the inner packet out. Forward the inner packet.

Read it back and the problem is right there in the middle. Nothing between step two and step three consulted anything. Any well-formed IP-in-IP frame from anywhere would be unwrapped and its contents released as a real flow. The id was in the envelope and I was not reading it.

That is not a simulator with a missing feature. It is a different program. Production would never let an inner flow continue just because the outer header parsed, and a simulator that does is not reproducing the path, it is reproducing a hole.

The step I had left out is a check against a session table, which is just the in-memory record of the sessions this system handed out: which ids are live, which client each belongs to, which addresses that client was permitted to send from. So the real sequence has four steps rather than three, and the third one is the point of the whole exercise.

<figure>
  <img src="/blog/steering-check.svg" alt="Four steps: wrap the packet with a traffic id in IP-in-IP, open it in a netstack, verify against a session table, then pass or drop." width="720" height="240" />
  <figcaption>Wrap, open, verify, then pass or drop. The check is the steering.</figcaption>
</figure>

**Wrap** carries the original packet plus the id that explains why this packet is on this path.

**Open** is not "strip the outer header and continue." It is parse the outer packet, recover the inner one, and recover the id that came with it.

**Verify** is where the policy lives. Does this id match a session we issued. Is that session still alive. Do the addresses on the inner packet match what the session was allowed to send. A well-formed envelope is not consent.

**Pass** is a decision with a real alternative. If the check fails, the inner packet does not get released into the network as ordinary traffic. It ends there.

Once the check was in, I started seeing packets fail in a way I had not had a category for. Outer header fine. Inner header fine. Id present but belonging to a session that had already ended. From the wire that packet looked like a healthy tunnel. From inside the stack it was a stranger with an expired badge, and it got dropped.

## Why this could not live in the kernel

The no-root constraint turned out to be the smaller reason for the design.

If you ask the kernel to handle IP-in-IP, it will do the unwrapping for you and hand you a packet. What it cannot do is consult your session table, because that table lives in your process and means nothing to the kernel. You would end up unwrapping in one place and checking in another, with the packet already loose in between.

So the network stack runs in the process instead, using gVisor's userspace networking, which implements the IP and TCP handling in ordinary Go rather than in the kernel. The code that issued the id and the code that receives the packet are in the same memory. The check happens before the inner packet is ever a routable thing. Not having to be root was a convenience. Sharing memory with the session table was the actual argument.

## What I ask now

The question I ask about any tunnel is: after you opened the envelope, what did you prove. If the only answer is "it was well-formed IP-in-IP," nothing has been steered. Something has been forwarded.

That question also fixed a category of bug report. "The tunnel is up" means the outer path delivered a frame. It does not mean the inner flow was allowed to continue, and those two are usually investigated by different people looking at different things.

I still catch myself in the old version of this. Checking that an id parses rather than that it binds to a live session, which lets a well-shaped token through. Letting the inner packet into the stack first and planning to filter it later with ordinary routing rules, by which point it already looks like legitimate traffic. And describing the userspace stack as a workaround for not having root, when the reason it is the right answer is that the check and the unwrap can see the same memory.

I am not done being wrong about tunnels. I am just done calling a wrap a steering decision.

If this is useful, wrong, or incomplete, write to me.
