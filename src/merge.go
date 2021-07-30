package src

import (
	"errors"
	data "mygit/src/database"
	"mygit/src/database/lock"
	util "mygit/src/database/util"
	"path/filepath"
)

type RedunduntCas struct {
	redundant []string
}

func FilterCommit(commits []string, objId string, r *RedunduntCas, d *data.Database) error {
	if util.Contains(r.redundant, objId) {
		return nil
	}

	willRemove := append(r.redundant, objId)

	others := util.RemovedSlice(commits, willRemove)

	cas, err := GenerateCAS(objId, others, d)

	if err != nil {
		return err
	}

	_, err = cas.FindCas()
	if err != nil {
		return err
	}

	if cas.IsMarked(objId, PARENT_TWO) {
		r.redundant = append(r.redundant, objId)
	}

	for _, otherobjId := range others {
		if cas.IsMarked(otherobjId, PARENT_ONE) {
			r.redundant = append(r.redundant, otherobjId)
		}
	}

	return nil

}

var ErrorBCANotFound = errors.New("BCA not Found")

func GetBCA(headObjId, mergeObjId string, d *data.Database) (string, error) {
	//BCAが一つもなかった時ってエラーでよさそうだけど...
	commits, err := FindBCA(headObjId, mergeObjId, d)
	if err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", ErrorBCANotFound
	}

	return commits[0], nil
}

func FindBCA(headObjId, mergeObjId string, d *data.Database) ([]string, error) {
	cas, err := GenerateCAS(headObjId, []string{mergeObjId}, d)
	if err != nil {
		return nil, err
	}
	commits, err := cas.FindCas()
	if err != nil {
		return nil, err
	}

	if len(commits) <= 1 {
		//commitsが1以下の時はBCAが見つかったか、一つもなかった時
		return commits, nil
	}

	r := &RedunduntCas{}

	//commitsが2以上ある場合
	for _, objId := range commits {
		err := FilterCommit(commits, objId, r, d)
		if err != nil {
			return nil, err
		}
	}

	return util.RemovedSlice(commits, r.redundant), nil

}

func StartMerge(rootPath, name, email, mergeMessage string, args []string) error {

	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	headObjId, err := repo.r.ReadHead()
	if err != nil {
		return err
	}

	rev, err := ParseRev(args[0])
	if err != nil {
		return err
	}
	mergeObjId, err := ResolveRev(rev, repo)
	if err != nil {
		return err
	}

	baseObjId, err := GetBCA(headObjId, mergeObjId, repo.d)
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

	treeDiff, err := TreeDiffGenerateAndCompareCommit(baseObjId, mergeObjId, repo)
	if err != nil {
		return err
	}

	// base->mergeの差分をtargetに適用するのが3wayMergeの本質
	m := GenerateMigration(treeDiff, repo)
	err = m.ApplyChanges()
	if err != nil {
		return err
	}

	//indexもアップデート
	err = repo.i.Write(repo.i.Path)
	if err != nil {
		return err
	}

	err = WriteCommit([]string{headObjId, mergeObjId}, name, email, mergeMessage, repo)
	if err != nil {
		return err
	}

	return nil
}
