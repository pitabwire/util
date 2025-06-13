package util_test

import (
	"testing"

	"github.com/pitabwire/util"
)

type sortBytes []byte

func (s sortBytes) Len() int           { return len(s) }
func (s sortBytes) Less(i, j int) bool { return s[i] < s[j] }
func (s sortBytes) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func TestUnique(t *testing.T) {
	testCases := []struct {
		Input string
		Want  string
	}{
		{"", ""},
		{"abc", "abc"},
		{"aaabbbccc", "abc"},
	}

	for _, test := range testCases {
		input := []byte(test.Input)
		want := test.Want
		got := string(input[:util.Unique(sortBytes(input))])
		if got != want {
			t.Fatal("Wanted ", want, " got ", got)
		}
	}
}

type sortByFirstByte []string

func (s sortByFirstByte) Len() int           { return len(s) }
func (s sortByFirstByte) Less(i, j int) bool { return s[i][0] < s[j][0] }
func (s sortByFirstByte) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func TestUniquePicksLastDuplicate(t *testing.T) {
	input := []string{
		"aardvark",
		"avacado",
		"cat",
		"cucumber",
	}
	want := []string{
		"avacado",
		"cucumber",
	}
	got := input[:util.Unique(sortByFirstByte(input))]

	if len(want) != len(got) {
		t.Errorf("Wanted %#v got %#v", want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("Wanted %#v got %#v", want, got)
		}
	}
}

func TestUniquePanicsIfNotSorted(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Unique did not panic on unsorted input")
		}
	}()
	unsorted := sortBytes{'b', 'a'}
	_ = util.Unique(unsorted)
}

func TestUniqueStrings(t *testing.T) {
	testCases := []struct {
		Input []string
		Want  []string
	}{
		{[]string{"b", "a", "a", "c"}, []string{"a", "b", "c"}},
	}
	for _, test := range testCases {
		got := util.UniqueStrings(test.Input)
		if len(got) != len(test.Want) {
			t.Errorf("Wanted %v got %v", test.Want, got)
		}
		for i := range got {
			if got[i] != test.Want[i] {
				t.Errorf("Wanted %v got %v", test.Want, got)
			}
		}
	}
}
