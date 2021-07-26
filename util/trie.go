package util

type Trie struct {
	Matched  bool
	Children map[string]*Trie
}

func GenerateTrieFromPaths(paths []string) *Trie {
	root := GenerateTrie()
	if len(paths) == 0 {
		root.Matched = true
		return root
	}

	for _, p := range paths {
		trie := root

		for _, disectedPath := range DisectPath(p) {

			_, ok := trie.Children[disectedPath]
			if !ok {
				trie.Children[disectedPath] = GenerateTrie()
			}

			trie = trie.Children[disectedPath]
		}

		trie.Matched = true //disectした最後だけtrue

	}
	return root
}

func GenerateTrie() *Trie {
	return &Trie{
		Matched:  false,
		Children: make(map[string]*Trie),
	}
}

func GenerateTrieMacthedTrue() *Trie {
	return &Trie{
		Matched:  true,
		Children: make(map[string]*Trie),
	}
}

func (t *Trie) ChildrenHasKey(key string) bool {
	_, ok := t.Children[key]
	return ok
}

func (t *Trie) GetOrCreateChildren(key string) *Trie {
	_, ok := t.Children[key]
	if !ok {
		t.Children[key] = GenerateTrie()
	}

	return t.Children[key]
}
