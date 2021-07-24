package src

import (
	"fmt"
	"io"
	"mygit/src/database"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	"path/filepath"
)

func StartCheckout(rootPath string, args []string, w io.Writer) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	indexPath := filepath.Join(gitPath, "index")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	target := args[0]

	currentRef, err := repo.r.CurrentRef("HEAD")
	if err != nil {
		return err
	}
	currentObjId, err := currentRef.ReadObjId()
	if err != nil {
		return err
	}

	l := lock.NewFileLock(indexPath)
	l.Lock()
	defer l.Unlock()

	err = repo.i.Load()
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
	err = repo.r.SetHead(target, targetObjId)
	if err != nil {
		return err
	}

	updatedCurrentRef, err := repo.r.CurrentRef("HEAD")
	if err != nil {
		return err
	}

	err = PrintPreviousHead(currentObjId, targetObjId, currentRef, repo, w)
	if err != nil {
		return err
	}
	err = PrintDetachMentNotice(target, currentRef, updatedCurrentRef, w)
	if err != nil {
		return err
	}
	err = PrintNewHead(target, targetObjId, currentRef, updatedCurrentRef, repo, w)
	if err != nil {
		return err
	}
	return nil
}

func PrintPreviousHead(currentObjId, targetObjId string, currentRef *database.SymRef, repo *Repository, w io.Writer) error {
	//previousHeadはHEADがdirectCommitのときで、HEADが指しているCommitを離れてしまうと参照が難しくなるから
	if currentRef.IsHead() && currentObjId != targetObjId {
		err := PrintHeadPosition("Previous HEAD position was", currentObjId, repo, w)

		if err != nil {
			return err
		}
	}

	return nil
}

func PrintHeadPosition(message, objId string, repo *Repository, w io.Writer) error {
	o, err := repo.d.ReadObject(objId)
	if err != nil {
		return err
	}

	c, ok := o.(*con.CommitFromMem)

	if !ok {
		return ErrorObjeToEntryConvError
	}

	w.Write([]byte(fmt.Sprintf("%s %s %s\n", message, repo.d.ShortObjId(c.ObjId), c.GetFirstLineMessage())))

	return nil

}

var DETACHED_HEAD_MESSAGE = `
 You are in detached HEAD state. You can look...

 mygit branch <new-branch-name>
`

func PrintDetachMentNotice(target string, currentRef, updatedRef *database.SymRef, w io.Writer) error {
	//DetachMentNoticeは、
	//checkout前がDeatchedHEADではなく(=HEADがRefを指している)
	//checkout後がDetachedHEAD(HEADが直接ObjIdを指している)

	if !currentRef.IsHead() && updatedRef.IsHead() {
		w.Write([]byte(fmt.Sprintf("Note: checking out '%s'\n", target)))
		w.Write([]byte("\n"))
		w.Write([]byte(DETACHED_HEAD_MESSAGE + "\n"))
		w.Write([]byte("\n"))
	}
	return nil
}

func PrintNewHead(target, targetObjId string, currentRef, updatedRef *database.SymRef, repo *Repository, w io.Writer) error {
	if updatedRef.IsHead() {
		//checkout後がDetachedHeadの時
		err := PrintHeadPosition("HEAD is now at", targetObjId, repo, w)
		if err != nil {
			return err
		}
	} else if currentRef.Path == updatedRef.Path {
		//同じブランチにチェックアウトした時
		w.Write([]byte(fmt.Sprintf("Already on '%s'\n", target)))
	} else {
		//ここは普通にcheckout後HEADがRefを指しているとき
		w.Write([]byte(fmt.Sprintf("Switched to branch '%s'\n", target)))
	}
	return nil
}
