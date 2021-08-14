package src

import (
	"errors"
	"fmt"
	"io/ioutil"
	"mygit/src/database/lock"
	ers "mygit/src/errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type PendingCommit struct {
	Path string
}

type PendingType string

const (
	PENDING_MERGE_TYPE      PendingType = ":merge"
	PEDING_CHERRY_PICK_TYPE             = ":cherry_pick"
	PENDING_REVERT_TYPE                 = ":revert"
)

var typesMap = map[PendingType]string{
	PENDING_MERGE_TYPE:      "MERGE_HEAD",
	PEDING_CHERRY_PICK_TYPE: "CHERRY_PICK_HEAD",
	PENDING_REVERT_TYPE:     "REVERT_HEAD",
}

var Merge_HEAD = "Merge_HEAD"
var Merge_MSG = "Merge_MSG"

var ErrorInvalidMergeType = errors.New("ErrorInvalidMMergeType")

func (p *PendingCommit) GetMergePathFromType(mergeType PendingType) (string, error) {
	mType, ok := typesMap[mergeType]
	if !ok {
		return "", ErrorInvalidMergeType
	}

	return mType, nil
}

func (p *PendingCommit) GetHeadPath(mergeType PendingType) (string, error) {
	mType, err := p.GetMergePathFromType(mergeType)
	if err != nil {
		return "", err
	}

	return filepath.Join(p.Path, mType), nil
}

func (p *PendingCommit) GetMessagePath() string {
	return filepath.Join(p.Path, Merge_MSG)
}

func GeneratePendingCommit(path string) *PendingCommit {

	return &PendingCommit{
		Path: path,
	}
}

func (p *PendingCommit) GetMergeObjId(mergeType PendingType) (string, error) {

	mPath, err := p.GetHeadPath(mergeType)
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadFile(mPath)

	if err == nil {
		return strings.TrimSpace(string(content)), nil
	}

	if errors.Is(err, syscall.ENOENT) {
		return "", &ers.FileNotExistOnConflictError{
			Message: fmt.Sprintf("There is no merge in progress (%s missng).\n", filepath.Base(mPath)),
		}
	}

	return "", err

}

func (p *PendingCommit) GetMergeMessage() (string, error) {

	content, err := ioutil.ReadFile(p.GetMessagePath())
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

func (p *PendingCommit) Start(objId, message string, mergeType PendingType) error {

	path, err := p.GetHeadPath(mergeType)
	if err != nil {
		return err
	}

	err = p.Write(path, objId)
	if err != nil {
		return err
	}

	err = p.Write(p.GetMessagePath(), message)
	if err != nil {
		return err
	}

	return nil
}

func (p *PendingCommit) Clear(mergeType PendingType) error {

	headPath, err := p.GetHeadPath(mergeType)
	if err != nil {
		return err
	}

	if stat, _ := os.Stat(headPath); stat == nil {
		return &ers.FileNotExistOnConflictError{
			Message: fmt.Sprintf("There is no merge to abort (%s missing).", filepath.Base(headPath)),
		}
	}
	err = os.RemoveAll(headPath)
	if err != nil {
		return err
	}

	messagePath := p.GetMessagePath()

	if stat, _ := os.Stat(messagePath); stat == nil {
		return &ers.FileNotExistOnConflictError{
			Message: fmt.Sprintf("There is no merge to abort (%s missing).", filepath.Base(messagePath)),
		}
	}
	err = os.RemoveAll(messagePath)
	if err != nil {
		return err
	}

	return nil
}

func (p *PendingCommit) GetMergeType() (string, error) {
	for _, path := range typesMap {
		absPath := filepath.Join(p.Path, path)
		stat, _ := os.Stat(absPath)

		if stat != nil {
			return path, nil
		}
	}

	return "", ErrorInvalidMergeType
}

func (p *PendingCommit) InProgress() bool {

	for _, path := range typesMap {
		absPath := filepath.Join(p.Path, path)
		stat, _ := os.Stat(absPath)

		if stat != nil {
			return true
		}
	}

	return false
}
