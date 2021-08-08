package src

import (
	"fmt"
	"io"
	"mygit/src/database/lock"
	ers "mygit/src/errors"
	"os"
	"path/filepath"
)

type RmOption struct {
	hasCached    bool
	hasForce     bool
	hasRecursive bool
}

var (
	BOTH_CHANGED      = "staged content different from both the file and the HEAD"
	INDEX_CHANGED     = "changes staged in the index"
	WORKSPACE_CHANGED = "local modifications"
)

type Rm struct {
	Inspector   *Inspector
	UnCommitted []string
	UnStaged    []string
	BothChanged []string
	repo        *Repository
	option      *RmOption
}

func GenerateRm(repo *Repository, option *RmOption) *Rm {
	return &Rm{
		repo: repo,
		Inspector: &Inspector{
			repo: repo,
		},
		option: option,
	}
}

//エラーに伴わないやつなら一か所に集めないでWriteしてもよさそうか...?
func RemoveFile(path string, option *RmOption, repo *Repository, w io.Writer) error {
	//workSpaceとindexからremove
	repo.i.Remove(path)

	if !option.hasCached {
		err := repo.w.Remove(path)
		if err != nil {
			return err
		}
	}

	w.Write([]byte(fmt.Sprintf("rm %s\n", path)))

	return nil
}

func (r *Rm) CreateErrorForUncommitedAndUnstaged() error {

	var errorMessage string

	for _, errorStruct := range []struct {
		message string
		paths   []string
	}{
		{
			message: BOTH_CHANGED,
			paths:   r.BothChanged,
		},
		{
			message: INDEX_CHANGED,
			paths:   r.UnCommitted,
		},
		{
			message: WORKSPACE_CHANGED,
			paths:   r.UnStaged,
		},
	} {
		if len(errorStruct.paths) == 0 {
			continue
		}
		var filesMessage string
		if len(errorStruct.paths) == 1 {
			filesMessage = "file has"
		} else {
			filesMessage = "files have"
		}

		errorMessage += fmt.Sprintf("error: the following %s %s:\n", filesMessage, errorStruct.message)

		for _, path := range errorStruct.paths {
			errorMessage += fmt.Sprintf("   %s\n", path)
		}
	}

	return &ers.InvalidIndexPathOnRemovalError{
		Message: errorMessage,
	}

}

func (r *Rm) PlanRemoval(path string) error {

	if r.option.hasForce {
		//forceならplanせずに強制実行
		return nil
	}

	stat, _ := r.repo.w.StatFile(path)

	headObjId, err := r.repo.r.ReadHead()
	if err != nil {
		return err
	}

	e, err := r.repo.d.LoadTreeEntryWithPath(headObjId, path)
	if err != nil {
		return err
	}

	indexEntry, _ := r.repo.i.EntryForPath(path)

	stagedChange := r.Inspector.CompareTreeToIndex(e, indexEntry)
	unStagedChange, err := r.Inspector.CompareIndextoWorkSpace(indexEntry, stat)

	if err != nil {
		return err
	}

	if stagedChange != "" && unStagedChange != "" {
		//index<->commit , index <-> workspace両方とも変わっているとき
		r.BothChanged = append(r.UnStaged, path)
	} else if stagedChange != "" {
		r.UnCommitted = append(r.UnCommitted, path)
	} else if unStagedChange != "" {
		r.UnStaged = append(r.UnStaged, path)

	}

	return nil
}

func RunExamine(path string, hasRecursive bool, repo *Repository) ([]string, error) {
	if repo.i.IsIndexedDir(path) {
		if hasRecursive {
			//-r があるなら指定したDirのchildrenを全部削除
			return repo.i.ChildPaths(path), nil
		} else {
			//-r optionなしではdirはremove出来ない
			return nil, &ers.InvalidIndexPathOnRemovalError{
				Message: fmt.Sprintf("not removing '%s' recursively without -r", path),
			}
		}
	}

	if repo.i.IsIndexedFile(path) {
		return []string{path}, nil
	}

	return nil, &ers.InvalidIndexPathOnRemovalError{
		Message: fmt.Sprintf("pathspec '%s' did not match any files", path),
	}
}

func ExaminePath(hasRecursive bool, args []string, repo *Repository) ([]string, error) {
	var examinedPaths []string
	for _, path := range args {
		paths, err := RunExamine(path, hasRecursive, repo)
		if err != nil {
			return nil, err
		}

		examinedPaths = append(examinedPaths, paths...)
	}

	return examinedPaths, nil
}

func (rm *Rm) CheckRmError() error {

	if len(rm.UnCommitted) != 0 || len(rm.UnStaged) != 0 || len(rm.BothChanged) != 0 {
		return rm.CreateErrorForUncommitedAndUnstaged()
	}

	return nil
}

func RunRm(args []string, option *RmOption, repo *Repository, w io.Writer) error {
	rm := GenerateRm(repo, option)

	examinedPaths, err := ExaminePath(option.hasRecursive, args, repo)

	if err != nil {
		return err
	}

	for _, path := range examinedPaths {
		err := rm.PlanRemoval(path)
		if err != nil {
			return err
		}
	}
	//planRemovalの結果rm.untrackedやrm.unstaged,rm.bothChangedが空でなければエラー
	err = rm.CheckRmError()
	if err != nil {
		return err
	}

	for _, path := range examinedPaths {
		err := RemoveFile(path, option, repo, w)
		if err != nil {
			return err
		}
	}

	err = repo.i.Write(repo.i.Path)
	if err != nil {
		return err
	}

	return nil

}

//RmはHEADと一致しているやつのみremove(これならもしrmしてもcommit済みということなので、そこのobjectsから取り出して、workspaceとindexを元に戻せるから)
func StartRm(rootPath string, args []string, option *RmOption, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")

	repo := GenerateRepository(rootPath, gitPath, dbPath)

	_, indexNonExist := os.Stat(repo.i.Path)

	l := lock.NewFileLock(repo.i.Path)
	l.Lock()
	defer l.Unlock()

	if indexNonExist == nil {
		//.git/indexがある場合のみLoad、newFileLockで存在しないならindexを作ってしまうのでStatの後にしなければならない
		err := repo.i.Load()
		if err != nil {
			return err
		}
	}

	return ers.HandleWillWriteError(RunRm(args, option, repo, w), w)

}
