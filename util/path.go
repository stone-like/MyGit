package util

import (
	"path/filepath"
	"sort"
	"strings"
)

func Descend(path string) []string {

	var parentsAndMe []string

	//some/dummy/pathだと[some,dummy,path]
	splitted := strings.Split(path, string(filepath.Separator))

	var curPath string

	for _, s := range splitted {

		if s == "" {
			s = "/"
		}
		curFullPath := filepath.Join(curPath, s)
		parentsAndMe = append(parentsAndMe, curFullPath)
		curPath = curFullPath
	}
	return parentsAndMe
}

func createParentDirs(path string) []string {
	var parents []string
	dir := filepath.Dir(path)

	if dir != "." {
		ret := createParentDirs(dir)
		parents = append(parents, dir)
		parents = append(parents, ret...)
	}

	return parents

}

func ParentDirs(path string) []string {
	ret := createParentDirs(path)
	sort.Slice(ret, func(i, j int) bool {
		return len(ret[i]) < len(ret[j])
	})

	return ret

}
