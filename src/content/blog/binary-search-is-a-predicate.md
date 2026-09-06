---
title: "Binary search is a predicate, not a sorted array"
description: "A problem about bananas and a deadline had no sorted array anywhere in it, so I wrote a loop. The sorted thing was the answers, and I had not thought to look there."
pubDate: 2026-08-22
tags: ["dsa", "systems"]
---

Baski has four piles of bananas: 3, 6, 7, and 11. There are 8 hours before someone comes to collect them.

Baski picks an eating speed, some number of bananas per hour, and sticks with it. Each hour Baski chooses one pile and eats that many from it. If fewer than that remain in the pile, Baski eats what is left and the hour is still spent, because an hour cannot be split across two piles. The question is the slowest speed that still clears every pile inside 8 hours.

I solved it by trying speeds. Speed 1, count the hours, too slow. Speed 2, count again. Keep going until one fits. That is correct, and it is fast enough for four small piles, and it falls over the moment the piles are large. What I did not notice, for an embarrassingly long time, was that I had just written a linear scan over a sorted sequence.

## Why I did not see it

Binary search had arrived in my head as one picture and never left it. A sorted array, a target value, look at the middle, throw away the half that cannot contain it. Repeat.

That picture is correct. It is also one application of something smaller, and because I had learned the application rather than the idea, my trigger for reaching for binary search was seeing a sorted array in the input. Bananas and a deadline contain no array. The piles are just four sizes in no particular order, and sorting them tells you nothing useful. So the tool stayed in the drawer.

<figure>
  <img src="/blog/dsa-sorted.svg" alt="Two boxes: sorted array, find this number, versus a predicate ok of k, then search the first yes." width="720" height="240" />
  <figcaption>The array was never the point. The predicate was.</figcaption>
</figure>

## The sorted thing was the answers

Go back to what my loop was doing. For each candidate speed it asked one yes-or-no question: can Baski finish in time at this speed. A function that returns yes or no like that is called a predicate, and it is worth giving this one a name, `ok(k)`, so it can be talked about separately from the searching.

Writing it out for this problem: for each pile, work out how many hours it takes at speed `k`, which is the pile size divided by `k` and rounded up, since a partly-eaten pile still costs a full hour. Add those up. If the total is 8 or less, the answer is yes.

Try `k = 3`. The piles need `ceil(3/3) + ceil(6/3) + ceil(7/3) + ceil(11/3)` hours, which is `1 + 2 + 3 + 4`, so 10 hours. Too slow. `ok(3)` is no.

Try `k = 4`. That is `ceil(3/4) + ceil(6/4) + ceil(7/4) + ceil(11/4)`, which is `1 + 2 + 2 + 3`, so exactly 8. `ok(4)` is yes.

Now the part I had walked straight past. If speed 4 works, speed 5 works too, because eating faster can never make Baski finish later. And if speed 3 fails, speed 2 fails as well, for the same reason in reverse. So laying the answers out in order of speed gives `no, no, no, yes, yes, yes, yes...`, and it never goes back.

That property has a name, monotonic, which just means the sequence only moves one way: once it turns to yes it stays yes. And a monotonic sequence of yes and no is exactly what a sorted array is, viewed through the question "is this element at least the target." The sorted structure I had been waiting to be handed was sitting in the answers to my own loop the whole time.

<figure>
  <img src="/blog/dsa-predicate.svg" alt="A row of no, no, no, then yes, yes, yes. The first yes is the answer." width="720" height="220" />
  <figcaption>You are not hunting an index. You are hunting the first yes.</figcaption>
</figure>

Once you can see that row, the job is no longer "find the slowest workable speed." It is "find where the row turns from no to yes," and that is a binary search.

The range to search over is the speeds from 1 up to the largest pile, because at the largest pile's size every pile takes one hour and nothing faster helps. Each probe evaluates `ok(mid)` and discards half of what is left. The first yes is the answer, and for these piles it is 4.

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

Notice how little of that is the clever bit. `ok` still walks all four piles, one at a time, exactly as my original loop did. The search does not make the walk faster. It changes how many times the walk runs, from once per possible speed down to about as many times as you can halve the range, which for a range of a billion is thirty.

## Three steps, in order

What I write down now, before touching any bounds, is three things.

**Name the predicate.** Write `ok(x)` as a function of a candidate answer that returns yes or no. The candidate can be an index, a size, a duration, a rate. It only has to live somewhere on an ordered range.

**Say why a yes stays a yes.** One sentence, out loud: if speed N finishes in time, speed N+1 finishes in time, because eating faster never makes you later. If the answers can flicker between yes and no, there is no first yes to find and binary search will confidently return nonsense.

**Then search for the first yes.** Set the low end at a known no or just below the range, the high end at a known yes or just past it, and halve.

Finding a number in a sorted array turns out to be this same frame with the predicate left unnamed: `ok(i)` is "is `a[i]` at least the target," and it is monotonic because the array is sorted. The array was one way of getting monotonicity, not the requirement.

The shape shows up well outside puzzles too. The smallest timeout that still covers the slowest one percent of requests. The fewest workers that still finish a batch before a deadline, provided adding a worker never makes it later. Whenever you can name the yes-or-no question and argue that a yes stays a yes, you can search the answer instead of walking to it.

I know I am back in the old habit when I only reach for binary search after seeing something sorted, or when I have started writing a loop over candidate sizes without having named the predicate, or when I try to binary search an unsorted array, decide the algorithm does not apply, and never ask what question I am actually asking of each element.

The parts I still get wrong are mostly at the edges. Off-by-one errors around the first yes: inclusive bounds, a midpoint that stops moving, a probe that answers no for the exact value I want. I still draw the row of noes and yeses on paper when I am unsure. I still occasionally force the frame onto something that is not monotonic, and "does this cache size improve the slow tail" is the example that catches me, because the answer can get worse before it gets better and the search will happily lie about it. And I sometimes treat the frame as permission to skip thinking about the walk, when the walk is the predicate and the search is only how many times it runs.

The question that survives all of it is the same one. What am I allowed to ask about a candidate answer, and does a yes stay a yes.

If this is useful, wrong, or incomplete, write to me.
