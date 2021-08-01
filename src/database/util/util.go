package util

import (
	"errors"
	"mygit/src/database/content"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func RemovedSlice(s []string, e []string) []string {
	var result []string
	for _, v := range e {
		if !Contains(s, v) {
			//sの中でeに入っていないやつだけを取得
			result = append(result, v)
		}
	}

	return result
}

func Contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func ContainsInt(s []int, e int) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func Delete(s []string, i int) []string {
	s = append(s[:i], s[i+1:]...)
	n := make([]string, len(s))
	copy(n, s)
	return n
}

var ErrorMapNonExists = errors.New("map non exists")

var ErrorSliceNonExists = errors.New("slice non exists")

func DeleteFromMap(m map[string]content.Object, key string) map[string]content.Object {
	if _, ok := m[key]; !ok {
		return m
	}

	delete(m, key)
	return m
}

//reflectとか使って二つまとめたい
func DeleteFromParnet(m map[string][]string, key string) map[string][]string {
	if _, ok := m[key]; !ok {
		return m
	}

	delete(m, key)
	return m
}

func DeleteSpecificKey(s []string, key string) []string {
	for i, k := range s {
		if k == key {
			return Delete(s, i)
		}
	}
	return s
}

func SortStringSlice(s []string) {
	sort.Strings(s)

	sort.Slice(s, func(j, k int) bool {
		return len(s[j]) < len(s[k])
	})
}

func SortedMapKey(m map[string]int) []string {
	temp := make([]string, 0, len(m))
	for k, _ := range m {
		temp = append(temp, k)
	}

	sort.Strings(temp)

	return temp
}

func CheckRegExp(reg, branchName string) bool {
	return regexp.MustCompile(reg).MatchString(branchName)
}

func CheckRegExpSubString(reg, branchName string) [][]string {
	return regexp.MustCompile(reg).FindAllStringSubmatch(branchName, -1)
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

func ParentDirs(path string, ascend bool) []string {
	ret := createParentDirs(path)

	if ascend {
		// xxx/yyy
		// xxxの順番
		sort.Slice(ret, func(i, j int) bool {
			return len(ret[i]) > len(ret[j])
		})
	} else {
		// xxx
		// xxx/yyyの順番
		sort.Slice(ret, func(i, j int) bool {
			return len(ret[i]) < len(ret[j])
		})
	}

	return ret
}

func FilePathWalkDir(root string, ignoreList []string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		p, er := filepath.Rel(root, path)

		if er != nil {
			return er
		}
		if !info.IsDir() {
			//.git/xxx/yyyとあるときに
			match, er := pathMatch(ignoreList, p)

			if er != nil {
				return er
			}

			if !match {
				files = append(files, p)
			}

		}

		return nil
	})
	return files, err
}

func pathMatch(s []string, e string) (bool, error) {
	for _, v := range s {
		b := strings.HasPrefix(e, v)

		if b {
			return true, nil
		}
	}
	return false, nil
}
