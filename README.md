# keysort
An adapter to enable the [Schwartzian Transform](https://en.wikipedia.org/wiki/Schwartzian_transform) when sorting in Go. **keysort** is currently an alpha release, see the warning below.

Example
-------

Given a type `HardToSort`, where calculating the comparison key `TrueValue()` is expensive:

    type HardToSort struct {
        // Simulate a hard-to-calculate value by hiding access to this value.
        hiddenValue int
    }
    
    // Getting the True Value takes some time.
    func (h HardToSort) TrueValue() int {
        time.Sleep(200 * time.Millisecond)
        return h.hiddenValue
    }

To perform a sort that minimizes calls to `TrueValue()` by **automatically** memoizing calls, simply implement  `keysort.Interface`, which is only one more function than `sort.Interface`

    type ByHiddenValue []HardToSort

    func (hs ByHiddenValue) Swap(i, j int) {
        hs[i], hs[j] = hs[j], hs[i]
    }

    func (hs ByHiddenValue) Len() int {
	      return len(hs)
    }

    func (hs ByHiddenValue) LessVal(i, j interface{}) bool {
        return i.(int) < j.(int)
    }

    func (hs ByHiddenValue) Key(i int) (interface{}, error) {
	       return hs[i].TrueValue(), nil
    }

Then you can perform your sort with:

    sort.Sort(keysort.Keysort(ByHiddenValue([]HardToSort{{13}, {11}, {9}, {12}})))
    
which is faster because it automatically memoizes calls to TrueValue.

You can also precompute and memoize all the Key functions concurrently on initialize, using

    sort.Sort(keysort.PrimedKeysort(ByHiddenValue([]HardToSort{{13}, {11}, {9}, {12}})))

If this concurrency can be exploited by the Go runtime (e.g. you have multiple processors, or calculating the Key functions can be run on multiple processors), you will see a noticeable speedup.


Motivation
----------
The [Schwartzian Transform](https://en.wikipedia.org/wiki/Schwartzian_transform) is a way of speeding up the process of sorting items that need to be compared by keys, when the cost of calculating the key for each item is expensive.

For example, consider sorting a list of `image.Image`s by their average luminescence. Calculating the luminescence of a given image is an expensive operation that requires a nested loop over `i` and `j` within `Image.Bounds()`, calculating the luminescence, taking the sum of all of these, and then averaging them all.

To sort this, Go's sort.Interface requires that you implement `Len()`, `Swap()` and `Less()`. A naive implementation of Less is shown below:

    func (b ByLuminescence) Less(i, j int) {
        Lum(b[i]) < Lum(b[j]) 
    }

Each time the sorting algorithm must compare two images, it needs to compute the luminescence twice. Clearly some caching can be used to improve performance. **keysort** provides a way to get this caching without duplicating a lot of code.


