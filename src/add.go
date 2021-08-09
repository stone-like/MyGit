package src

import (
	data "mygit/src/database"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	"os"
	"path/filepath"
)

//Addの時にindexとworkspaceを比較してdeletedなファイルの場合は、indexからも削除
func StartAdd(rootPath, uName, uEmail, message string, selectedPath []string) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	// w := &WorkSpace{
	// 	Path: rootPath,
	// }

	// d := &data.Database{
	// 	Path: dbPath,
	// }

	// i := data.GenerateIndex(filepath.Join(gitPath, "index"))

	_, indexNonExist := os.Stat(repo.i.Path)

	l := lock.NewFileLock(repo.i.Path)
	l.Lock()
	defer l.Unlock()

	if indexNonExist == nil {
		//.git/indexがある場合のみLoad、newFileLockで存在しないならindexを作ってしまうのでStatの後にしなければならない
		err := repo.i.Load()
		if err != nil {
			return err
		}
	}

	for _, path := range selectedPath {
		//selectedPathに"."を指定した場合,filepath.Join("aaa/bbb",".")="aaa/bbb"となる
		absPath := filepath.Join(rootPath, path)
		pathList, err := repo.w.ListFiles(absPath)

		if err != nil {
			return err
		}

		if len(pathList) == 0 {
			//file
			err = AddIndex(path, repo)
			if err != nil {
				return err
			}
		} else {
			//dir
			for _, innerPath := range pathList {
				relPathFromRoot := filepath.Join(path, innerPath)
				err = AddIndex(relPathFromRoot, repo)
				if err != nil {
					return err
				}
			}
		}

	}

	//workspaceから削除されたファイルをindexからも削除
	s := GenerateStatus()
	//指定したcommitObjIdでstatusをみる
	err := s.IntitializeStatus(repo)
	if err != nil {
		return err
	}

	for path, status := range s.WorkSpaceChanges {
		//workSpaceとindexの変化でworkSpaceになくてIndexにあるものはAddの時IndexからもRemove
		if status == WORKSPACE_DELETE {
			repo.i.Remove(path)
		}
	}

	repo.i.Write(repo.i.Path)

	return nil
}

func AddIndex(path string, repo *Repository) error {
	c, err := repo.w.ReadFile(path)

	if err != nil {
		return err
	}

	b := &con.Blob{
		Content: c,
	}

	repo.d.Store(b)

	stat, err := repo.w.StatFile(path)

	if err != nil {
		return err
	}

	err = repo.i.Add(path, b.ObjId, stat, data.CreateIndex)

	if err != nil {
		return err
	}

	return nil
}
