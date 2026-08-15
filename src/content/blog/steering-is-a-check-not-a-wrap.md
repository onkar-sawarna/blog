---
title: "I thought wrapping the packet was the hard part"
description: "Traffic steering looks like IP-in-IP. The useful work is what happens after you unwrap it: read the id, verify it, then decide whether the inner packet may continue."
pubDate: 2026-08-16
tags: ["networking", "systems"]
draft: true
---

I used to think traffic steering was an encapsulation problem.

You take a packet. You put it inside another packet. You send the outer one toward a path you chose. If the wrap is correct, the network will do the rest. That is a comfortable picture. It is also incomplete.

The part that mattered was not the wrap. It was the unwrap, the check, and the decision to let the inner packet continue.

## The wrong model

IP-in-IP looks like a pipe. Outer header says where this envelope is going. Inner header is the original packet. Strip the outer header, and the inner packet is "back on the network." A lot of tunnel diagrams stop there.

If you are steering traffic, the inner packet is not automatically yours to forward. Someone put an identifier in that envelope: which session, which client, which path this flow is supposed to be on. Call it a traffic id. It rides with the packet, inside the wrap, because the outer IP header does not know any of that.

The wrong model treats that id as decoration. The wrap is the mechanism. The id is metadata you might log. Forwarding is what IP already does.

## Where it broke

I was building a simulator for this path: many clients, steered traffic, no kernel TUN, no root. Each flow had to look like production. That meant the packet could not just arrive. Something had to open it and decide.

The shape is simple. The traffic id is wrapped inside IP-in-IP. A userspace netstack (gVisor) receives the outer packet, pulls the inner one out, and reads the id. Then it verifies. Is this an id we issued. Is this session still alive. Does this inner packet belong on this path. Only after that check does the stack let the packet pass.

<figure>
  <img src="/blog/steering-check.svg" alt="Four steps: wrap the packet with a traffic id in IP-in-IP, open it in a netstack, verify against a session table, then pass or drop." />
  <figcaption>Wrap, open, verify, then pass or drop. The check is the steering.</figcaption>
</figure>

If you skip the verify step, the simulator is a decapsulator. Any well-formed IP-in-IP frame becomes a real inner packet. That is not steering. That is a hole. Production would not let a random inner flow continue because the outer header was valid. The simulator should not either.

This is also why the work lived in userspace. A kernel interface will unwrap IP-in-IP and hand you a packet. It will not know your session table. The netstack is in-process, so the same code that allocated the id can see the packet, check it, and drop it before it ever looks like a normal IPv4 flow.

I have watched a packet survive the wrap and die on the check: outer header fine, inner header fine, id missing or stale. From the wire it looked like a tunnel. From the stack it was a stranger.

## The model that stuck

Steering is a checkpoint that happens to use a tunnel.

**Wrap** carries the original packet plus the id that says why this packet is on this path.

**Open** is not "strip four bytes and continue." It is parse the outer packet, recover the inner one, recover the id.

**Verify** is the actual policy. The id has to match a session you created. The inner addresses have to match what that session was allowed to send. A pretty envelope is not consent.

**Pass** is a decision. If the check fails, the inner packet does not get a second life as ordinary IP. It ends there.

Once I had that order in my head, a lot of "the tunnel is up" bugs got shorter. The tunnel being up means the outer path delivered a frame. It does not mean the inner flow was allowed.

## How I recognize the old model now

I am in the old model when:

- The first test is "can we encapsulate," and we never assert on the id after decap.
- A packet that unwraps cleanly is marked success.
- The session table and the netstack are treated as two different programs that happen to share a process.
- Someone says the path is down because IP-in-IP arrived, and we never ask whether the inner packet was allowed to leave.

The replacement habit is one question: after you open the envelope, what did you prove. If the answer is only "it was IP-in-IP," you have not steered anything.

## What I would still get wrong

Verifying the id format and not the binding. A well-shaped token that does not map to a live session should not pass.

Letting the inner packet into the stack before the check, then trying to filter it as a normal route. By then it already looks like traffic.

Treating the userspace stack as a convenience for "no root." The reason it is useful is that the check and the unwrap share memory. That is the point.

I am not done being wrong about tunnels. I am just done calling a wrap a steering decision.

If this is useful, wrong, or incomplete, write to me.
