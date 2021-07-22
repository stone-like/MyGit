package src

import (
	"io"
	"path/filepath"
)

func StartBranch(rootPath string, args []string, w io.Writer) error {
	//gitPathとかdbPathはまとめた方がよさそう
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	if len(args) == 0 {
		//list branch
	} else if len(args) == 1 {
		branchName := args[0]
		startObjId, err := repo.r.ReadHead()
		if err != nil {
			return err
		}
		err = repo.r.CreateBranch(branchName, startObjId)
		if err != nil {
			return err
		}
	} else if len(args) == 2 {
		branchName := args[0]
		start_point := args[1]

		rev, err := ParseRev(start_point)
		if err != nil {
			return err
		}

		startObjId, err := ResolveRev(rev, repo)
		if err != nil {
			return err
		}

		err = repo.r.CreateBranch(branchName, startObjId)
		if err != nil {
			return err
		}

	}

	return nil
}
