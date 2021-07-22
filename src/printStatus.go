package src

import (
	"errors"
	"fmt"
	"io"
	"mygit/src/database/util"
)

const (
	INDEX_DELETE = iota
	INDEX_MODIFIED
	INDEX_ADDED
	WORKSPACE_DELETE
	WORKSPACE_MODIFIED
	WORKSPACE_ADDED
)

func (s *Status) WriteStatus(w io.Writer, isLong bool) error {
	util.SortStringSlice(s.Changed)
	util.SortStringSlice(s.Untracked)

	if isLong {
		s.WriteLongStatus(w)
	} else {
		s.WritePorcelainStatus(w)
	}

	return nil
}

func (s *Status) WritePorcelainStatus(w io.Writer) {
	for _, p := range s.Changed {
		status := s.GenerateStatus(p)
		w.Write([]byte(fmt.Sprintf("%s %s\n", status, p)))
	}

	for _, p := range s.Untracked {
		w.Write([]byte(fmt.Sprintf("?? %s\n", p)))
	}
}

var (
	IndexChangeMessage     = "Changes to be Commited"        //commitとindexに違いが生じたとき
	WorkSpaceChangeMessage = "Changes not staged for commit" //indexにあってworkspaceにある
	UntrackFileMessage     = "Untracked files"               // indexになくてworkspaceにある
	LongAdded              = "new file:"
	LongDeleted            = "deleted:"
	LongModified           = "modified:"
	LABELWIDTH             = 20
	CommitStatusWorkSpace  = "no changes added to commit"
	CommitStatusUntracked  = "nothing added to commit but untracked files present"
	CommitStatusNothing    = "nothing to commit, working tree clean"
)

func (s *Status) WriteLongStatus(w io.Writer) {

	s.GenerateChangesMessage(IndexChangeMessage, s.IndexChanges, w)
	s.GenerateChangesMessage(WorkSpaceChangeMessage, s.WorkSpaceChanges, w)
	s.GenerateChangesMessage(UntrackFileMessage, s.Untracked, w)

	s.PrintCommitStatus(w)
}

//PrinteCommitStatusは次にcommitしたときにどうなるかを表す、Commitの対象がある時(add .済み)の時は
//何も表示しない
//IndexとWorkSpaceの間に変化があってadd .していないときはメッセージ
func (s *Status) PrintCommitStatus(w io.Writer) {
	if len(s.IndexChanges) != 0 {
		//Index <-> Commit間の変化
		return
	}

	if len(s.WorkSpaceChanges) != 0 {
		//Index <-> CommWorkSpaced間の変化(M,D)
		w.Write([]byte(CommitStatusWorkSpace))
	} else if len(s.Untracked) != 0 {
		//Index <-> CommWorkSpaced間の変化(??)
		w.Write([]byte(CommitStatusUntracked))
	} else {
		//変化なし
		w.Write([]byte(CommitStatusNothing))
	}
}

func (s *Status) GenerateChangesMessage(message string, changeSet interface{}, w io.Writer) {

	switch v := changeSet.(type) {
	case []string:
		w.Write([]byte(GenerateChangesMessageForUntaracked(message, v)))
	case map[string]int:
		w.Write([]byte(GenerateChangesMessageForChangeSet(message, v)))
	default:
		break
	}
}

//色とかless対応はまたあとで
func GenerateChangesMessageForChangeSet(message string, changeSet map[string]int) string {
	if len(changeSet) == 0 {
		return ""
	}

	var content string

	content += message + ":\n"

	sortedKey := util.SortedMapKey(changeSet)

	for _, k := range sortedKey {
		status := GetStatusString(changeSet[k], true)
		content += fmt.Sprintf("\t%s%s", status, k)
	}

	content += "\n"

	return content
}

func GenerateChangesMessageForUntaracked(message string, untracked []string) string {
	if len(untracked) == 0 {
		return ""
	}

	var content string

	content += message + ":\n"
	for _, v := range untracked {
		content += fmt.Sprintf("\t%s%s", " ", v)
	}

	content += "\n"

	return content
}

var ErrorInvalidChanges = errors.New("invalid Changes and Changed consistency")

func (s *Status) GenerateStatus(path string) string {
	//rangeであるやつだけここに来るので、mapの存在チェックはいらない
	left := s.GetStatusFromChanges(INDEX, path)
	right := s.GetStatusFromChanges(WORKSPACE, path)

	return left + right
}

var (
	INDEX     = "index"
	WORKSPACE = "workspace"
)

func (s *Status) GetStatusFromChanges(setType string, path string) string {
	if setType == INDEX {
		change, ok := s.IndexChanges[path]
		if !ok {
			return " "
		} else {
			return GetStatusString(change, false)
		}
	} else {
		change, ok := s.WorkSpaceChanges[path]
		if !ok {
			return " "
		} else {
			return GetStatusString(change, false)
		}
	}
}

func GetStatusString(status int, isLong bool) string {
	if isLong {
		switch status {
		case INDEX_ADDED:
			return PaddingSpace(LongAdded, LABELWIDTH)
		case INDEX_MODIFIED:
			return PaddingSpace(LongModified, LABELWIDTH)
		case INDEX_DELETE:
			return PaddingSpace(LongDeleted, LABELWIDTH)
		case WORKSPACE_MODIFIED:
			return PaddingSpace(LongModified, LABELWIDTH)
		case WORKSPACE_DELETE:
			return PaddingSpace(LongDeleted, LABELWIDTH)
		default:
			return " "
		}
	} else {
		switch status {
		case INDEX_ADDED:
			return "A"
		case INDEX_MODIFIED:
			return "M"
		case INDEX_DELETE:
			return "D"
		case WORKSPACE_MODIFIED:
			return "M"
		case WORKSPACE_DELETE:
			return "D"
		default:
			return " "
		}
	}

}

func PaddingSpace(message string, paddingWidth int) string {
	return fmt.Sprintf("%*s", paddingWidth, message)
}
