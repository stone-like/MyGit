package src

import (
	"fmt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"path/filepath"
)

func StartCommit(rootPath, uName, uEmail, message string) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	indexPath := filepath.Join(gitPath, "index")

	// w := &WorkSpace{
	// 	Path: rootPath,
	// }

	d := &data.Database{
		Path: dbPath,
	}

	i := data.GenerateIndex(indexPath)

	r := &data.Refs{
		Path: gitPath,
	}

	// pathList, err := w.ListFiles(rootPath)

	// if err != nil {
	// 	return err
	// }

	// var entryList []*con.Entry

	// for _, path := range pathList {
	// 	c, err := w.ReadFile(path)

	// 	if err != nil {
	// 		return err
	// 	}

	// 	b := &con.Blob{
	// 		Content: c,
	// 	}

	// 	// bcontent := d.CreateContent(b)
	// 	// d.SetObjId(b, bcontent)
	// 	d.Store(b)

	// 	stat, err := w.StatFile(path)

	// 	if err != nil {
	// 		return err
	// 	}
	// 	entryList = append(entryList, &con.Entry{
	// 		Mode:  int(stat.Mode()),
	// 		Path:  path,
	// 		ObjId: b.ObjId,
	// 	})
	// }
	// m := make(map[string]con.Object)

	err := i.Load()
	if err != nil {
		return err
	}

	tm := make(map[string]con.Object)
	t := &con.Tree{
		Entries: tm,
	}

	es, err := i.GetEntries()

	if err != nil {
		return err
	}

	t.Build(es)
	// t.GenerateObjId()

	t.Traverse(func(t *con.Tree) {
		d.Store(t)
	})

	fmt.Printf("tree: %s\n", t.GetObjId())

	author := con.GenerateAuthor(uName, uEmail)

	parent, err := r.ReadHead()
	if err != nil {
		return err
	}

	commit := &con.Commit{
		ObjId:   t.GetObjId(),
		Parent:  parent,
		Tree:    t,
		Author:  author,
		Message: message,
	}

	d.Store(commit)

	r.UpdateHead(commit.ObjId)

	var rootMessage string
	if len(parent) == 0 {
		rootMessage = "(root-commit) "
	} else {
		rootMessage = ""
	}

	fmt.Printf("[%s %s] %s", rootMessage, commit.GetObjId(), message)

	return nil
}
