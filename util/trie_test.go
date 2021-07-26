package util

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTrie(t *testing.T) {
	tr := GenerateTrieFromPaths([]string{"aaa/bbb/ccc"})
	expected := &Trie{
		Children: map[string]*Trie{
			"aaa": {
				Children: map[string]*Trie{
					"bbb": {
						Children: map[string]*Trie{
							"ccc": {
								Matched:  true,
								Children: make(map[string]*Trie),
							},
						},
					},
				},
			},
		},
	}
	if diff := cmp.Diff(tr, expected); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}
}
