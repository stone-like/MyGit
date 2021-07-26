package src

import (
	con "mygit/src/database/content"
	"mygit/util"
	"path/filepath"
)

type PathFilter struct {
	routes *util.Trie
	path   string
}

func GeneratePathFilterWithTrieAndPath(routes *util.Trie, path string) *PathFilter {
	return &PathFilter{
		routes: routes,
		path:   path,
	}
}

func GeneratePathFilterWithTrie(routes *util.Trie) *PathFilter {
	return &PathFilter{
		routes: routes,
		path:   "",
	}
}

//普通にPathFilterを作ると一段目のTrieのmatch=trueとなっているのですべてのpathがマッチする
//すなわちfilterされない
func GeneratePathFilter() *PathFilter {
	return &PathFilter{
		routes: util.GenerateTrieMacthedTrue(),
		path:   "",
	}
}

//PathFilterの役割はtreeDiffを取る時に、あらかじめlogで渡したpathのみでfilterすること
//つまりdiffがa.txt,b.txt,c.txtとあってもpathでa.txtしかとらなければ、a.txtのDiffしか表示されなくする
func (p *PathFilter) EachEntry(entries map[string]con.Object, fn func(k string, v con.Object) error) error {
	for k, v := range entries {
		if p.routes.Matched || p.routes.ChildrenHasKey(k) {
			err := fn(k, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PathFilter) Join(name string) *PathFilter {
	var nextRoutes *util.Trie
	if p.routes.Matched {
		nextRoutes = p.routes
	} else {
		nextRoutes = p.routes.GetOrCreateChildren(name)
	}
	return GeneratePathFilterWithTrieAndPath(nextRoutes, filepath.Join(p.path, name))
}
