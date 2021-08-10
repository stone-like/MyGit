package src

import (
	"errors"
	"fmt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"mygit/src/database/util"
	e "mygit/src/errors"
	"path/filepath"
	"strconv"
	"strings"
)

type Ref struct {
	Name string
}

func (rf *Ref) ToString() string {
	return ""
}

type Parent struct {
	Rev       BranchObj
	ParentNum int
}

func (rv *Parent) ToString() string {
	return ""
}

type Ancestor struct {
	Rev BranchObj
	N   int
}

func (a *Ancestor) ToString() string {
	return ""
}

type BranchObj interface {
	ToString() string
}

var aliasMap = map[string]string{
	"@": "HEAD",
}

var (
	PARENT   = `^(.+)\^(\d*)$`
	ANCESTOR = `^(.+)~(\d+)$`
)

func CommitParentWithMultipleParentVersion(objId string, parentNum int, repo *Repository) (string, error) {
	return CommitParents(objId, parentNum, repo)
}

func CommitParent(objId string, repo *Repository) (string, error) {
	return CommitParents(objId, 1, repo)
}

func CommitParents(objId string, parentNum int, repo *Repository) (string, error) {
	o, err := LoadTypedObject(objId, "commit", repo)

	if err != nil {
		return "", err
	}
	//LoadTypedObjectでコミットであることは確定
	c, _ := o.(*con.CommitFromMem)

	return c.Parents[parentNum-1], nil

}

func LoadTypedObject(objId, objType string, repo *Repository) (con.ParsedObj, error) {
	o, err := repo.d.ReadObject(objId)
	if err != nil {
		return nil, err
	}

	switch objType {
	case "commit":
		{
			c, ok := o.(*con.CommitFromMem)

			if !ok {
				return nil, &e.ObjConvertionError{
					Type:    "commit",
					Message: fmt.Sprintf("object %s is a %s, not a %s", objId, o.Type(), objType),
				}
			}

			return c, nil
		}

	default:
		{
			return nil, ErrorObjeToEntryConvError
		}
	}
}

func ResolveRev(obj BranchObj, repo *Repository) (string, error) {
	switch v := obj.(type) {
	case *Parent:
		objId, err := ResolveRev(v.Rev, repo)
		if err != nil {
			return "", err
		}

		targetObjId, err := CommitParentWithMultipleParentVersion(objId, v.ParentNum, repo)
		if err != nil {
			return "", AddInfoToObjConvertionError(objId, err)
		}
		return targetObjId, nil
	case *Ancestor:
		objId, err := ResolveRev(v.Rev, repo)
		if err != nil {
			return "", err
		}

		targetObjId := objId
		for i := 0; i < v.N; i++ {
			target, err := CommitParent(targetObjId, repo)
			if err != nil {
				return "", AddInfoToObjConvertionError(targetObjId, err)
			}
			targetObjId = target
		}
		return targetObjId, nil
	case *Ref:
		targetObjId, err := ResolveRef(v, repo)
		if err != nil {
			return "", AddInfoToObjConvertionError(v.Name, err)
		}

		//objIdがコミットかチェック
		_, err = LoadTypedObject(targetObjId, "commit", repo)

		if err != nil {
			return "", AddInfoToObjConvertionError(v.Name, err)
		}

		return targetObjId, nil

	default:
		return "", con.ErrorUndefinedGitObjectType

	}
}

func AddInfoToObjConvertionError(targetName string, err error) error {
	switch er := err.(type) {
	case *e.ObjConvertionError:
		{
			er.CriticalInfo = fmt.Sprintf("Not a valid object name: %s", targetName)
			return er
		}
	case *e.InvalidObjectError:
		{
			er.CriticalInfo = fmt.Sprintf("Not a valid object name: %s", targetName)
			return er
		}
	default:
		return err
	}

}

var ErrorNonExistObjId = errors.New("non exists objId")

func ResolveRef(r *Ref, repo *Repository) (string, error) {
	objId, _ := repo.r.ReadRef(r.Name)

	if objId != "" {
		return objId, nil
	}

	//このobjDir以降によってr.nameがブランチでなくてobjIdでも処理できる
	//oidはobjectPathのdirname+pathnameなので
	objDir := repo.d.ObjDirname(r.Name)

	candicates, err := repo.w.ListFiles(objDir)
	if err != nil {
		return "", err
	}

	objIds := make([]string, 0, len(candicates))

	for _, c := range candicates {
		objId, match := PrefixMatch(r.Name, filepath.Base(objDir), filepath.Base(c))
		if match {
			objIds = append(objIds, objId)
		}
	}

	//完全なobjIdじゃなかったとしてもcandicateが一つなら通常実行
	if len(objIds) == 1 {
		return objIds[0], nil
	}

	if len(objIds) > 1 {
		return "", CreateAnbiguosSha1Message(r.Name, objIds, repo)
	}

	return "", ErrorNonExistObjId

}

func CreateAnbiguosSha1Message(name string, objIds []string, repo *Repository) error {

	var objInfo []string

	for _, id := range objIds {
		o, err := repo.d.ReadObject(id)

		if err != nil {
			return err
		}

		shortObjId := repo.d.ShortObjId(o.GetObjId())

		info := fmt.Sprintf(" %s %s", shortObjId, o.Type())

		switch v := o.(type) {
		case *con.CommitFromMem:
			objInfo = append(objInfo, info+fmt.Sprintf(" %s - %s", v.Author.ShortTime(), v.GetFirstLineMessage()))
		default:
			objInfo = append(objInfo, info)
		}

	}

	message := fmt.Sprintf("short SHA1 %s is ambiguos", name)
	hint := append([]string{"The candidates are:"}, objInfo...)

	return &e.InvalidObjectError{
		Message: message,
		Hint:    hint,
	}
}

func PrefixMatch(name, dirname, filename string) (string, bool) {
	objId := dirname + filename

	return objId, strings.HasPrefix(objId, name)
}

func ParseRev(branchName string) (BranchObj, error) {

	parentExp := util.CheckRegExpSubString(PARENT, branchName)
	if len(parentExp) != 0 {
		rev, err := ParseRev(parentExp[0][1])
		if err != nil {
			return nil, err
		}

		var parentNum int

		//targetStringと一つ目のヒットと二つ目のヒットで合計3
		if len(parentExp[0]) == 3 && parentExp[0][2] != "" {
			//parentNumまで正規表現でヒットした場合
			i, err := strconv.Atoi(parentExp[0][2])
			if err != nil {
				return nil, err
			}
			parentNum = i
		} else {
			parentNum = 1
		}

		return &Parent{
			Rev:       rev,
			ParentNum: parentNum,
		}, nil

	}

	ansExp := util.CheckRegExpSubString(ANCESTOR, branchName)
	if len(ansExp) != 0 {
		rev, err := ParseRev(ansExp[0][1])
		if err != nil {
			return nil, err
		}

		i, err := strconv.Atoi(ansExp[0][2])
		if err != nil {
			return nil, err
		}
		ans := &Ancestor{
			Rev: rev,
			N:   i,
		}

		return ans, nil
	}

	err := data.CheckValidRef(branchName)

	if err != nil {
		return nil, err
	}
	name, ok := aliasMap[branchName]

	if !ok {
		return &Ref{
			Name: branchName,
		}, nil
	} else {
		return &Ref{
			Name: name,
		}, nil
	}

}
