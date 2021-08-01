package database

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func Test_Contain(t *testing.T) {
	ks := KeysSlice{
		EntryKey{Path: "xxxx", Stage: 0},
		EntryKey{Path: "test2", Stage: 1},
	}

	for _, d := range []struct {
		title    string
		expected bool
		target   EntryKey
	}{
		{
			"contained",
			true,
			EntryKey{Path: "xxxx", Stage: 0},
		},
		{
			"Notcontained",
			false,
			EntryKey{Path: "xxxx", Stage: 3},
		},
		{
			"contained2",
			true,
			EntryKey{Path: "test2", Stage: 1},
		},
	} {

		t.Run(d.title, func(t *testing.T) {
			assert.Equal(t, d.expected, ks.Contains(d.target))
		})
	}

}

func Test_Delete(t *testing.T) {
	ks := KeysSlice{
		EntryKey{Path: "xxxx", Stage: 0},
		EntryKey{Path: "test2", Stage: 1},
	}

	ret := ks.Delete(EntryKey{Path: "test2", Stage: 1})

	if diff := cmp.Diff(KeysSlice{{Path: "xxxx", Stage: 0}}, ret); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func Test_Sort(t *testing.T) {
	em := EntriesMap{EntryKey{Path: "test2", Stage: 3}: nil,
		EntryKey{Path: "test2", Stage: 1}: nil,
		EntryKey{Path: "aaa", Stage: 1}:   nil}

	expected := []EntryKey{
		{Path: "aaa", Stage: 1},
		{Path: "test2", Stage: 1},
		{Path: "test2", Stage: 3}}

	sortedKeys := em.GetSortedkey()

	if diff := cmp.Diff(expected, sortedKeys); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}
