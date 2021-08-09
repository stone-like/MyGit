package src

import (
	"fmt"
	"io/ioutil"
	"mygit/src/database/lock"
	ers "mygit/src/errors"
	"os"
	"path/filepath"
	"strings"
)

type PendingCommit struct {
	HeadPath    string
	MessagePath string
}

var Merge_HEAD = "Merge_HEAD"
var Merge_MSG = "Merge_MSG"

func GeneratePendingCommit(path string) *PendingCommit {
	return &PendingCommit{
		HeadPath:    filepath.Join(path, Merge_HEAD),
		MessagePath: filepath.Join(path, Merge_MSG),
	}
}

func (p *PendingCommit) GetMergeObjId() (string, error) {
	content, err := ioutil.ReadFile(p.HeadPath)
	if err != nil {
		return filepath.Base(p.HeadPath), err
	}
	return strings.TrimSpace(string(content)), nil
}

func (p *PendingCommit) GetMergeMessage() (string, error) {
	content, err := ioutil.ReadFile(p.MessagePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (p *PendingCommit) Write(path, content string) error {
	l := lock.NewFileLock(path)
	l.Lock()
	defer l.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	f.Write([]byte(content))
	f.Write([]byte("\n"))
	return nil

}

func (p *PendingCommit) Start(objId, message string) error {
	err := p.Write(p.HeadPath, objId)
	if err != nil {
		return err
	}

	err = p.Write(p.MessagePath, message)
	if err != nil {
		return err
	}

	return nil
}

func (p *PendingCommit) Clear() error {
	if stat, _ := os.Stat(p.HeadPath); stat == nil {
		return &ers.FileNotExistOnConflictError{
			Message: fmt.Sprintf("There is no merge to abort (%s missing).", filepath.Base(p.HeadPath)),
		}
	}
	err := os.RemoveAll(p.HeadPath)
	if err != nil {
		return err
	}

	if stat, _ := os.Stat(p.MessagePath); stat == nil {
		return &ers.FileNotExistOnConflictError{
			Message: fmt.Sprintf("There is no merge to abort (%s missing).", filepath.Base(p.MessagePath)),
		}
	}
	err = os.RemoveAll(p.MessagePath)
	if err != nil {
		return err
	}

	return nil
}

func (p *PendingCommit) InProgress() bool {
	stat, _ := os.Stat(p.HeadPath)
	//存在すればまだConflict中なのでtrue
	return stat != nil
}
