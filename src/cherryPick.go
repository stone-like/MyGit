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
	hasAbort    bool
	hasQuit     bool
}

type CherryPick struct {
	args   []string
	repo   *Repository
	seq    *Sequencer
	pc     *PendingCommit
	option *CherryPickOption
	w      io.Writer
}

func GenerateCherryPick(args []string, seq *Sequencer, pc *PendingCommit, repo *Repository, option *CherryPickOption, w io.Writer) *CherryPick {
	return &CherryPick{
		args:   args,
		repo:   repo,
		seq:    seq,
		pc:     pc,
		option: option,
		w:      w,
	}
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

func HandleConflict(c *con.CommitFromMem, m *Merge, cp *CherryPick) error {

	//seqをつかってToDoへ書き込み
	err := cp.seq.WriteToDo()
	if err != nil {
		return err
	}

	message := CreateConflictMessage(c.Message, cp.repo)
	err = cp.pc.Start(m.rightObjId, message, PEDING_CHERRY_PICK_TYPE)
	if err != nil {
		return err
	}

	return CreateConflictMessageError(m.rightName, CherryPickConflictMessage)
}

func ContinueCommitProcess(cp *CherryPick) error {
	l := lock.NewFileLock(cp.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := cp.repo.i.Load()
	if err != nil {
		return err
	}

	err = WriteCherryPickCommit(cp.pc, cp.repo)
	if err != nil {
		return err
	}

	return nil
}

func HandleCherryPickContinue(cp *CherryPick) error {

	err := ContinueCommitProcess(cp)
	if err != nil {
		return err
	}

	//toDoPathをLock
	sl := lock.NewFileLock(cp.seq.GetToDoPath())
	sl.Lock()
	defer sl.Unlock()

	err = cp.seq.Load()
	if err != nil {
		return err
	}

	//RunPick中にエラーが出ると、shiftをする前でエラーとしてreturnされるので、まず操作完了させるためにshiftから
	cp.seq.Shift()

	//seq処理を再開
	return ResumeSeq(cp)

}

func HandleInProgressCherryPick(m *Merge) error {
	return CreateConflictMessageError(m.rightName, CherryPickConflictMessage)
}

func RunPick(c *con.CommitFromMem, cp *CherryPick) error {

	// if cp.option.hasContinue {
	// 	return HandleCherryPickContinue(cp)
	// }

	m, err := GenerateCherryPickMerge(c, cp.repo)
	if err != nil {
		return err
	}

	if cp.pc.InProgress() {
		return HandleInProgressCherryPick(m)
	}

	err = m.ResolveMerge(cp.w)
	if err != nil {
		return err
	}

	if cp.repo.i.IsConflicted() {
		return HandleConflict(c, m, cp)
	}

	pickedCommit, err := CreateCommit(
		[]string{m.leftObjId},
		c.Author.Name,
		c.Author.Email,
		c.Message,
		cp.repo,
	)
	if err != nil {
		return err
	}

	return FinishPick(pickedCommit, cp.repo)
}

func ResumeSeq(cp *CherryPick) error {

	for {

		nextCommit := cp.seq.NextCommand()

		if nextCommit == nil {
			break
		}

		err := RunPick(nextCommit, cp)
		if err != nil {
			return err
		}

		//正常にPickできたらShiftでSeqから取り除く
		_, err = cp.seq.Shift()
		if err != nil {
			return err
		}

		//正常にpickできたら、abortSafetyを最新のコミットにupdate
		err = cp.seq.UpdateAbortSafetyLatest()
		if err != nil {
			return err
		}
	}

	//エラーなしでできたら.git/sequencerを消す
	cp.seq.Clear()

	return nil
}

func CommitReverse(ss []*con.CommitFromMem) []*con.CommitFromMem {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}

	return ss
}

func StoreCommitToSeq(cp *CherryPick) error {
	revlist, err := GenerateRevListWithWalk(false, cp.repo, cp.args)
	if err != nil {
		return err
	}

	//時間が遅い順に返ってくる(F,E,D)のでreverse
	commitList, err := revlist.GetAllCommits()
	if err != nil {
		return err
	}

	for _, c := range CommitReverse(commitList) {
		cp.seq.Push(c)
	}

	return nil
}

//abortはquitに加えてcherryPickをする前に戻す、なのでsequencer.abortでresetHardと、updateRefを使用
func HandleCherryPickAbort(cp *CherryPick) error {
	if cp.pc.InProgress() {
		err := cp.pc.Clear(PEDING_CHERRY_PICK_TYPE)
		if err != nil {
			return err
		}
	}

	l := lock.NewFileLock(cp.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := cp.repo.i.Load()
	if err != nil {
		return err
	}

	err = cp.seq.Abort()
	if err != nil {
		return err
	}

	//indexもアップデート
	err = cp.repo.i.Write(cp.repo.i.Path)
	if err != nil {
		return err
	}

	return nil
}

//quitでやることはcherryPickで作ったファイル群を消すことだけ、ただindex,workspace,HEADはcherryPickの影響が残ったまま
func HandleCherryPickQuit(cp *CherryPick) error {
	//quit前にConflict中だったら
	if cp.pc.InProgress() {
		err := cp.pc.Clear(PEDING_CHERRY_PICK_TYPE)
		if err != nil {
			return err
		}
	}

	return cp.seq.Clear()

}

func RunCherryPick(cp *CherryPick) error {

	if cp.option.hasQuit {
		return HandleCherryPickQuit(cp)
	}

	if cp.option.hasAbort {
		return HandleCherryPickAbort(cp)
	}

	if cp.option.hasContinue {
		return HandleCherryPickContinue(cp)
	}

	err := cp.seq.Start()
	if err != nil {
		return err
	}

	//toDoPathをLock,toDoはCherrypick失敗時、つまりresumrSeqの中のhandleConflictで書かれるのでここでToDoをロック(ResumeSeqの中の方がいいのかもしれないが)
	l := lock.NewFileLock(cp.seq.GetToDoPath())
	l.Lock()
	defer l.Unlock()

	//実際にcherryPickを始める前にStoreでTODOCOmmitを全部Seq.commandに追加している
	//Pick成功時にはcommandから取り除く
	//なのでエラー時にseq.Commandのやつを書き込めばtodo予定の奴だけ書き込まれる、成功済みは書き込まれない
	err = StoreCommitToSeq(cp)
	if err != nil {
		return err
	}

	return ResumeSeq(cp)

}

func StartCherryPick(rootPath string, args []string, option *CherryPickOption, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	seq := GenerateSequencer(repo)
	pc := GeneratePendingCommit(repo.r.Path)

	cp := GenerateCherryPick(args, seq, pc, repo, option, w)

	return ers.HandleWillWriteError(RunCherryPick(cp), w)

}
