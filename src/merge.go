package src

import (
	"fmt"
	"io"
	"mygit/src/database/lock"
	ers "mygit/src/errors"
	"path/filepath"
)

//leftが基本的にHEADでrightがマージする対象
type Merge struct {
	repo       *Repository
	leftName   string
	rightName  string
	leftObjId  string
	rightObjId string
	baseObjId  string
}

func GenerateMerge(leftName, rightName string, repo *Repository) (*Merge, error) {
	rev, err := ParseRev(leftName)
	if err != nil {
		return nil, err
	}
	leftObjId, err := ResolveRev(rev, repo)
	if err != nil {
		return nil, err
	}

	rev, err = ParseRev(rightName)
	if err != nil {
		return nil, err
	}
	rightObjId, err := ResolveRev(rev, repo)
	if err != nil {
		return nil, err
	}

	baseObjId, err := GetBCA(leftObjId, rightObjId, repo.d)
	if err != nil {
		return nil, err
	}

	return &Merge{
		repo:       repo,
		leftName:   leftName,
		rightName:  rightName,
		leftObjId:  leftObjId,
		rightObjId: rightObjId,
		baseObjId:  baseObjId}, nil
}

func (m *Merge) ResolveMerge(w io.Writer) error {
	l := lock.NewFileLock(m.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := m.repo.i.Load()
	if err != nil {
		return err
	}

	resolveMerge := GenerateResolveMerge(m, w)
	err = resolveMerge.Resolve()
	if err != nil {
		return err
	}

	//indexもアップデート
	err = m.repo.i.Write(m.repo.i.Path)
	if err != nil {
		return err
	}

	return nil
}

func (m *Merge) CommitMerge(name, email, message string) error {
	err := WriteCommit([]string{m.leftObjId, m.rightObjId}, name, email, message, m.repo)
	if err != nil {
		return err
	}

	return nil
}

func (m *Merge) AlreadyMerged() bool {
	return m.baseObjId == m.rightObjId
}

func (m *Merge) FastForward() bool {
	return m.baseObjId == m.leftObjId
}

// A <- B [master] <= HEAD
//       \
//         C <- D    [topic]
//この時すでにbaseとなるBはマージ先のtopicに含まれていて、B=left=baseとなるので
//leftとrightの差分をとって,HEADをrightに合わせればいい
func (m *Merge) HandleFastForward(w io.Writer) error {
	aShortObjId := m.repo.d.ShortObjId(m.leftObjId)
	bShortObjId := m.repo.d.ShortObjId(m.rightObjId)

	w.Write([]byte(fmt.Sprintf("Updating %s..%s\n", aShortObjId, bShortObjId)))
	w.Write([]byte("Fast-forward\n"))

	l := lock.NewFileLock(m.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := m.repo.i.Load()
	if err != nil {
		return err
	}

	treeDiff, err := TreeDiffGenerateAndCompareCommit(m.leftObjId, m.rightObjId, m.repo)
	if err != nil {
		return err
	}

	// base->mergeの差分をtargetに適用するのが3wayMergeの本質
	mig := GenerateMigration(treeDiff, m.repo)
	err = mig.ApplyChanges()
	if err != nil {
		return err
	}

	//indexもアップデート
	err = m.repo.i.Write(m.repo.i.Path)
	if err != nil {
		return err
	}

	//Headをupdate,HEAD -> refs/heads/masterなので、masterが指す先をrightObjIdにする
	m.repo.r.UpdateHead(m.rightObjId)

	return nil

}

var AlreadyMergedMessage = "Already up to date\n"
var MergeFaildMessage = "Automatic merge failed: fix conflicts and then commit the result.\n"

func HandleContinue(pc *PendingCommit, mc MergeCommand, repo *Repository, w io.Writer) error {
	//conflict後マージを再開するとき
	err := repo.i.Load()
	if err != nil {
		return err
	}

	//ResumeMergeのThere is no merge~はmerge continueでしか行われない、なぜなら、
	//commitのResumeMergeはinProgressの時にしか起きない->conflict解消前の時のみ、解消前ではno mergeとはならない
	//merge --continueの時はフラグを立てればconflict解消後でもresumeMergeは起動する
	err = ResumeMerge(
		mc.Name,
		mc.Email,
		pc,
		repo,
		w,
	)
	if err != nil {
		return err
	}

	return nil

}

var INPROGRESS_MESSAGE = "Merging is not possible because youy unmerged files"

func HandleInProgressMerge(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("error: %s\n", INPROGRESS_MESSAGE)))
	w.Write([]byte(CONFLICT_MESSAGE))
	w.Write([]byte("\n"))

}

