---
title: "I thought ping meant reachable"
description: "ICMP coming back feels like the network is fine. It answers one question. The request you actually care about is usually a different path, a different port, and a different failure."
pubDate: 2026-08-16
tags: ["networking"]
draft: true
---

For a long time, when something could not connect, I pinged it.

If echo came back, the network was fine and the bug was in the app. If echo did not come back, the network was down and I started traceroute. That was the whole decision tree. I would have called that knowing how networks work. I was wrong.

## The wrong model

Ping is a gift. You send ICMP echo. You get ICMP echo reply. Round trip, one number, a yes or a no. After enough years of SSH and broken boxes, that yes starts to mean "the host is on the network." The no starts to mean "the path is dead."

That mapping is wrong in both directions.

ICMP is a control message. It is not the TCP handshake. It is not TLS. It is not the HTTP GET. Firewalls drop ping and still pass 443. Middleboxes answer ping on a box that is not the thing you think you reached. A host can echo and still refuse the port. A host can refuse ping and still serve the app.

Ping tells you whether that host, on that IP, was willing to answer that probe, from where you stood, right then. That is a real fact. It is a small fact.

## Where it broke

Networking work made this concrete. Tunnels, proxies, and anything that sits between a client and a service will happily lie to ping.

I have had ICMP succeed to an address and the TCP connect hang. The box was up. The path for echo was open. The path for that port was not: a security group, a proxy ACL, a listener that was bound to localhost, a tunnel that forwarded ICMP and did not forward the session.

<figure>
  <img src="/blog/ping-split-anim.svg" alt="A packet completing an ICMP echo to the host while a TCP connection to port 443 stops at a filter." />
  <figcaption>Echo comes back. The connect to 443 does not. Same IP.</figcaption>
</figure>

I have had ICMP fail and the service work. Someone blocked echo and left 443 alone. I spent time on a "down" host that was taking traffic.

I have had ping hit a different machine than the request. NAT, anycast, a load balancer that answers ICMP on a VIP and sends the real connection somewhere else. The echo reply was honest. It was honest about the wrong question.

The useful questions look like this:

- Can I get an echo from this IP.
- Can I complete a TCP handshake on this port.
- Can I finish the handshake the app uses (TLS, a proxy CONNECT, a tunnel).
- Can the request the user sent return a useful response.

Those are four checks. They fail independently. Treating the first one as the last one is how you debug the wrong layer for an hour.

<figure>
  <img src="/blog/ping-layers.svg" alt="Four stacked checks: request, session, TCP port, and ICMP. Ping only lives on the bottom layer." />
  <figcaption>Ping lives on L3. The user cares about the top of the stack.</figcaption>
</figure>

Traceroute has the same trap. It shows you where ICMP or UDP probes started getting lost. It does not show you the path the TCP session took, and it does not show you a middlebox that only interferes with the application protocol.

## The model that stuck

Reachability is not one bit. It is a stack of questions, and you have to name which one you asked.

**L3.** Did a packet with this destination IP get a reply of some kind. Ping lives here. So does "the route exists."

**L4.** Did this port accept a connection from here. `connect()` hanging or resetting is a different outage than a missing echo.

**The session on top.** Did TLS finish. Did the proxy accept the user. Did the tunnel assign a session. A SYN-ACK is not a login.

**The request.** Did the thing we called the service do the work and respond. A 200, a reset, a timeout after the handshake: three different bugs.

When I say "the network is down" now, I try to replace it with one of those. "ICMP to that IP fails from this host." "TCP to 443 times out." "Handshake works, first byte never arrives." Those sentences point at a layer. "The network is down" points at a feeling.

## How I recognize the old model now

I am in the old model when:

- The first command is ping, and I stop if it works.
- I tell someone the host is up because echo returned, and I have not opened the port.
- I tell someone the host is down because echo failed, and I have not tried the port the app uses.
- Traceroute looks ugly and I treat that as proof the application path is the same.

The replacement habit is small. Write the question: what has to succeed for this user. Then probe that. Ping is allowed. It is a first check, not a verdict.

## What I would still get wrong

Pinging a name and trusting the IP I get back without asking who answered. DNS and ping together can hide a stale record or a VIP.

Checking from my laptop and calling the path good. The client that failed is often on a different network, behind a different proxy, with a different MTU.

Confusing "I got a reply" with "I reached the process." A lot of infrastructure will answer for a box that is not running the thing you wanted.

I am not done being wrong about this. I am just done treating echo reply as the whole path.

If this is useful, wrong, or incomplete, write to me.
