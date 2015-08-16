// Package keysort provides a way perform a Schwartzian transform of data before
// sorting in Go.
package keysort

import (
	"runtime"
	"sync"
)

// The keysort Interface must be implemented by any container type that you want
// to sort by a key using the Schwartzian transform.
type Interface interface {
	LessVal(i, j interface{}) bool
	Key(i int) (interface{}, error)
	Swap(i, j int)
	Len() int
}

// A KeySortable wraps an Interface, and implements sort.Interface.
// This is meant to be created by calling Keysort(Interface)
type keySortable struct {
	// wrapped is the container that must be sorted.
	wrapped Interface
	// swaps is a slice of ints to keep track of swaps that have been
	// performed.
	swaps []int
	// memo is a temporary map to memoize the Key() function.
	// memo maps the _original_ index of the element to the value of its Key() function.
	memo map[int]interface{}
	// lock coordinates access to memo.
	lock sync.Mutex
}

// Given an instance of a keysort.Interface, create a keySortable struct that
// implements sort.Interface.
func Keysort(wrapped Interface) (ks keySortable) {
	wrappedLen := wrapped.Len()
	swaps := make([]int, wrappedLen)
	for i := 0; i < wrapped.Len(); i++ {
		swaps[i] = i
	}

	ks.wrapped = wrapped
	ks.memo = map[int]interface{}{}
	ks.swaps = swaps

	return
}

// Given an instance of a keysort.Interface, create a keySortable struct that
// implements sort.Interface, and Prime() it.
// parallelism is passed to Prime()
func PrimedKeysort(wrapped Interface, parallelism int) (ks keySortable) {
	ks = Keysort(wrapped)
	ks.prime(parallelism)
	return
}

// Less is designed to implement sort.Interface.
// Delegates the call to wrapped.ValLess() after retrieving the memoized values for the
// keys i, j.
func (ks keySortable) Less(i, j int) bool {
	IValue, _ := ks.Key(i)
	JValue, _ := ks.Key(j)
	return ks.wrapped.LessVal(IValue, JValue)
}

// Key calculates the value of calling wrapped.Key() on the element that is
// currently at index i.
func (ks keySortable) Key(i int) (interface{}, error) {
	// Look up the original index of what is currently at i
	originalIndex := ks.swaps[i]
	ks.lock.Lock()
	defer ks.lock.Unlock()
	var err error = nil

	if _, ok := ks.memo[originalIndex]; !ok {
		// Release lock while calculating value of Key().
		ks.lock.Unlock()

		var value interface{}
		value, err = ks.wrapped.Key(originalIndex)

		ks.lock.Lock()
		ks.memo[originalIndex] = value
	}
	return ks.memo[ks.swaps[i]], err
}

// Len is designed to implement sort.Interface.
// Delegates the call to to wrapped.Len()
func (ks keySortable) Len() int {
	return ks.wrapped.Len()
}

// Swap is designed to implement sort.Interface.
// Delegates the call to wrapped.Swap, while keeping track of the swaps.
func (ks keySortable) Swap(i, j int) {
	ks.swaps[i], ks.swaps[j] = ks.swaps[j], ks.swaps[i]
	ks.wrapped.Swap(i, j)
}

// Prime precomputes each wrapped.Key() in goroutines.
// parallelism is how many goroutines to run at a time. If parallelism is less than one, an runtime.GOMAXPROCS goroutines are used.
// INCOMPLETE: Does not aggregate any errors that Key might return.
func (ks keySortable) prime(parallelism int) error {
	values := make(chan int)
	wg := &sync.WaitGroup{}

	if parallelism < 1 {
		parallelism = runtime.GOMAXPROCS(-1)
	}

	wg.Add(parallelism)
	for i := 0; i < parallelism; i++ {
		go func() {
			for val := range values {
				ks.Key(val)
			}
			wg.Done()
		}()
	}

	for i := 0; i < ks.Len(); i++ {
		values <- i
	}
	close(values)

	wg.Wait()
	return nil
}
