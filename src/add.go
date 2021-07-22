package src

import (
	data "mygit/src/database"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	"os"
	"path/filepath"
)

func StartAdd(rootPath, uName, uEmail, message string, selectedPath []string) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")

	w := &WorkSpace{
		Path: rootPath,
	}

	d := &data.Database{
		Path: dbPath,
	}

	i := data.GenerateIndex(filepath.Join(gitPath, "index"))

	_, indexNonExist := os.Stat(i.Path)

	l := lock.NewFileLock(i.Path)
	l.Lock()
	defer l.Unlock()

	if indexNonExist == nil {
		//.git/indexがある場合のみLoad、newFileLockで存在しないならindexを作ってしまうのでStatの後にしなければならない
		err := i.Load()
		if err != nil {
			return err
		}
	}

	for _, path := range selectedPath {
		//selectedPathに"."を指定した場合,filepath.Join("aaa/bbb",".")="aaa/bbb"となる
		absPath := filepath.Join(rootPath, path)
		pathList, err := w.ListFiles(absPath)

		if err != nil {
			return err
		}

		if len(pathList) == 0 {
			//file
			err = AddIndex(path, i, w, d)
			if err != nil {
				return err
			}
		} else {
			//dir
			for _, innerPath := range pathList {
				relPathFromRoot := filepath.Join(path, innerPath)
				err = AddIndex(relPathFromRoot, i, w, d)
				if err != nil {
					return err
				}
			}
		}

	}

	i.Write(i.Path)

	return nil
}

func AddIndex(path string, i *data.Index, w *WorkSpace, d *data.Database) error {
	c, err := w.ReadFile(path)

	if err != nil {
		return err
	}

	b := &con.Blob{
		Content: c,
	}

	d.Store(b)

	stat, err := w.StatFile(path)

	if err != nil {
		return err
	}

	err = i.Add(path, b.ObjId, stat, data.CreateIndex)

	if err != nil {
		return err
	}

	return nil
}
