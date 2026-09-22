---
title: "Binary search is a predicate, not a sorted array"
description: "A problem about bananas and a deadline had no sorted array anywhere in it, so I wrote a loop. The sorted thing was the answers, and I had not thought to look there."
pubDate: 2026-08-22
tags: ["dsa", "systems"]
---

Baski has four piles of bananas: 3, 6, 7, and 11. There are 8 hours before someone comes to collect them.

Baski picks an eating speed, some number of bananas per hour, and sticks with it. Each hour Baski chooses one pile and eats that many from it. If fewer than that remain, Baski eats what is left and the hour is still spent. An hour cannot be split across two piles. The question is the slowest speed that still clears every pile inside 8 hours.

I solved it by trying speeds. Speed 1, count the hours, too slow. Speed 2, count again. Keep going until one fits. That is correct, and it is fast enough for four small piles, and it falls over the moment the piles are large. What I did not notice was that I had written a linear scan over a sorted sequence.

## Why I did not see it

Binary search had arrived in my head as one picture. A sorted array, a target value, look at the middle, throw away the half that cannot contain it. Repeat.

That picture is correct. It is also one application of something smaller. My trigger for reaching for binary search was seeing a sorted array in the input. Bananas and a deadline contain no array. The piles are just four sizes. So the tool stayed in the drawer.

<figure>
  <img src="/blog/dsa-sorted.svg" alt="Two boxes: sorted array, find this number, versus a predicate ok of k, then search the first yes." width="720" height="240" />
  <figcaption>Figure 1. The array was never the point. The predicate was.</figcaption>
</figure>

## The sorted thing was the answers

Go back to what my loop was doing. For each candidate speed it asked one yes-or-no question: can Baski finish in time at this speed. A function that returns yes or no like that is a predicate. Call this one `ok(k)`.

For each pile, hours at speed `k` is the pile size divided by `k`, rounded up, because a partly-eaten pile still costs a full hour. Add those up. If the total is 8 or less, the answer is yes.

Try `k = 3`. The piles need `ceil(3/3) + ceil(6/3) + ceil(7/3) + ceil(11/3)` hours, which is `1 + 2 + 3 + 4`, so 10. Too slow. `ok(3)` is no.

Try `k = 4`. That is `1 + 2 + 2 + 3`, so exactly 8. `ok(4)` is yes.

Now the part I walked past. If speed 4 works, speed 5 works too, because eating faster can never make Baski finish later. If speed 3 fails, speed 2 fails as well. Laying the answers out in order of speed gives `no, no, no, yes, yes, yes...`, and it never goes back.

That property is called monotonic: once it turns to yes it stays yes. A monotonic sequence of yes and no is exactly what a sorted array is, viewed through the question "is this element at least the target." The sorted structure I had been waiting to be handed was sitting in the answers to my own loop.

<figure>
  <object class="figure-svg" data="/blog/dsa-predicate.svg" type="image/svg+xml" width="720" height="220" style="aspect-ratio: 720 / 220" aria-label="A row of no, no, no, then yes, yes, yes. The first yes is the answer.">
    <img src="/blog/dsa-predicate.svg" alt="A row of no, no, no, then yes, yes, yes. The first yes is the answer." width="720" height="220" />
  </object>
  <figcaption>Figure 2. You are not hunting an index. You are hunting the first yes.</figcaption>
</figure>

Once you can see that row, the job is no longer "find the slowest workable speed." It is "find where the row turns from no to yes." That is a binary search.

The range is speeds from 1 up to the largest pile. At that size every pile takes one hour, and nothing faster helps. Each probe evaluates `ok(mid)` and discards half of what is left. The first yes is the answer. For these piles it is 4.

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

`ok` still walks all four piles, exactly as my original loop did. The search does not make the walk faster. It changes how many times the walk runs, from once per possible speed down to about as many times as you can halve the range. For a range of a billion that is thirty.

## Three steps, in order

What I write down now, before touching any bounds:

**Name the predicate.** Write `ok(x)` as yes or no for a candidate answer. The candidate can be a speed, a size, a timeout. It only has to live on an ordered range.

**Say why a yes stays a yes.** If speed N finishes in time, speed N+1 finishes in time, because eating faster never makes you later. If the answers can flicker, there is no first yes to find and binary search will return nonsense.

**Then search for the first yes.** Low end at a known no, high end at a known yes, and halve.

Finding a number in a sorted array is this same frame with the predicate left unnamed: `ok(i)` is "is `a[i]` at least the target." The array was one way of getting monotonicity, not the requirement.

The shape shows up outside puzzles too. The smallest timeout that still covers the slowest one percent of requests. The fewest workers that still finish a batch before a deadline, provided adding a worker never makes it later. The cheapest delivery slot that still arrives before the festival sale ends. Whenever you can name the yes-or-no question and argue that a yes stays a yes, you can search the answer instead of walking to it.

I still only reach for binary search after seeing something sorted. I still force the frame onto something that is not monotonic, like "does this cache size improve the slow tail," because that answer can get worse before it gets better.

The question that survives is the same one. What am I allowed to ask about a candidate answer, and does a yes stay a yes.

If this is useful, wrong, or incomplete, write to me.
