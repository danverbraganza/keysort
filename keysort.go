// Package keysort provides a way perform a Schwartzian transform of data before
// sorting in Go.
package keysort

import (
	"fmt"
	"runtime"
	"strings"
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
	// errors is a map of original indices to error objects encountered by this object.
	errors map[int]error
	// lock coordinates access to memo and errors.
	sync.Mutex
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
	ks.errors = map[int]error{}
	ks.swaps = swaps
	return
}

// Given an instance of a keysort.Interface, create a keySortable struct that
// implements sort.Interface, and call memoize on it.
// parallelism is how many goroutines to run at once while memoizing.
func PrimedKeysort(wrapped Interface, parallelism int) (ks keySortable) {
	ks = Keysort(wrapped)
	ks.memoize(parallelism, ks.allIndexes)
	return
}

// Less is designed to implement sort.Interface. Delegates the call to
// wrapped.ValLess() after retrieving (and memoizing if necessary) values for
// the keys i, j.
func (ks keySortable) Less(i, j int) bool {
	IValue := ks.Key(i)
	JValue := ks.Key(j)

	// If there was an error, always return false from now on.
	if ks.Errors() != nil {
		return false
	}

	return ks.wrapped.LessVal(IValue, JValue)
}

// Key calculates the value of calling wrapped.Key() on the element that is
// currently at index i.
func (ks keySortable) Key(i int) interface{} {
	// Look up the original index of what is currently at i
	originalIndex := ks.swaps[i]
	ks.Lock()
	defer ks.Unlock()
	var err error = nil

	if _, ok := ks.memo[originalIndex]; !ok {
		// Release lock while calculating value of Key().
		ks.Unlock()

		var value interface{}
		value, err = ks.wrapped.Key(originalIndex)

		ks.Lock()
		// Whatever happened, write the value down.
		ks.memo[originalIndex] = value

		if err != nil {
			// If there was an error, note it.
			ks.errors[originalIndex] = err
		} else {
			// If there wasn't an error, ensure it's cleared.
			delete(ks.errors, originalIndex)
		}
	}
	return ks.memo[ks.swaps[i]]
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

// memoize precomputes each wrapped.Key() in goroutines.
// parallelism is how many goroutines to run at a time. If parallelism is less than one, an runtime.GOMAXPROCS goroutines are used.
func (ks keySortable) memoize(parallelism int, genIndexes func(chan<- int)) {

	// Channel on which we send indices to the key functions.
	iChan := make(chan int)
	wg := &sync.WaitGroup{}
	if parallelism < 1 {
		parallelism = runtime.GOMAXPROCS(-1)
	}

	wg.Add(parallelism)
	for i := 0; i < parallelism; i++ {
		go func() {
			for i := range iChan {
				ks.Key(i)
			}
			wg.Done()
		}()
	}

	genIndexes(iChan)
	wg.Wait()
}

// ClearErrors clears all the errors created on this keysort.
func (ks keySortable) ClearErrors() {
	ks.Lock()
	defer ks.Unlock()
	for k := range ks.errors {
		delete(ks.errors, k)
	}

}

// RetryFailed retries all the indexes that threw an error before
// parallelism is passed to memoize.
// All past errors are cleared on a retry.
func (ks keySortable) RetryFailed(parallelism int) {
	ks.ClearErrors()
	ks.memoize(parallelism, ks.erroredIndexes)
}

// allIndexes generates every possible index on the channel passed in as an
// argument, and then closes the channel.
func (ks keySortable) allIndexes(iChan chan<- int) {
	for i := 0; i < ks.Len(); i++ {
		iChan <- i
	}
	close(iChan)
}

// erroredIndexes generates only those indexes that have errored on the channel
// passed in as an argument, and then closes the channel.
func (ks keySortable) erroredIndexes(iChan chan<- int) {
	erroredIndices := []int{}
	ks.Lock()
	for i := range ks.errors {
		erroredIndices = append(erroredIndices, i)
	}
	ks.Unlock()

	for i := range erroredIndices {
		iChan <- i
	}
	close(iChan)
}

// Errors returns a non-nil error if one or more of the Key functions returned
// an error.
func (ks keySortable) Errors() error {
	if len(ks.errors) == 0 {
		return nil
	}
	return PrimingError{ks.errors}
}

// PrimingError is returned whenever a prime step fails. It may
// be queried to find the broken indexes, test what the errors were, and retry
// if necessary.
type PrimingError struct {
	Errors map[int]error
}

// Error returns a string representation of this error.
func (e PrimingError) Error() string {
	errorStrings := []string{}
	for i, err := range e.Errors {
		errorStrings = append(errorStrings, fmt.Sprintf("%d: %s", i, err.Error()))
	}

	return fmt.Sprintf(
		"Problem pre-computing Key functions.\n%s\n",
		strings.Join(errorStrings, "\t%s\n"))
}
