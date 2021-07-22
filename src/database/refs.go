package database

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"mygit/src/database/lock"
	"mygit/src/database/util"
	"os"
	"path/filepath"
	"strings"
)

type Refs struct {
	Path string
}

//branch名には制約がある
//ダメな例
// 名前が"."で始まる
// 名前に".."が含まれる
// ASCIIの制御文字が含まれる
// ":","?","[","\","^","~",SPACE,\tが含まれる
// "*"が含まれる（REFNAME_REFSPEC_PATTERNがセットされていればOK）
// "/"で終わる
// ".lock"で終わる
// "@{"を含む
var (
	ErrorAsciiControlContained  = errors.New("asciiControlContained")
	ErrorInitialDot             = errors.New("initialDotContained")
	ErrorPathComponentContained = errors.New("pathComponentContained")
	ErrorDiskTraversal          = errors.New("DiskTraversalContained")
	ErrorInitialSlash           = errors.New("initialSlashContained")
	ErrorTailSlash              = errors.New("tailSlashContained")
	ErrorExtIsLock              = errors.New("extIsLockError")
	ErrorRevisionContained      = errors.New("revisionComponentContained")
	ErrorBranchAlreadyExists    = errors.New("branchAlreadyExists")
)

func (r *Refs) CreateBranch(branchName string, startObjId string) error {

	err := CheckValidRef(branchName)
	if err != nil {
		return err
	}

	path := filepath.Join(r.HeadsPath(), branchName)

	if _, err := os.Stat(path); err == nil {
		//すでに存在するbranchだったら
		return ErrorBranchAlreadyExists
	}

	// objId, err := r.ReadHead()
	// if err != nil {
	// 	return err
	// }
	err = r.UpdateRefFile(path, startObjId)
	if err != nil {
		return err
	}

	return nil
}

func CheckValidRef(branchName string) error {
	if util.CheckRegExp(`^\.`, branchName) {
		return ErrorInitialDot
	}

	if util.CheckRegExp(`\/\.`, branchName) {
		return ErrorPathComponentContained
	}

	if util.CheckRegExp(`\.\.`, branchName) {
		return ErrorDiskTraversal
	}

	if util.CheckRegExp(`^\/`, branchName) {
		return ErrorInitialSlash
	}

	if util.CheckRegExp(`\/$`, branchName) {
		return ErrorTailSlash
	}

	if util.CheckRegExp(`\.lock$`, branchName) {
		return ErrorExtIsLock
	}

	if util.CheckRegExp(`@\{`, branchName) {
		return ErrorRevisionContained
	}

	if util.CheckRegExp(`[\x00-\x20*:?\[\\^~\x7f]`, branchName) {
		return ErrorAsciiControlContained
	}

	return nil

}

// func CheckRegExp(reg, branchName string) bool {
// 	return regexp.MustCompile(reg).MatchString(branchName)
// }

func (r *Refs) CreateHeadPath(path string) error {
	if _, err := os.Stat(path); err != nil {

		dirPath := filepath.Dir(path)
		_, err := os.Stat(dirPath)
		if err != nil {
			//Dirが存在しないときは
			err := os.MkdirAll(dirPath, os.ModePerm)
			if err != nil {
				return err
			}
		}

		f, err := os.Create(path)
		defer f.Close()
		if err != nil {
			return err
		}

		return nil
	}

	return nil

}

func (r *Refs) UpdateRefFile(path, objId string) error {
	err := r.CreateHeadPath(path)

	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()
	if err != nil {
		return err
	}

	//RefFileUpdateの時はwriteするときロックで保護すればいいはず
	//HeadからobjIdを読むときは保護いらない->HEADのobjIdによって新しいbranchの指すobjIdは変わるけど、
	//それは同時にコミットが起きてバラバラになってしまうみたいな時困るのであって、コミットAのときにブランチがさすObjIdAで、コミットBの時にブランチが指すObjIdBとなるのは大丈夫だと思う
	//コミットの方は同時に起こらないようにロックしてある
	err = lock.Flock(path, func() {
		f.Write([]byte(objId + "\n"))
	})
	if err != nil {
		return err
	}

	return nil

}

func (r *Refs) UpdateHead(objId string) error {
	r.UpdateRefFile(r.HeadPath(), objId)
	return nil
}

var ErrorPathNotExists = errors.New("PathNotExists")

func (r *Refs) ReadRef(name string) (string, error) {
	path, isExists := r.PathForName(name)

	if !isExists {
		return "", ErrorPathNotExists
	}

	return ReadRefFile(path)

}

func ReadRefFile(name string) (string, error) {
	bytes, err := ioutil.ReadFile(name)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(bytes)), nil
}

func (r *Refs) PathForName(name string) (string, bool) {
	pref := []string{r.Path, r.RefsPath(), r.HeadsPath()}

	for _, r := range pref {
		target := filepath.Join(r, name)

		if stat, err := os.Stat(target); err != nil {
			//存在しなかったらcontinue
			continue
		} else {
			if !stat.IsDir() {
				//存在して、ファイルだったら
				return target, true
			}
		}

	}

	return "", false
}

func (r *Refs) ReadHead() (string, error) {
	if _, err := os.Stat(r.HeadPath()); err != nil {
		return "", nil
	}

	f, err := os.Open(r.HeadPath())

	if err != nil {
		return "", err
	}

	s := bufio.NewScanner(f)

	var str string
	for s.Scan() {
		str += s.Text()
	}

	return string(str), nil
}

func (r *Refs) RefsPath() string {
	return filepath.Join(r.Path, "refs")
}

func (r *Refs) HeadsPath() string {
	return filepath.Join(r.RefsPath(), "heads")
}

func (r *Refs) HeadPath() string {
	return filepath.Join(r.Path, "HEAD")
}
