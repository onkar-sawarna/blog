---
title: "I thought logs were observability"
description: "A pile of log lines can feel like seeing the system. It isn't. Observability is whether a UI can tell you what is sick, which machine, and since when, without grepping."
pubDate: 2026-08-15
tags: ["observability", "systems"]
---

For a long time, when something broke, I opened the logs.

That was the whole move. Find the service. Grep the error. Scroll until a line looked guilty. If the line was there, I had seen the system. If it wasn't, I added another print and waited for the next incident.

I would have called that observability. I was wrong.

## The wrong model

Logs feel like vision because they are made of words. A timeout has a sentence. A retry has a sentence. A connection that never came back has a sentence. After enough years in networking and security, you get fast at reading those sentences. You start to think the skill *is* the seeing.

It isn't. Logs are a record of what a process already decided to say. Someone, months ago, guessed which lines the next incident would need. You are reading that guess.

That distinction does not matter when the failure is one you have already had. It matters the first time the failure is new.

## Where it broke

The work I do now made this concrete.

You install an agent on a machine that is not yours (a customer's host), usually from the CLI. That agent watches two things. First, the software on their side that runs the cluster: what services it thinks are up, what it just restarted, what it is trying to launch. Second, the machines themselves: CPU, disk, memory, the processes that are actually alive. Then a UI has to turn that into a picture a human can use. Not a dump. A picture. Is the cluster healthy. Which host. Which service. Since when.

If all the agent did was ship log files off that box, the UI would be a search box. You would still be grepping. You would still not know whether one node is sick or the whole cluster is, whether the cluster manager is lying, or whether the manager is fine and a worker under it is dying quietly.

<figure>
  <img src="/blog/logs-agent.svg" alt="A customer host feeds an agent. The useful path is a UI that can say which host and since when. The failure path is a search box over shipped logs." />
  <figcaption>Same host. If the agent only ships logs, the UI is grep with a nicer font.</figcaption>
</figure>

The useful questions look like this:

- Is this one host or every host?
- Is the cluster manager healthy while the workers are not?
- Did this start in the last ten minutes, or have we been sliding for an hour?
- Is the box out of disk, or is a service just restarting in a loop?

A log line can tell you that a process exited. It will not tell you whether that exit is the disease or the sneeze. It will not tell you if the same exit is happening on every node or one. It will not tell you if the manager already tried to replace it, or if the disk filled up first and everything else is fallout.

I have sat in front of a perfectly healthy volume of logs and still not been able to answer "is this getting worse." That is the tell. If the UI cannot answer a question you did not plan for last quarter, you do not have a window. You have a diary with a nicer font.

<figure>
  <img src="/blog/logs-diary.svg" alt="A wall of timeout, retry, accept, deny next to a blank box for the question is this getting worse." />
  <figcaption>The pile can look complete and still leave the only useful question empty.</figcaption>
</figure>

Networking trained me to miss this. Tunnels, sessions, handshakes, firewalls: they all emit a lot of text. Text is comforting. You can have a wall of accept, deny, and timeout and still not know whether identity is wrong, the path is wrong, or the thing on the other side is just slow. A cluster is the same shape. The manager talks. The nodes talk. The UI has to decide which voice to trust.

## The model that stuck

Observability is not how much output you stored. It is whether someone can open a UI and find out what is wrong without SSHing onto the box.

Three kinds of signal, three jobs:

**Logs** tell you what one process thinks happened, once, with whatever context someone remembered to attach. Use them when you already know which process and which minute.

**Metrics** tell you the shape over time: how often, how wrong, how long, how full. Use them when the question is "are we sick, how much, since when, and is it one slice or all of them."

**Traces** tell you where a single request went. Use them when the question is "which hop ate the time."

The fourth thing is not a signal. It is **context**: host, service, cluster, version, tenant, so you can cut the other three without guessing. Without context, a metric is a number and a log is a short story with the names removed. The agent on the customer machine is useless if every line looks the same.

<figure>
  <img src="/blog/logs-signals.svg" alt="Four boxes: logs for one process once, metrics for shape over time, traces for which hop ate time, and context to cut the other three." />
  <figcaption>Three signals, three jobs. Context is how you cut. It is not a fourth pile of text.</figcaption>
</figure>

I still want logs. I want them less often, with more on the line, and I want them after a metric or a host list has pointed at a neighborhood. Opening the raw stream first is walking every street in a city because you do not have a map.

## How I recognize the old model now

I am in the old model when:

- The first move is grep, not "what am I trying to find out."
- The UI is a wall of counters nobody can explain.
- A new failure means a new log line, shipped in a hurry, as if the next incident will be kind enough to look the same.
- We can prove a process died and cannot say whether the cluster noticed.

The replacement habit is smaller than it sounds. Write the question down. Then pick the signal that can answer it. If nothing you collect can answer it, that is the gap, not a missing print.

<figure>
  <img src="/blog/logs-grep.svg" alt="Two boxes: grep first, then invent the question, versus write the question first, then pick the signal." />
  <figcaption>The old move starts at grep. The new one starts at the question.</figcaption>
</figure>

On a customer host that means: can the UI tell me the manager is up, the workers are up, and this one disk is the reason a service will not stay running. If I have to SSH to find that out, the agent did not finish the job.

## What I would still get wrong

A unique label for every user or every file on every metric. You will learn what a bill looks like, not what the system looks like.

Logging on the hot path "just in case." You will find that line when the disk is full, not when you need it.

Treating a pretty UI as understanding. A graph you cannot use to make a decision is decoration. Shipping an agent and calling it done is the same mistake at a larger scale.

I am not done being wrong about this. I am just done calling a stack of logs a window.

If this is useful, wrong, or incomplete, write to me. I would rather correct the model in public than keep the comfortable one.
