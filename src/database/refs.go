package database

import (
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

type RefObj interface {
	GetObjIdOrPath() string
	ReadObjId() (string, error)
}

type Ref struct {
	ObjId string
}

func (r *Ref) GetObjIdOrPath() string {
	return r.ObjId
}

func (r *Ref) ReadObjId() (string, error) {
	return r.ObjId, nil
}

type SymRef struct {
	Path string
	Refs *Refs
}

func (sr *SymRef) GetObjIdOrPath() string {
	return sr.Path
}

func (sr *SymRef) ReadObjId() (string, error) {
	return sr.Refs.ReadRef(sr.Path)
}

//IsHead=trueならDetachedHeadということで良いっぽい？
func (sr *SymRef) IsHead() bool {
	return sr.Path == "HEAD"
}

func (sr *SymRef) ShortName() string {
	return sr.Refs.ShortName(sr.Path)
}

func (r *Refs) ShortName(path string) string {
	return filepath.Base(path)
}

var refPrefix = `^ref: (.+)$`

func (r *Refs) ReadObjIdOrSymRef(path string) (RefObj, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	//InitでHEADから読み取るときはまだ内容がないのでその時はRefを返せばいい
	s := strings.TrimSpace(string(b))
	refExp := util.CheckRegExpSubString(refPrefix, s)

	//len(nil)は0となるので refExp != nilチェックはいらない
	if len(refExp) != 0 {
		return &SymRef{
			Path: refExp[0][1],
			Refs: r,
		}, nil
	} else {
		return &Ref{
			ObjId: strings.TrimSpace(string(b)),
		}, nil
	}
}

func (r *Refs) ReadSymRef(path string) (string, error) {
	ref, err := r.ReadObjIdOrSymRef(path)
	if err != nil {
		return "", err
	}

	switch v := ref.(type) {
	case *Ref:
		return v.GetObjIdOrPath(), nil
	case *SymRef:
		return r.ReadSymRef(filepath.Join(r.Path, v.GetObjIdOrPath()))
	default:
		return "", ErrorUnexpectedObjType
	}

}

//最終的にSymRefで返す
func (r *Refs) CurrentRef(source string) (*SymRef, error) {
	ref, err := r.ReadObjIdOrSymRef(filepath.Join(r.Path, source))

	if err != nil {
		return nil, err
	}

	switch v := ref.(type) {
	case *SymRef:
		return r.CurrentRef(v.GetObjIdOrPath())
	case *Ref:
		return &SymRef{
			Path: source,
			Refs: r,
		}, nil
	default:
		return nil, ErrorUnexpectedObjType
	}
}

func (r *Refs) DeleteBranch(path string) (string, error) {
	p := filepath.Join(r.HeadsPath(), path)

	stat, _ := os.Stat(p)

	if stat == nil {
		//存在しなければエラー
		return "", ErrorPathNotExists
	}

	l := lock.NewFileLock(p)
	l.Lock()
	defer l.Unlock()

	objId, err := r.ReadSymRef(p)
	if err != nil {
		return "", err
	}

	err = os.RemoveAll(p)
	if err != nil {
		return "", err
	}
	//refs/heads/features/xxxがあったとして、今xxxを削除してfeaturesが空になったとする
	//そうするとfeaturesを削除したい(headsまでは削除しない)
	err = util.DeleteParentDir(path, r.HeadsPath())
	if err != nil {
		return "", err
	}

	return objId, nil
}

// func (r *Refs) DeleteParentDir(path string) error {
// 	for _, p := range util.ParentDirs(path, true) {
// 		absPath := filepath.Join(r.HeadsPath(), p)
// 		if absPath == r.HeadsPath() {
// 			break
// 		}

// 		files, err := ioutil.ReadDir(absPath)
// 		if err != nil {
// 			return err
// 		}

// 		if len(files) != 0 {
// 			//Dirが空でなければ削除しない
// 			break
// 		}

// 		err = os.Remove(absPath)

// 		if err != nil {
// 			return err
// 		}

// 	}

// 	return nil
// }

func (r *Refs) ListBranches() ([]*SymRef, error) {
	return r.ListRefs(r.HeadsPath())
}

func (r *Refs) ListRefs(headsPath string) ([]*SymRef, error) {
	lists, err := util.FilePathWalkDir(headsPath, []string{".", ".."})
	if err != nil {
		return nil, err
	}

	var temp []*SymRef
	//WalkDirを使うことでnestedDirの下にあるファイルも一気に取れる
	for _, l := range lists {
		//refs/heads/~の相対パスとなる
		relPath, err := filepath.Rel(r.Path, filepath.Join(r.HeadsPath(), l))

		if err != nil {
			return nil, err
		}
		temp = append(temp, &SymRef{
			Path: relPath,
			Refs: r,
		})
	}

	return temp, nil
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
	//CreateHeadPathで存在しない場合を作っているのでここではO_CREATEしなくていい
	//で、OpenFileだけすると上書きじゃなくてAPPENDになるのでos.Createを使う
	//O_TRUNCATEを使うかos.Createを使うか、あんまりOpenFileは使わなくてもいいかな
	//ない場合は作りたいならos.Create使えばいい、用途としてはPermissionまで指定したい時
	f, err := os.Create(path)
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
		f.Write([]byte(fmt.Sprintf("%s\n", objId)))
	})
	if err != nil {
		return err
	}

	return nil

}

func (r *Refs) UpdateHead(objId string) error {
	r.UpdateSymRef(r.HeadPath(), objId)
	return nil
}

func (r *Refs) UpdateSymRef(path, objId string) error {
	l := lock.NewFileLock(path)
	l.Lock()
	defer l.Unlock()

	ref, err := r.ReadObjIdOrSymRef(path)
	if err != nil {
		return err
	}

	symRef, ok := ref.(*SymRef)
	//つまりHEAD->masterを指していたとしたらmasterにobjIdを書き込む
	//DetachedHEADだったらそのままHEADにobjIdを書く
	if !ok {
		//Refの場合
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer func() {
			err := f.Close()
			if err != nil {
				fmt.Println(err)
			}
		}()
		f.Write([]byte(fmt.Sprintf("%s\n", objId)))
	} else {
		//SymRefの場合
		err = r.UpdateSymRef(filepath.Join(r.Path, symRef.GetObjIdOrPath()), objId)
		if err != nil {
			return err
		}
	}

	return nil

}

func (r *Refs) SetHead(revPath, objId string) error {
	path := filepath.Join(r.HeadsPath(), revPath)

	stat, _ := os.Stat(path)

	if stat != nil && !stat.IsDir() {

		relPath, _ := filepath.Rel(r.Path, path) //refs/heads/~以下だけ書きたい
		err := r.UpdateRefFile(r.HeadPath(), fmt.Sprintf("ref: %s", relPath))
		if err != nil {
			return err
		}
	} else {
		err := r.UpdateRefFile(r.HeadPath(), objId)
		if err != nil {
			return err
		}
	}

	return nil
}

var ErrorPathNotExists = errors.New("PathNotExists")

func (r *Refs) ReadRef(name string) (string, error) {
	path, isExists := r.PathForName(name)

	if !isExists {
		return "", ErrorPathNotExists
	}

	return r.ReadSymRef(path)

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
	return r.ReadSymRef(r.HeadPath())
	// if _, err := os.Stat(r.HeadPath()); err != nil {
	// 	return "", nil
	// }

	// f, err := os.Open(r.HeadPath())

	// if err != nil {
	// 	return "", err
	// }

	// s := bufio.NewScanner(f)

	// var str string
	// for s.Scan() {
	// 	str += s.Text()
	// }

	// return string(str), nil
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
