---
title: "Binary search is a predicate, not a sorted array"
description: "The useful version is a yes-or-no on the answer. Name ok(x), prove a yes stays a yes, then search the first x that works."
pubDate: 2026-08-22
tags: ["dsa", "systems"]
---

Binary search usually arrives as one picture: a sorted array, a target, cut the middle, throw away a half.

That picture is true. It is also one use of a smaller idea. If nobody hands you a sorted list, it is easy to put the algorithm away.

<figure>
  <img src="/blog/dsa-sorted.svg" alt="Two boxes: sorted array, find this number, versus a predicate ok of k, then search the first yes." />
  <figcaption>The array was never the point. The predicate was.</figcaption>
</figure>

## The wrong model

A sorted array is a special case. The values go up. The question "is `a[i]` at least X" flips from no to yes once and stays yes. That flip is the whole trick.

Treat the array as the requirement and the lookup as the skill, and you miss every problem where the thing that is sorted is not the input. It is the answers to `ok(x)`.

That is most of the interesting ones.

## Where it broke

Baski has piles of bananas, a deadline in hours, and it does not look like search at all.

The piles are `3, 6, 7, 11`. Baski has 8 hours. Baski picks a speed k: bananas per hour. Each hour Baski picks one pile and eats k from it. If the pile has fewer than k left, Baski still burns the whole hour. An hour cannot split across piles. The job is the smallest k that finishes every pile before the deadline.

There is no sorted array of answers sitting in the input. The piles are just sizes. Linear scan does not tell you k. Trying every speed from 1 to the biggest pile works, and it is slow in the way that feels honest until the piles are huge.

Write the predicate first.

`ok(k)`: for each pile, add `ceil(pile / k)` hours. If the sum is at most 8, return yes.

Try k = 3.

`ceil(3/3) + ceil(6/3) + ceil(7/3) + ceil(11/3)` is `1 + 2 + 3 + 4` = 10. Ten hours. `ok(3)` is no.

Try k = 4.

`ceil(3/4) + ceil(6/4) + ceil(7/4) + ceil(11/4)` is `1 + 2 + 2 + 3` = 8. Eight hours. `ok(4)` is yes.

If 4 works, 5 works. Eating faster never makes Baski later. If 3 fails, 2 fails. The answers are `no, no, no, yes, yes, yes`. That is the only property the search needs.

<figure>
  <img src="/blog/dsa-predicate.svg" alt="A row of no, no, no, then yes, yes, yes. The first yes is the answer." />
  <figcaption>You are not hunting an index. You are hunting the first yes.</figcaption>
</figure>

The search space is the integers from 1 to the largest pile (eat that pile in one hour and anything smaller is free). Binary search those integers. Each probe is `ok(mid)`. The first k that returns yes is the answer. For this list it is 4.

```cpp
bool ok(int k, const vector<int>& piles, int h) {
    long long hours = 0;
    for (int pile : piles) {
        hours += (pile + k - 1LL) / k;
        if (hours > h) return false;
    }
    return true;
}

int min_speed(const vector<int>& piles, int h) {
    int lo = 1;
    int hi = *max_element(piles.begin(), piles.end());
    while (lo < hi) {
        int mid = lo + (hi - lo) / 2;
        if (ok(mid, piles, h)) hi = mid;
        else lo = mid + 1;
    }
    return lo;
}
```

`ok` is the walk. The loop is only how many times you run it. The input never needed to be sorted.

## The model that stuck

I keep a three-step frame. I write it down before I touch bounds.

**1. Name `ok(x)`.** A function of a candidate answer that returns yes or no. Indices, sizes, times, rates. The candidate has to live on an ordered range.

**2. Prove it is monotonic.** Once `ok` becomes yes, it stays yes. Or the other way. If it flickers, you do not have a search, you have a mess. Write one sentence: if N works, N+1 works, because...

**3. Search the first yes.** Low is the last no, or just below the range. High is a known yes, or just past the range. Each probe evaluates `ok(mid)` and throws away a half. The walk that implements `ok` can be linear. That is fine. You do log(range) walks, not a walk per possible x.

Finding a number in a sorted list is still this frame. `ok(i)` is "`a[i] >= target`". You do not have to wait for a sorted list to use it.

The same shape shows up outside puzzles. Smallest timeout that still covers a p99. Fewest workers that still finish before a deadline, if adding a worker never makes you later. If you can name `ok` and the yes stays a yes, you can search x.

## How I recognize the old model now

I am in the old model when:

- I only reach for binary search after I see `sort()`.
- I start writing a nested loop over possible sizes and have not named `ok`.
- I try to binary search an array that is not sorted and call the algorithm broken, instead of naming the predicate I actually have.
- I optimize the walk and never notice I am walking the same range the long way.

The replacement habit is the frame in order. Name `ok`. Say why a yes stays a yes. Then search. The input can stay messy.

## What I would still get wrong

Off-by-one on the first yes. Inclusive bounds, mid that never moves, a probe that returns no for the exact answer. I still draw the no/yes row when I am not sure.

Forcing a predicate onto a problem that is not monotonic. "Does this cache size reduce p99" is not `ok`. You can get worse, then better. Search will lie.

Treating the frame as a reason to skip the walk. The walk is `ok`. The search is only how many times you run it.

The useful question is still the same: what am I allowed to ask, and does a yes stay a yes.

If this is useful, wrong, or incomplete, write to me.
