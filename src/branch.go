package src

import (
	"fmt"
	"io"
	"math"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"path/filepath"
	"strings"
)

type BranchOption struct {
	HasV bool
	HasD bool
	HasF bool
}

func DeleteBranches(rootPath string, args []string, option *BranchOption, repo *Repository, w io.Writer) error {
	for _, p := range args {
		err := DeleteBranch(p, option, repo, w)
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteBranch(path string, option *BranchOption, repo *Repository, w io.Writer) error {
	if !option.HasF {
		return nil //forceでなければdeleteできない
	}

	objId, err := repo.r.DeleteBranch(path)
	if err != nil {
		return err
	}
	shortObjId := ShortOid(objId, repo.d)
	w.Write([]byte(fmt.Sprintf("Deleted branch %s (was %s)\n", path, shortObjId)))

	return nil

}

func ListBranch(rootPath string, option *BranchOption, repo *Repository, w io.Writer) error {

	currentRef, err := repo.r.CurrentRef("HEAD")
	if err != nil {
		return err
	}
	branches, err := repo.r.ListBranches()
	if err != nil {
		return err
	}

	var maxWidth int

	for _, symRef := range branches {
		maxWidth = int(math.Max(float64(maxWidth), float64(len(symRef.ShortName()))))
	}

	for _, symRef := range branches {
		info := formatRef(symRef, currentRef)
		if option.HasV {
			extraInfo, err := addExctraInfoToBranch(maxWidth, symRef, repo)
			if err != nil {
				return err
			}

			info += extraInfo
		}
		w.Write([]byte(fmt.Sprintf("%s\n", info)))
	}

	return nil
}

func addExctraInfoToBranch(maxWidth int, s *data.SymRef, repo *Repository) (string, error) {
	objId, err := s.ReadObjId()
	if err != nil {
		return "", err
	}
	o, err := repo.d.ReadObject(objId)
	if err != nil {
		return "", err
	}
	c, ok := o.(*con.CommitFromMem)

	if !ok {
		return "", ErrorObjeToEntryConvError
	}

	shortOid := ShortOid(c.ObjId, repo.d)
	space := strings.Repeat(" ", (maxWidth - len(s.ShortName())))

	return fmt.Sprintf("%s %s %s", space, shortOid, c.GetFirstLineMessage()), nil
}

func formatRef(s, currentRef *data.SymRef) string {
	if s.Path == currentRef.Path {
		return fmt.Sprintf("* %s", s.ShortName())
	} else {
		return fmt.Sprintf("  %s", s.ShortName())
	}
}

func StartBranch(rootPath string, args []string, option *BranchOption, w io.Writer) error {
	//gitPathとかdbPathはまとめた方がよさそう
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	if option.HasD {
		err := DeleteBranches(rootPath, args, option, repo, w)
		if err != nil {
			return err
		}
	} else if len(args) == 0 {
		//list branch
		err := ListBranch(rootPath, option, repo, w)
		if err != nil {
			return err
		}
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
