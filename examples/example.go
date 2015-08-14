package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/danverbraganza/keysort"
)

type HardToSort struct {
	HiddenValue int
}

var count int

func (h HardToSort) TrueValue() int {
	count++
	time.Sleep(200 * time.Millisecond)
	return h.HiddenValue
}

func ExampleHardToSortSlice() []HardToSort {
	return []HardToSort{
		{13},
		{11},
		{9},
		{12},
		{8},
		{10},
	}
}

type ByHiddenValue []HardToSort

func (hs ByHiddenValue) Swap(i, j int) {
	hs[i], hs[j] = hs[j], hs[i]
}

func (hs ByHiddenValue) Len() int {
	return len(hs)
}

func (hs ByHiddenValue) Less(i, j int) bool {
	return hs[i].TrueValue() < hs[j].TrueValue()
}

func SortExampleNaively() {
	count = 0
	hs := ExampleHardToSortSlice()
	sort.Sort(ByHiddenValue(hs))
	fmt.Println(hs)
	fmt.Println(count)
}

func (hs ByHiddenValue) LessVal(i, j interface{}) bool {
	return i.(int) < j.(int)
}

func (hs ByHiddenValue) Key(i int) (interface{}, error) {
	return hs[i].TrueValue(), nil
}

func BenchmarkSortFunc(factory func() sort.Interface) {
	count = 0
	start := time.Now()
	s := factory()
	sort.Sort(s)
	end := time.Now()
	fmt.Println("sorted:", s)
	fmt.Println("Calls to Keyfunc:", count)
	fmt.Println("Elapsed time:", end.Sub(start))
}

func main() {
	BenchmarkSortFunc(func() sort.Interface { return ByHiddenValue(ExampleHardToSortSlice()) })
	BenchmarkSortFunc(func() sort.Interface { return keysort.By(ByHiddenValue(ExampleHardToSortSlice())) })
	BenchmarkSortFunc(func() sort.Interface { return keysort.PrimedBy(ByHiddenValue(ExampleHardToSortSlice()), 0) })
}
