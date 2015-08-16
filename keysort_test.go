package keysort

import (
	"math/rand"
	"sort"
	"testing"
)

const SPECIMEN_SIZE = 20

type ExampleToSort struct {
	// We must never sort by NotKey.
	NotKey, IntKey int
	StringKey      string
}

func GenSpecimen(num int) []ExampleToSort {
	result := make([]ExampleToSort, num)
	for i := 0; i < num; i++ {
		var example ExampleToSort
		switch i {
		// Ensure the first two elements are always out of order.
		case 0:
			example = ExampleToSort{NotKey: 0, IntKey: 1, StringKey: "bbb"}
		case 1:
			example = ExampleToSort{NotKey: 1, IntKey: 0, StringKey: "aaa"}
		default:
			// Come up with a random example.
			example = ExampleToSort{
				NotKey: rand.Intn(num),
				IntKey: rand.Intn(num),
				StringKey: string(byte(rand.Intn(num)))}
		}
		result[i] = example
	}
	return result
}

func TestKeysortByIntKey(t *testing.T) {
	specimen := ByIntKey{GenSpecimen(SPECIMEN_SIZE)}

	sort.Sort(Keysort(specimen))

	if !sort.IsSorted(specimen) {
		t.Errorf("Keysort failed for ByIntKey")
	}
}

func TestKeysortByIntKeyCounted(t *testing.T) {
	specimen := ByIntKeyCounted{GenSpecimen(SPECIMEN_SIZE), 0}
	sort.Sort(Keysort(specimen))

	if !sort.IsSorted(ByIntKey{specimen.SpecimenSliceSorter}) {
		t.Errorf("Keysort failed for ByIntKey.")
	}

	if specimen.count > SPECIMEN_SIZE {
		t.Errorf("Key() called too often.")
	}
}

func TestPrimedKeysortByIntKey(t *testing.T) {
	specimen := ByIntKey{GenSpecimen(SPECIMEN_SIZE)}

	sort.Sort(PrimedKeysort(specimen, -1))

	if !sort.IsSorted(specimen) {
		t.Errorf("PrimedKeysort failed for ByIntKey")
	}
}


func TestKeysortByStringKey(t *testing.T) {
	specimen := ByStringKey{GenSpecimen(SPECIMEN_SIZE)}

	sort.Sort(Keysort(specimen))

	if !sort.IsSorted(specimen) {
		t.Errorf("Keysort failed for ByStringKey")
	}
}


func TestPrimedKeysortByStringKey(t *testing.T) {
	specimen := ByStringKey{GenSpecimen(SPECIMEN_SIZE)}

	sort.Sort(PrimedKeysort(specimen, -1))

	if !sort.IsSorted(specimen) {
		t.Errorf("PrimedKeysort failed for ByStringKey")
	}
}

type SpecimenSliceSorter []ExampleToSort

func (s SpecimenSliceSorter) Len() int {
	return len(s)
}

func (s SpecimenSliceSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SpecimenSliceSorter) At(i int) ExampleToSort {
	return s[i]
}

type ByIntKey struct {SpecimenSliceSorter}

func (s ByIntKey) Less(i, j int) bool {
	return s.At(i).IntKey < s.At(j).IntKey
}

func (s ByIntKey) LessVal(i, j interface{}) bool {
	return i.(int) < j.(int)
}

func (s ByIntKey) Key(i int) (interface{}, error) {
	return s.At(i).IntKey, nil
}

type ByStringKey struct {SpecimenSliceSorter}

func (s ByStringKey) Less(i, j int) bool {
	return s.At(i).StringKey < s.At(j).StringKey
}

func (s ByStringKey) LessVal(i, j interface{}) bool {
	return i.(string) < j.(string)
}

func (s ByStringKey) Key(i int) (interface{}, error) {
	return s.At(i).StringKey, nil
}

type ByIntKeyCounted struct {
	SpecimenSliceSorter
	count int
}

func (s ByIntKeyCounted) LessVal(i, j interface{}) bool {
	return i.(int) < j.(int)
}

func (s ByIntKeyCounted) Key(i int) (interface{}, error) {
	s.count++
	return s.At(i).IntKey, nil
}
