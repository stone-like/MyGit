package src

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	"mygit/util"
	"os"
	"path/filepath"
	"strings"
)

var ignoreList = []string{
	".",
	".git", ".vscode", "cmd", "src", "util",
	".mygit.yml", "go.mod", "go.sum", "main.go", "testData"}

type WorkSpace struct {
	Path string
}

func (w *WorkSpace) ReadFile(path string) (string, error) {

	absPath := filepath.Join(w.Path, path)
	if _, err := os.Stat(absPath); err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadFile(absPath)

	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (w *WorkSpace) ListDir(path string) (map[string]con.FileState, error) {
	fileAndStat, err := w.FilePathWalkDirStat(path, ignoreList)

	if err != nil {
		return nil, err
	}

	return fileAndStat, nil

}

func (w *WorkSpace) FilePathWalkDirStat(root string, ignoreList []string) (map[string]con.FileState, error) {
	fileAndStat := make(map[string]con.FileState)

	//DirStatでは root/someDir/xxx.txtのxxx.txtまではやらなくていい
	files, err := ioutil.ReadDir(root)

	if err != nil {
		return nil, err
	}

	for _, f := range files {
		match, er := pathMatch(ignoreList, f.Name())

		if er != nil {
			return nil, er
		}

		if !match {

			stat, er := os.Stat(filepath.Join(root, f.Name()))

			if er != nil {
				return nil, er
			}

			relPath, er := filepath.Rel(w.Path, filepath.Join(root, f.Name()))

			if er != nil {
				return nil, er
			}

			fileAndStat[relPath] = stat
		}
	}

	return fileAndStat, nil

}

func (w *WorkSpace) ListFiles(path string) ([]string, error) {
	files, err := w.FilePathWalkDir(path, ignoreList)

	if err != nil {
		return nil, err
	}

	return files, nil

}

func (w *WorkSpace) FilePathWalkDir(root string, ignoreList []string) ([]string, error) {
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

func (w *WorkSpace) WriteFile(path, content string) error {
	f, err := os.Create(filepath.Join(w.Path, path))

	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	if err != nil {
		return err
	}

	f.Write([]byte(content))

	return nil
}

//os.FileInfoを独自の構造体にwrapした方がよさそう
func (w *WorkSpace) StatFile(path string) (con.FileState, error) {
	absPath := filepath.Join(w.Path, path)
	return os.Stat(absPath)
}

func (w *WorkSpace) ApplyMigration(m *Migration) error {
	w.ApplyChangeList(m, MIGRATION_DELETE)
	//revereseするのはnestの深いdirからdeleteしていくため

	for _, k := range util.SortKeysReverse(util.SortedKeys(m.Rmdirs)) {
		w.RemoveDir(k)
	}

	for _, k := range util.SortedKeys(m.Mkdirs) {
		w.MakeDir(k)
	}

	w.ApplyChangeList(m, MIGRATION_UPDATE)
	w.ApplyChangeList(m, MIGRATION_CREATE)

	return nil
}

func (w *WorkSpace) RemoveDir(path string) error {
	err := os.RemoveAll(filepath.Join(w.Path, path))
	if err != nil {
		return err
	}

	return nil
}

func (w *WorkSpace) MakeDir(path string) error {
	absPath := filepath.Join(w.Path, path)
	stat, nonExists := os.Stat(absPath)

	if nonExists == nil {
		if !stat.IsDir() {
			//すでにファイルで存在しているとき
			err := os.RemoveAll(absPath)
			if err != nil {
				return err
			}

			os.MkdirAll(absPath, os.ModePerm)
		}
	}

	//存在しないとき
	os.MkdirAll(absPath, os.ModePerm)

	return nil
}

func (w *WorkSpace) ApplyChangeList(m *Migration, action string) error {
	for path, newItem := range m.Changes[action] {
		absPath := filepath.Join(w.Path, path)

		err := os.RemoveAll(absPath)
		if err != nil {
			return err
		}

		if action == MIGRATION_DELETE {
			continue
		}
		//treeDiffからとってきたやつは全部blobの想定なので
		content, err := m.BlobContent(newItem.ObjId)
		if err != nil {
			return err
		}

		//おそらくfilePathがxxx/dummy.txtでxxxがないのでだめだった<-いまはok

		//fileLock
		l := lock.NewFileLock(absPath)
		l.Lock()
		defer l.Unlock()
		//apply_change_listyはファイルのみでDirがない、Dirは別の関数で作る
		//書き込みのみ
		f, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE, os.ModePerm)
		defer func() {
			err := f.Close()
			fmt.Printf("closeError is: %s\n", err)
		}()

		if err != nil {
			return err
		}

		os.Chmod(absPath, fs.FileMode(newItem.Mode))

		f.Write([]byte(content))
	}

	return nil
}
