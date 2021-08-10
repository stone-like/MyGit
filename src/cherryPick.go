package src

import (
	"errors"
	"fmt"
	"io"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	ers "mygit/src/errors"
	"path/filepath"
	"strings"
)

type CherryPickOption struct {
	hasContinue bool
}

var ErrorCommithasNotParentOnCherryPick = errors.New("ErrorCommithasNotParentOnCherryPick")

var CherryPickConflictMessage = `hint: 
 after resolving the conflicts, mark the corrected paths
 with 'mygit add <paths>' or 'mygit rm <paths>'
 and commit the result with 'mygit commit'
`

//通常のマージとは違いrightが指定したcommit,baseがcommitの親となる
func GenerateCherryPickMerge(c *con.CommitFromMem, repo *Repository) (*Merge, error) {
	//baseがとれなくなるのでエラー
	if len(c.Parents) == 0 {
		return nil, ErrorCommithasNotParentOnCherryPick
	}

	shortObjId := ShortOid(c.ObjId, repo.d)
	leftName := "HEAD"
	leftObjId, err := repo.r.ReadHead()
	if err != nil {
		return nil, err
	}
	rightName := fmt.Sprintf("%s... %s", shortObjId, strings.TrimSpace(c.GetFirstLineMessage()))
	rightObjId := c.ObjId

	return &Merge{
		repo:       repo,
		leftName:   leftName,
		leftObjId:  leftObjId,
		rightName:  rightName,
		rightObjId: rightObjId,
		baseObjId:  c.Parents[0],
	}, nil
}

func FinishPick(c *con.Commit, repo *Repository) error {
	err := WriteCommit(c, repo)
	if err != nil {
		return err
	}
	PrintCommit(c)
	return nil
}

func CreateConflictMessage(commitMessage string, repo *Repository) string {
	var str string
	str += fmt.Sprintf("%s\n", commitMessage)
	str += fmt.Sprintf("\n")
	str += fmt.Sprintf("Conflicts:\n")
	for path, _ := range repo.i.ConflictPaths() {
		str += fmt.Sprintf("\t%s\n", path)
	}

	return str
}

func CreateConflictMessageError(rightName, message string) error {
	return &ers.MergeFailOnConflictError{
		Message: fmt.Sprintf("error: could not apply %s\n%s", rightName, message),
	}
}

func HandleConflict(c *con.CommitFromMem, m *Merge, pc *PendingCommit, repo *Repository) error {

	message := CreateConflictMessage(c.Message, repo)
	err := pc.Start(m.rightObjId, message, PEDING_CHERRY_PICK_TYPE)
	if err != nil {
		return err
	}

	return CreateConflictMessageError(m.rightName, CherryPickConflictMessage)
}

func HandleCherryPickContinue(pc *PendingCommit, repo *Repository) error {
	l := lock.NewFileLock(repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := repo.i.Load()
	if err != nil {
		return err
	}

	return WriteCherryPickCommit(pc, repo)

}

func HandleInProgressCherryPick(m *Merge) error {
	return CreateConflictMessageError(m.rightName, CherryPickConflictMessage)
}

func RunPick(c *con.CommitFromMem, option *CherryPickOption, repo *Repository, w io.Writer) error {
	pc := GeneratePendingCommit(repo.r.Path)

	if option.hasContinue {
		return HandleCherryPickContinue(pc, repo)
	}

	m, err := GenerateCherryPickMerge(c, repo)
	if err != nil {
		return err
	}

	if pc.InProgress() {
		return HandleInProgressCherryPick(m)
	}

	err = m.ResolveMerge(w)
	if err != nil {
		return err
	}

	if repo.i.IsConflicted() {
		return HandleConflict(c, m, pc, repo)
	}

	pickedCommit, err := CreateCommit(
		[]string{m.leftObjId},
		c.Author.Name,
		c.Author.Email,
		c.Message,
		repo,
	)
	if err != nil {
		return err
	}

	return FinishPick(pickedCommit, repo)
}

func StartCherryPick(rootPath string, args []string, option *CherryPickOption, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	rev, err := ParseRev(args[0])
	if err != nil {
		return err
	}
	objId, err := ResolveRev(rev, repo)
	if err != nil {
		return err
	}
	o, err := repo.d.ReadObject(objId)
	if err != nil {
		return err
	}

	c, ok := o.(*con.CommitFromMem)
	if !ok {
		return ErrorObjeToEntryConvError
	}

	return ers.HandleWillWriteError(RunPick(c, option, repo, w), w)

}
