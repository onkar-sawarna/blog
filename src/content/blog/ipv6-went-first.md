---
title: "IPv6 went first"
description: "The shop feels slow. The API is fast. The phone tried IPv6, got no reply, then used IPv4."
pubDate: 2026-09-29
tags: ["networking", "systems"]
draft: false
---

A buyer opens the shop and taps a pair of size-8 white sneakers. The page spins long enough that I start looking at Redis. When the request finally reaches my API, it finishes in 40 milliseconds. The code is fine. The wait happened before the request arrived.

I used to call that a network problem in general. It was more specific: the phone tried one address, that address did not answer, and only then did it try the other.

<figure>
  <img src="/blog/v6-spinner.svg" alt="Phone spinner feels slow. API log shows GET sneakers in 40ms." width="720" height="280" />
  <figcaption>Figure 1. I blamed the API. The wait was before the GET.</figcaption>
</figure>

## Two addresses for one shop

The phone asks DNS for the shop name. DNS can send back two answers. An A record is an IPv4 address, the older 32-bit kind. A AAAA record is an IPv6 address, the newer longer kind. I had published both.

The phone now has two places it can open a TCP connection. TCP is the byte stream the page uses. It does not pick at random. It tries IPv6 first.

It tries IPv6 at all for two reasons. The carrier already gave the phone its own IPv6 address, because there are not enough public IPv4 addresses for every SIM. And I published a AAAA. If I had not, the phone would only have the IPv4 address, and this wait would not happen.

When both answers exist, the phone's software is built to prefer IPv6. The idea is that if you advertised IPv6, that should be the normal path. I turned that on when I added the AAAA record.

<figure>
  <img src="/blog/v6-dns.svg" alt="DNS returns A for IPv4 and AAAA for IPv6. One path works. One is a hole." width="720" height="280" />
  <figcaption>Figure 2. One name. Two addresses. The phone tries IPv6 first.</figcaption>
</figure>

My office laptop used IPv4 only, and the shop was fast. The buyer's phone used the carrier's IPv6. The AAAA pointed at a machine that does not actually accept IPv6, or a filter drops those packets. From the buyer, it is the same: no reply.

## The first SYN gets no answer

The phone sends a SYN to the IPv6 address. SYN is the first packet of a TCP connection: "I want to start." Nothing comes back. The packet is gone. The phone waits.

Then it tries IPv4. It gets a reply, finishes the secure handshake, and sends the GET. That part takes 40 milliseconds. The buyer felt the wait on IPv6. They did not feel the API.

<figure>
  <img src="/blog/v6-syn-hole.svg" alt="IPv6 SYN gets no SYN-ACK. IPv4 SYN reaches the API." width="720" height="280" />
  <figcaption>Figure 3. Same shop. Two addresses. One of them never answers.</figcaption>
</figure>

That is why a ping to the IPv4 address, or a laptop on Wi-Fi that has no IPv6, can look fine while the phone is still slow.

## Trying both, if the client does it

Some clients do not wait for IPv6 to time out. They start IPv6, then start IPv4 after a short pause if IPv6 has not connected, and keep the first one that works. That race is called Happy Eyeballs.

A current phone browser often does this, so the extra wait is small. An old library or a small program that only calls `connect()` on the first address will wait much longer. Those are the reports that say the API is slow when the API log never moved.

<figure>
  <img src="/blog/v6-race.svg" alt="Try IPv6 first, then IPv4, keep the first connect. Happy Eyeballs." width="720" height="300" />
  <figcaption>Figure 4. Try both. Keep the one that connects.</figcaption>
</figure>

I still add a AAAA because a checklist is green. I do not test that address from a phone path before I publish it. That is how a dead IPv6 record reaches buyers.

## What I took from this

The shop can be fast and still feel slow. The wait is often the first address the phone tried, not the query.

I still get this wrong. I read API logs and ignore AAAA. I test from a laptop that never uses IPv6. I assume the phone will always race both addresses. I add IPv6 to look complete and leave a record that does not answer.

The story about how many connections one laptop can open is [here](/blog/i-thought-a-box-had-64k-connections/). This one is the extra address in DNS, and the first SYN that never came back.
