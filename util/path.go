package util

import (
	"io/ioutil"
	"os"
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

func DeleteParentDir(path, rootpath string) error {
	for _, p := range ParentDirs(path, true) {
		absPath := filepath.Join(rootpath, p)
		if absPath == rootpath {
			break
		}

		files, err := ioutil.ReadDir(absPath)
		if err != nil {
			return err
		}

		if len(files) != 0 {
			//Dirが空でなければ削除しない
			break
		}

		err = os.Remove(absPath)

		if err != nil {
			return err
		}

	}

	return nil
}

//WorkSpaceの奴ではなくこちらを主に使うようにする、
//WorkSpaceのListFileでもこっちを呼ぶ(refsとかのDatabase階層でも使いたいので)
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

func reverse(ss []string) []string {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}

	return ss
}

func DisectPath(path string) []string {

	var paths []string
	for {
		temp := filepath.Base(path)
		if temp == "/" || temp == "." {
			break
		}

		path = filepath.Dir(path)
		paths = append(paths, temp)
	}
	return reverse(paths)
}
