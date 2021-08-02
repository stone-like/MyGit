package src

import (
	"errors"
	"fmt"
	"io"
	"mygit/src/database/util"
	"reflect"

	u "mygit/util"
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
		err := s.WriteLongStatus(w)
		if err != nil {
			return err
		}
	} else {
		err := s.WritePorcelainStatus(w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Status) WritePorcelainStatus(w io.Writer) error {
	for _, p := range s.Changed {
		//ConflictしているやつもChnagesに入っているので,GenerateStatusで処理してもらう

		status, err := s.GenerateStatus(p)
		if err != nil {
			return err
		}
		w.Write([]byte(fmt.Sprintf("%s %s\n", status, p)))
	}

	for _, p := range s.Untracked {
		w.Write([]byte(fmt.Sprintf("?? %s\n", p)))
	}

	return nil
}

var (
	IndexChangeMessage     = "Changes to be Commited"        //commitとindexに違いが生じたとき
	WorkSpaceChangeMessage = "Changes not staged for commit" //indexにあってworkspaceにある
	UntrackFileMessage     = "Untracked files"               // indexになくてworkspaceにある
	ConflictedMessage      = "Unmerged paths"                //現在のindexがconflict状態の時
	LongAdded              = "new file:"
	LongDeleted            = "deleted:"
	LongModified           = "modified:"
	LABELWIDTH             = 20
	CONFLICT_LABELWIDTH    = 17
	CommitStatusWorkSpace  = "no changes added to commit"
	CommitStatusUntracked  = "nothing added to commit but untracked files present"
	CommitStatusNothing    = "nothing to commit, working tree clean"
)

var ErrorInvalidLabelType = errors.New("Invalid LabelType")

var Normal = ":normal"
var Conflict = ":conflict"

var LongStatus = map[[3]int]string{
	{INDEX_DELETE}:       LongDeleted,
	{INDEX_MODIFIED}:     LongModified,
	{INDEX_ADDED}:        LongAdded,
	{WORKSPACE_MODIFIED}: LongModified,
	{WORKSPACE_ADDED}:    LongAdded,
}

//sliceは比較不可なのでkeyにはできないがarrayならOK
var ConflictLongStatus = map[[3]int]string{
	{1, 2, 3}: "both modified:",
	{1, 2}:    "deleted by them:",
	{1, 3}:    "deleted by us:",
	{2, 3}:    "both added:",
	{2}:       "added by us:",
	{3}:       "added by them:",
}

func GetConflictLongStatus(key []int) (string, error) {

	var temp [3]int
	copy(temp[:], key)

	for intSet, str := range ConflictLongStatus {
		if reflect.DeepEqual(intSet, temp) {
			return str, nil
		}
	}

	return "", ErrorInvalidLabelType
}

var ConflictShortStatus = map[[3]int]string{
	{1, 2, 3}: "UU",
	{1, 2}:    "UD",
	{1, 3}:    "DU",
	{2, 3}:    "AA",
	{2}:       "AU",
	{3}:       "UA",
}

func GetConflictShortStatus(key []int) (string, error) {

	var temp [3]int
	copy(temp[:], key)

	for intSet, str := range ConflictShortStatus {
		if reflect.DeepEqual(intSet, temp) {
			return str, nil
		}
	}

	return "", ErrorInvalidLabelType
}

var StatusMap = map[string]map[[3]int]string{
	Normal:   LongStatus,
	Conflict: ConflictLongStatus,
}

var WidthMap = map[string]int{
	Normal:   LABELWIDTH,
	Conflict: CONFLICT_LABELWIDTH,
}

func (s *Status) WriteLongStatus(w io.Writer) error {

	err := s.GenerateChangesMessage(IndexChangeMessage, s.IndexChanges, w)
	if err != nil {
		return err
	}
	err = s.GenerateChangesMessage(ConflictedMessage, s.Conflicts, w)
	if err != nil {
		return err
	}
	err = s.GenerateChangesMessage(WorkSpaceChangeMessage, s.WorkSpaceChanges, w)
	if err != nil {
		return err
	}
	err = s.GenerateChangesMessage(UntrackFileMessage, s.Untracked, w)
	if err != nil {
		return err
	}

	s.PrintCommitStatus(w)

	return nil
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

//全部動作確認出来たら、[]string untrackedとかtypeにしてそこからGenerate~メソッドをはやす方向にリファクタリング
func (s *Status) GenerateChangesMessage(message string, changeSet interface{}, w io.Writer) error {

	switch v := changeSet.(type) {
	case []string:
		w.Write([]byte(GenerateChangesMessageForUntaracked(message, v)))
	case map[string]int:
		w.Write([]byte(GenerateChangesMessageForChangeSet(message, v)))
	case map[string][]int:
		content, err := GenerateChangesMessageForConflicts(message, v)
		if err != nil {
			return err
		}
		w.Write([]byte(content))
	default:
		break
	}
	return nil
}

func GenerateChangesMessageForConflicts(message string, changeSet map[string][]int) (string, error) {
	if len(changeSet) == 0 {
		return "", nil
	}

	var content string

	content += message + ":\n"

	for _, k := range u.SortedKeys(changeSet) {
		status, err := GetConflictStatusString(changeSet[k])
		if err != nil {
			return "", err
		}
		content += fmt.Sprintf("\t%s%s", status, k)
	}

	content += "\n"

	return content, nil
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

func (s *Status) GenerateStatus(path string) (string, error) {

	ints, ok := s.Conflicts[path]

	if ok {
		return GetConflictShortStatus(ints)
	}
	//rangeであるやつだけここに来るので、mapの存在チェックはいらない
	left := s.GetStatusFromChanges(INDEX, path)
	right := s.GetStatusFromChanges(WORKSPACE, path)

	return left + right, nil
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

func GetConflictStatusString(status []int) (string, error) {

	content, err := GetConflictLongStatus(status)
	if err != nil {
		return "", err
	}
	return PaddingSpace(content, CONFLICT_LABELWIDTH), nil

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
