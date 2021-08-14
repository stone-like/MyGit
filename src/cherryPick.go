package src

import (
	"errors"
	"fmt"
	"io"
	con "mygit/src/database/content"
	ers "mygit/src/errors"
	"path/filepath"
	"strings"
)

// type CherryPickOption struct {
// 	hasContinue bool
// 	hasAbort    bool
// 	hasQuit     bool
// }

// type CherryPick struct {
// 	args   []string
// 	repo   *Repository
// 	seq    *Sequencer
// 	pc     *PendingCommit
// 	option *CherryPickOption
// 	w      io.Writer
// }

// func GenerateCherryPick(args []string, seq *Sequencer, pc *PendingCommit, repo *Repository, option *CherryPickOption, w io.Writer) *CherryPick {
// 	return &CherryPick{
// 		args:   args,
// 		repo:   repo,
// 		seq:    seq,
// 		pc:     pc,
// 		option: option,
// 		w:      w,
// 	}
// }

type CherryPick struct {
}

func (ch *CherryPick) GetPendingType() PendingType {
	return PEDING_CHERRY_PICK_TYPE
}

var ErrorCommithasNotParentOnCherryPick = errors.New("ErrorCommithasNotParentOnCherryPick")

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

func RunPick(c *con.CommitFromMem, sd *SequenceData) error {

	// if cp.option.hasContinue {
	// 	return HandleCherryPickContinue(cp)
	// }

	m, err := GenerateCherryPickMerge(c, sd.repo)
	if err != nil {
		return err
	}

	if sd.pc.InProgress() {
		return HandleInProgress(m)
	}

	err = m.ResolveMerge(sd.w)
	if err != nil {
		return err
	}

	if sd.repo.i.IsConflicted() {
		return HandleConflict(c.Message, PEDING_CHERRY_PICK_TYPE, c, m, sd)
	}

	pickedCommit, err := CreateCommit(
		[]string{m.leftObjId},
		c.Author.Name,
		c.Author.Email,
		c.Message,
		sd.repo,
	)
	if err != nil {
		return err
	}

	return FinishRunCommit(pickedCommit, sd.repo)
}

//ここまでCheのみ

func CommitReverse(ss []*con.CommitFromMem) []*con.CommitFromMem {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}

	return ss
}

func (ch *CherryPick) StoreCommitToSeq(sd *SequenceData) error {
	revlist, err := GenerateRevListWithWalk(false, sd.repo, sd.args)
	if err != nil {
		return err
	}

	//時間が遅い順に返ってくる(F,E,D)のでreverse
	commitList, err := revlist.GetAllCommits()
	if err != nil {
		return err
	}

	for _, c := range CommitReverse(commitList) {
		sd.seq.Push(PICK, c)
	}

	return nil
}

func (ch *CherryPick) ContinueWriteCommit(sd *SequenceData) error {

	//まずadd .でstage1~3を除去することがmerge再開の前提
	if sd.repo.i.IsConflicted() {
		return HandleConflictedIndex()
	}

	headObjId, err := sd.repo.r.ReadHead()
	if err != nil {
		return err
	}

	pickObjId, err := sd.pc.GetMergeObjId(PEDING_CHERRY_PICK_TYPE)
	if err != nil {
		return err
	}

	o, err := sd.repo.d.ReadObject(pickObjId)
	if err != nil {
		return err
	}

	c, ok := o.(*con.CommitFromMem)
	if !ok {
		return ErrorObjeToEntryConvError
	}

	parents := []string{headObjId}

	pickedCommit, err := CreateCommit(parents, c.Author.Name, c.Author.Email, CHERRY_PICK_MESSAGE, sd.repo)
	if err != nil {
		return err
	}
	err = WriteCommit(pickedCommit, sd.repo)
	if err != nil {
		return err
	}

	err = sd.pc.Clear(PEDING_CHERRY_PICK_TYPE)
	if err != nil {
		return err
	}

	return nil

}

func StartCherryPick(rootPath string, args []string, option *SequenceOption, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	seq := GenerateSequencer(repo)
	pc := GeneratePendingCommit(repo.r.Path)

	sd := GenerateSequenceData(args, seq, pc, repo, option, w)

	return ers.HandleWillWriteError(RunSequencing(sd, &CherryPick{}), w)

}
