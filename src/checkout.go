package src

import (
	"io"
	"mygit/src/database/lock"
	"path/filepath"
)

func StartCheckout(rootPath string, args []string, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	indexPath := filepath.Join(gitPath, "index")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	target := args[0]

	l := lock.NewFileLock(indexPath)
	l.Lock()
	defer l.Unlock()

	err := repo.i.Load()
	if err != nil {
		return err
	}

	currentObjId, err := repo.r.ReadHead()
	if err != nil {
		return err
	}

	rev, err := ParseRev(target)
	if err != nil {
		return err
	}

	targetObjId, err := ResolveRev(rev, repo)
	if err != nil {
		return err
	}

	trDiff := GenerateTreeDiff(repo)
	err = trDiff.CompareObjId(currentObjId, targetObjId)
	if err != nil {
		return err
	}

	m := GenerateMigration(trDiff, repo)
	err = m.ApplyChanges()
	if err != nil {
		return err
	}

	//indexもチェックアウト先にアップデート(もちろん先にコンフリクトチェックはあるが)
	err = repo.i.Write(repo.i.Path)
	if err != nil {
		return err
	}
	//updateHeadと、indexとworkspaceの違いがないかをtest
	err = repo.r.UpdateHead(targetObjId)
	if err != nil {
		return err
	}

	return nil
}
