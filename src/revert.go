package src

import (
	"fmt"
	"io"
	con "mygit/src/database/content"
	ers "mygit/src/errors"
	"path/filepath"
	"strings"
)

type Revert struct {
}

func (r *Revert) GetPendingType() PendingType {
	return PENDING_REVERT_TYPE
}

func (r *Revert) StoreCommitToSeq(sd *SequenceData) error {
	revlist, err := GenerateRevListWithWalk(false, sd.repo, sd.args)
	if err != nil {
		return err
	}

	//時間が遅い順に返ってくる(F,E,D)、revertではこのまま適用すればいい
	commitList, err := revlist.GetAllCommits()
	if err != nil {
		return err
	}

	for _, c := range commitList {
		sd.seq.Push(REVERT, c)
	}

	return nil
}

func (r *Revert) ContinueWriteCommit(sd *SequenceData) error {

	//まずadd .でstage1~3を除去することがmerge再開の前提
	if sd.repo.i.IsConflicted() {
		return HandleConflictedIndex()
	}

	headObjId, err := sd.repo.r.ReadHead()
	if err != nil {
		return err
	}

	o, err := sd.repo.d.ReadObject(headObjId)
	if err != nil {
		return err
	}

	c, ok := o.(*con.CommitFromMem)
	if !ok {
		return ErrorObjeToEntryConvError
	}

	mergeMessage, err := sd.pc.GetMergeMessage()
	if err != nil {
		return err
	}

	parents := []string{headObjId}

	revertedCommit, err := CreateCommit(parents, c.Author.Name, c.Author.Email, mergeMessage, sd.repo)
	if err != nil {
		return err
	}
	err = WriteCommit(revertedCommit, sd.repo)
	if err != nil {
		return err
	}

	err = sd.pc.Clear(PENDING_REVERT_TYPE)
	if err != nil {
		return err
	}

	return nil

}

// A -> B -> Cとして
//Cに-BとRevertする、この時BaseObjをB,RightをA,LeftがCとなる,こうすることで
//B -> Aの変化をCに適用できる
func GenerateRevertMerge(c *con.CommitFromMem, repo *Repository) (*Merge, error) {
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
	rightName := fmt.Sprintf("parent of %s... %s", shortObjId, strings.TrimSpace(c.GetFirstLineMessage()))
	rightObjId := c.Parents[0]

	return &Merge{
		repo:       repo,
		leftName:   leftName,
		leftObjId:  leftObjId,
		rightName:  rightName,
		rightObjId: rightObjId,
		baseObjId:  c.ObjId,
	}, nil
}

func GenerateRevertCommitMessage(c *con.CommitFromMem) string {
	return fmt.Sprintf("Revert %s\n\nThis reverts commit %s\n", c.GetFirstLineMessage(), c.ObjId)
}

func RunRevert(c *con.CommitFromMem, sd *SequenceData) error {

	m, err := GenerateRevertMerge(c, sd.repo)
	if err != nil {
		return err
	}
	message := GenerateRevertCommitMessage(c)

	if sd.pc.InProgress() {
		return HandleInProgress(m)
	}

	err = m.ResolveMerge(sd.w)
	if err != nil {
		return err
	}

	if sd.repo.i.IsConflicted() {
		return HandleConflict(message, PENDING_REVERT_TYPE, c, m, sd)
	}

	revertedCommit, err := CreateCommit(
		[]string{m.leftObjId},
		c.Author.Name,
		c.Author.Email,
		message,
		sd.repo,
	)
	if err != nil {
		return err
	}

	return FinishRunCommit(revertedCommit, sd.repo)
}

func StartRevert(rootPath string, args []string, option *SequenceOption, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	seq := GenerateSequencer(repo)
	pc := GeneratePendingCommit(repo.r.Path)

	sd := GenerateSequenceData(args, seq, pc, repo, option, w)

	return ers.HandleWillWriteError(RunSequencing(sd, &Revert{}), w)
}
