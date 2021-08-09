package src

import (
	"errors"
	"fmt"
	"io"
	con "mygit/src/database/content"
	"path/filepath"
	"syscall"
)

func StartCommit(rootPath, uName, uEmail, message string, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")

	repo := GenerateRepository(rootPath, gitPath, dbPath)

	err := repo.i.Load()
	if err != nil {
		return err
	}

	//conflictのあとはadd . -> merge --c or commitで解消する、commitの時はこっち
	pc := GeneratePendingCommit(gitPath)
	if pc.InProgress() {
		return ResumeMerge(uName, uEmail, pc, repo, w)

	}

	parent, err := repo.r.ReadHead()
	if err != nil {
		return err
	}

	err = WriteCommit([]string{parent}, uName, uEmail, message, repo)
	if err != nil {
		return err
	}

	return nil
}

var CONFLICT_MESSAGE = `hint: Fix them up in the work tree, and then use 'mygit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.`

var CONFLICT_INDEXMESSAGE = "Commiting is not possible because you have unmerged files"

func HandleConflictedIndex(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("error: %s/n", CONFLICT_INDEXMESSAGE)))
	w.Write([]byte(CONFLICT_MESSAGE))
	w.Write([]byte("\n"))
}

func ResumeMerge(name, email string, pc *PendingCommit, repo *Repository, w io.Writer) error {
	//まずadd .でstage1~3を除去することがmerge再開の前提
	if repo.i.IsConflicted() {
		HandleConflictedIndex(w)
		return nil
	}

	head, err := repo.r.ReadHead()
	if err != nil {
		return err
	}
	mergeObjId, err := pc.GetMergeObjId()
	if err != nil {
		if errors.Is(err, syscall.ENOENT) {
			w.Write([]byte(fmt.Sprintf("There is no merge in progress (%s missng).\n", mergeObjId)))
		}
		return err
	}
	mergeMessage, err := pc.GetMergeMessage()
	if err != nil {
		return err
	}

	err = WriteCommit([]string{head, mergeObjId}, name, email, mergeMessage, repo)
	if err != nil {
		return err
	}

	//MergeHead等を削除
	err = pc.Clear()
	if err != nil {
		return err
	}

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

	_, err = repo.r.UpdateHead(c.ObjId)
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