//abortはmergeでconflictする前のHEADの状態に戻す
func HandleAbort(pc *PendingCommit, mc MergeCommand, repo *Repository, w io.Writer) error {
	err := pc.Clear()
	if err != nil {
		return err
	}

	l := lock.NewFileLock(repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err = repo.i.Load()
	if err != nil {
		return err
	}

	headObjId, err := repo.r.ReadHead()
	if err != nil {
		return err
	}

	//元のHEADの状態に戻す
	err = HanldeHard(headObjId, repo)
	if err != nil {
		return err
	}

	err = repo.i.Write(repo.i.Path)
	if err != nil {
		return err
	}
	return nil
}

func RunMerge(mc MergeCommand, m *Merge, w io.Writer) error {

	//3-wayMerge開始時にcommit中断用のファイルを作る
	pc := GeneratePendingCommit(filepath.Join(mc.RootPath, ".git"))

	//--abortの時
	if mc.Option.hasAbort {
		return HandleAbort(pc, mc, m.repo, w)
	}

	//--continueの時
	//conflictでcommit出来なかったやつをここでcommitしてMergeHeadを消す,--continueの時はここで終わり
	//conflictのあとはadd . -> merge --c or commitで解消する、merge --cのときはこっち
	if mc.Option.hasContinue {
		return HandleContinue(pc, mc, m.repo, w)
	}

	//まだconflict中だったら
	//Startより先にInProgressを判断しないと、StartでMergeHeadを作るので常にInProgressになるので、
	//InProgressを先に持ってくる
	if pc.InProgress() {
		HandleInProgressMerge(w)
		return nil
	}

	err := pc.Start(m.rightObjId, mc.Message)
	if err != nil {
		return err
	}
	//Null Merge
	if m.AlreadyMerged() {
		w.Write([]byte(AlreadyMergedMessage))
		return nil
	}

	if m.FastForward() {
		err := m.HandleFastForward(w)
		return err
	}

	err = m.ResolveMerge(w)
	if err != nil {
		return err
	}

	//ConflictがあるならIndexを更新するまでで、新しくCommitは作らない
	//migartionのときにdetectするのは現在のworkspace<->index,index<->commit間
	//対してmergeのconflictDetectionは異なるcommit間なのでDetectionの範囲が違う
	if m.repo.i.IsConflicted() {
		w.Write([]byte(MergeFaildMessage))
		return nil
	}

	err = m.CommitMerge(mc.Name, mc.Email, mc.Message)
	if err != nil {
		return err
	}

	//Commitが正常に終了した場合はMerge_HEAD等を削除する
	err = pc.Clear()
	if err != nil {
		return err
	}

	return nil
}

type MergeOption struct {
	hasContinue bool
	hasAbort    bool
}

type MergeCommand struct {
	RootPath string
	Name     string
	Email    string
	Message  string
	Args     []string
	Option   MergeOption
}

func StartMerge(mc MergeCommand, w io.Writer) error {

	gitPath := filepath.Join(mc.RootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(mc.RootPath, gitPath, dbPath)

	m, err := GenerateMerge("HEAD", mc.Args[0], repo)
	if err != nil {
		return err
	}

	return ers.HandleWillWriteError(RunMerge(mc, m, w), w)

}
