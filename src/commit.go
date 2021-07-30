package src

import (
	"fmt"
	con "mygit/src/database/content"
	"path/filepath"
)

func StartCommit(rootPath, uName, uEmail, message string) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	// indexPath := filepath.Join(gitPath, "index")

	repo := GenerateRepository(rootPath, gitPath, dbPath)

	// w := &WorkSpace{
	// 	Path: rootPath,
	// }

	// d := &data.Database{
	// 	Path: dbPath,
	// }

	// i := data.GenerateIndex(indexPath)

	// r := &data.Refs{
	// 	Path: gitPath,
	// }

	err := repo.i.Load()
	if err != nil {
		return err
	}

	// tm := make(map[string]con.Object)
	// t := &con.Tree{
	// 	Entries: tm,
	// }

	// es, err := i.GetEntries()

	// if err != nil {
	// 	return err
	// }

	// t.Build(es)
	// // t.GenerateObjId()

	// t.Traverse(func(t *con.Tree) {
	// 	d.Store(t)
	// })

	// fmt.Printf("tree: %s\n", t.GetObjId())

	// author := con.GenerateAuthor(uName, uEmail)

	// parent, err := r.ReadHead()
	// if err != nil {
	// 	return err
	// }

	// commit := &con.Commit{
	// 	ObjId:   t.GetObjId(),
	// 	Parents: []string{parent},
	// 	Tree:    t,
	// 	Author:  author,
	// 	Message: message,
	// }

	parent, err := repo.r.ReadHead()
	if err != nil {
		return err
	}

	err = WriteCommit([]string{parent}, uName, uEmail, message, repo)
	if err != nil {
		return err
	}

	// d.Store(commit)

	// r.UpdateHead(commit.ObjId)

	// var rootMessage string
	// if len(parent) == 0 {
	// 	rootMessage = "(root-commit) "
	// } else {
	// 	rootMessage = ""
	// }

	// fmt.Printf("[%s %s] %s", rootMessage, commit.GetObjId(), message)

	return nil
}

func CreateCommit(parents []string, name, email, message string, repo *Repository) (*con.Commit, error) {

	t, err := CreateTree(repo)
	if err != nil {
		return nil, err
	}

	author := con.GenerateAuthor(name, email)

	commit := &con.Commit{
		ObjId:   t.GetObjId(),
		Parents: parents,
		Tree:    t,
		Author:  author,
		Message: message,
	}

	return commit, nil
}

func WriteCommit(parents []string, name, email, message string, repo *Repository) error {

	c, err := CreateCommit(parents, name, email, message, repo)
	if err != nil {
		return err
	}

	WriteTree(c.Tree, repo)

	repo.d.Store(c)

	err = repo.r.UpdateHead(c.ObjId)
	if err != nil {
		return err
	}

	var rootMessage string
	if len(parents) == 0 {
		rootMessage = "(root-commit) "
	} else {
		rootMessage = ""
	}

	fmt.Printf("[%s %s] %s", rootMessage, c.GetObjId(), message)

	return nil

}

func CreateTree(repo *Repository) (*con.Tree, error) {
	t := con.GenerateTree()

	es, err := repo.i.GetEntries()

	if err != nil {
		return nil, err
	}

	t.Build(es)

	return t, nil
}

func WriteTree(t *con.Tree, repo *Repository) {
	t.Traverse(func(t *con.Tree) {
		repo.d.Store(t)
	})
	fmt.Printf("tree: %s\n", t.GetObjId())
}
