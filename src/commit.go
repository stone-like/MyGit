package src

import (
	"fmt"
	"io"
	con "mygit/src/database/content"
	ers "mygit/src/errors"
	"path/filepath"
)

func RunCommit(uName, uEmail, message string, pc *PendingCommit, repo *Repository, w io.Writer) error {
	if pc.InProgress() {
		return ResumeMerge(uName, uEmail, PENDING_MERGE_TYPE, pc, repo)

	}

	parent, err := repo.r.ReadHead()
	if err != nil {
		return err
	}

	err = ProcessCommit([]string{parent}, uName, uEmail, message, repo)
	if err != nil {
		return err
	}

	return nil
}

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

	return ers.HandleWillWriteError(RunCommit(uName, uEmail, message, pc, repo, w), w)
}

var CONFLICT_MESSAGE = `hint: Fix them up in the work tree, and then use 'mygit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.`

var CONFLICT_INDEXMESSAGE = "Commiting is not possible because you have unmerged files"

var CHERRY_PICK_MESSAGE = `It looks like you may be committing a cherry-pick.
if this is not correct, please remove the file
	.git/CHERRY_PICK_HEAD
and try again.
`

func HandleConflictedIndex() error {
	var str string
	str += fmt.Sprintf("error: %s\n", CONFLICT_INDEXMESSAGE)
	str += fmt.Sprintf("%s\n", CONFLICT_MESSAGE)

	return &ers.MergeFailOnConflictError{
		Message: str,
	}
}

// func WriteCherryPickCommit(pc *PendingCommit, repo *Repository) error {

// 	//まずadd .でstage1~3を除去することがmerge再開の前提
// 	if repo.i.IsConflicted() {
// 		return HandleConflictedIndex()
// 	}

// 	headObjId, err := repo.r.ReadHead()
// 	if err != nil {
// 		return err
// 	}

// 	pickObjId, err := pc.GetMergeObjId(PEDING_CHERRY_PICK_TYPE)
// 	if err != nil {
// 		return err
// 	}

// 	o, err := repo.d.ReadObject(pickObjId)
// 	if err != nil {
// 		return err
// 	}

// 	c, ok := o.(*con.CommitFromMem)
// 	if !ok {
// 		return ErrorObjeToEntryConvError
// 	}

// 	parents := []string{headObjId}

// 	pickedCommit, err := CreateCommit(parents, c.Author.Name, c.Author.Email, CHERRY_PICK_MESSAGE, repo)
// 	if err != nil {
// 		return err
// 	}
// 	err = WriteCommit(pickedCommit, repo)
// 	if err != nil {
// 		return err
// 	}

// 	err = pc.Clear(PEDING_CHERRY_PICK_TYPE)
// 	if err != nil {
// 		return err
// 	}

// 	return nil

// }

func WriteMergeCommit(name, email string, pc *PendingCommit, repo *Repository) error {
	//まずadd .でstage1~3を除去することがmerge再開の前提
	if repo.i.IsConflicted() {
		return HandleConflictedIndex()
	}

	head, err := repo.r.ReadHead()
	if err != nil {
		return err
	}
	mergeObjId, err := pc.GetMergeObjId(PENDING_MERGE_TYPE)
	if err != nil {
		return err
	}
	mergeMessage, err := pc.GetMergeMessage()
	if err != nil {
		return err
	}

	err = ProcessCommit([]string{head, mergeObjId}, name, email, mergeMessage, repo)
	if err != nil {
		return err
	}

	//MergeHead等を削除
	err = pc.Clear(PENDING_MERGE_TYPE)
	if err != nil {
		return err
	}

	return nil
}

// func WriteRevertCommit(pc *PendingCommit, repo *Repository) error {

// 	//まずadd .でstage1~3を除去することがmerge再開の前提
// 	if repo.i.IsConflicted() {
// 		return HandleConflictedIndex()
// 	}

// 	headObjId, err := repo.r.ReadHead()
// 	if err != nil {
// 		return err
// 	}

// 	o, err := repo.d.ReadObject(headObjId)
// 	if err != nil {
// 		return err
// 	}

// 	c, ok := o.(*con.CommitFromMem)
// 	if !ok {
// 		return ErrorObjeToEntryConvError
// 	}

// 	mergeMessage, err := pc.GetMergeMessage()
// 	if err != nil {
// 		return err
// 	}

// 	parents := []string{headObjId}

// 	revertedCommit, err := CreateCommit(parents, c.Author.Name, c.Author.Email, mergeMessage, repo)
// 	if err != nil {
// 		return err
// 	}
// 	err = WriteCommit(revertedCommit, repo)
// 	if err != nil {
// 		return err
// 	}

// 	err = pc.Clear(PENDING_REVERT_TYPE)
// 	if err != nil {
// 		return err
// 	}

// 	return nil

// }

func ResumeMerge(name, email string, pendingType PendingType, pc *PendingCommit, repo *Repository) error {
	switch pendingType {
	case PENDING_MERGE_TYPE:
		return WriteMergeCommit(name, email, pc, repo)
	// case PEDING_CHERRY_PICK_TYPE:
	// 	return WriteCherryPickCommit(pc, repo)
	// case PENDING_REVERT_TYPE:
	// 	return WriteRevertCommit(pc, repo)
	default:
		return ErrorInvalidMergeType
	}
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

func PrintCommit(c *con.Commit) {
	var rootMessage string
	if len(c.Parents) == 0 {
		rootMessage = "(root-commit) "
	} else {
		rootMessage = ""
	}

	fmt.Printf("[%s %s] %s", rootMessage, c.GetObjId(), c.Message)

}

func WriteCommit(c *con.Commit, repo *Repository) error {

	WriteTree(c.Tree, repo)

	repo.d.Store(c)

	_, err := repo.r.UpdateHead(c.ObjId)

	return err

}

//ProccessCommitはCreateCommit,WriteCommit,PrintCommitをまとめたもの
func ProcessCommit(parents []string, name, email, message string, repo *Repository) error {
	c, err := CreateCommit(parents, name, email, message, repo)
	if err != nil {
		return err
	}
	err = WriteCommit(c, repo)
	if err != nil {
		return err
	}

	PrintCommit(c)
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
