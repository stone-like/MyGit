package src

import (
	"fmt"
	"io"
	"mygit/src/database/lock"
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

func (m *Merge) ResolveMerge() error {
	l := lock.NewFileLock(m.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	err := m.repo.i.Load()
	if err != nil {
		return err
	}

	resolveMerge := GenerateResolveMerge(m)
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

func RunMerge(rootPath, name, email, mergeMessage string, m *Merge, w io.Writer) error {
	//Null Merge
	if m.AlreadyMerged() {
		w.Write([]byte(AlreadyMergedMessage))
		return nil
	}

	if m.FastForward() {
		err := m.HandleFastForward(w)
		return err
	}

	err := m.ResolveMerge()
	if err != nil {
		return err
	}

	//ConflictがあるならIndexを更新するまでで、新しくCommitは作らない
	//migartionのときにdetectするのは現在のworkspace<->index,index<->commit間
	//対してmergeのconflictDetectionは異なるcommit間なのでDetectionの範囲が違う
	if m.repo.i.IsConflicted() {
		//ここでMERGE＿HEADを後々作る
		return nil
	}

	err = m.CommitMerge(name, email, mergeMessage)
	if err != nil {
		return err
	}

	return nil
}

func StartMerge(rootPath, name, email, mergeMessage string, args []string, w io.Writer) error {

	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	m, err := GenerateMerge("HEAD", args[0], repo)
	if err != nil {
		return err
	}

	return RunMerge(rootPath, name, email, mergeMessage, m, w)

}
