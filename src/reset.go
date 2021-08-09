package src

import (
	data "mygit/src/database"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	"os"
	"path/filepath"
)

type Reset struct {
	CommitObjId string
	Args        []string
	repo        *Repository
	Option      *ResetOption
	Status      *Status
}

type ResetOption struct {
	hasSoft bool
	hasHard bool
}

//argsはresetするファイル達、commitObjIdは戻すコミットの場所
//ここではargsの最初を見て、revだったらargs[0]をcommit指定として使い、ファイル名としては使わないrevでなかったら戻すcommitはHEAD
func (res *Reset) SelectCommitObjId(args []string) error {
	var target string
	if len(args) == 0 {
		target = "HEAD"
	} else {
		target = args[0]
	}

	rev, err := ParseRev(target)
	if err != nil {
		return err
	}

	objId, err := ResolveRev(rev, res.repo)
	if err == nil {
		res.CommitObjId = objId
		//argsの先頭をCommitObjIdの対象として使ったのでpop
		res.Args = res.Args[1:]

		return nil

	}

	//revにそぐわない、普通のfilePathだったら
	objId, err = res.repo.r.ReadHead()
	if err != nil {
		return err
	}
	res.CommitObjId = objId

	return nil

}

func (res *Reset) ResetPath(path string) error {
	list, err := res.repo.d.LoadTreeListWithPath(res.CommitObjId, path)
	if err != nil {
		return err
	}

	//指定したpathをindexから削除
	if path != "" {
		res.repo.i.Remove(path)
	}

	//戻したいCommitの情報をindexに追加
	for path, e := range list {
		res.repo.i.AddFromDB(path, e)
	}

	return nil

}

func (res *Reset) HandleMixed() error {
	//argsが0は何を意味するかというと、特定のファイルではなく、すべてを指定したコミットまで戻すということ
	//まずindexを全部削除してからResetPashで対象Commnitのindexをadd,元のindexを削除しないとindexが混ざりあってしまう
	if len(res.Args) == 0 {
		res.repo.i = res.repo.i.Reset()
		err := res.ResetPath("")
		if err != nil {
			return err
		}
		return nil
	}

	for _, arg := range res.Args {
		err := res.ResetPath(arg)
		if err != nil {
			return err
		}
	}

	return nil
}

func HardResetPath(path string, s *Status, repo *Repository) error {
	repo.i.Remove(path)
	err := repo.w.Remove(path)
	if err != nil {
		return err
	}
	//ここまで削除

	//ここからIndexとworkSpaceを指定したコミット(s.HeadTree)に合わせる
	e, ok := s.HeadTree[path]
	if !ok {
		return nil
	}
	o, err := repo.d.ReadObject(e.ObjId)
	if err != nil {
		return err
	}
	b, blobOk := o.(*con.Blob)
	if !blobOk {
		return ErrorObjeToEntryConvError
	}

	err = repo.w.WriteFileWithMode(path, b.Content, e.Mode)
	if err != nil {
		return err
	}

	stat, err := repo.w.StatFile(path)
	if err != nil {
		return err
	}

	err = repo.i.Add(path, e.ObjId, stat, data.CreateIndex)
	if err != nil {
		return err
	}

	return nil
}

//HandleHardは他コマンドでも使うのでresとは分けてある

//Gitでは--hardでファイル指定はできない、ファイル指定は--hardとかをつけてはできないので、--mixedのみということになる
//なのでworkspaceにおいて特定のファイルを特定のコミットまで戻すときは
//git checkout HEAD -- test_file.txt
//のようにする
func HanldeHard(commitObjId string, repo *Repository) error {
	s := GenerateStatus()
	//指定したcommitObjIdでstatusをみる
	err := s.IntitializeStatusWithObjId(commitObjId, repo)
	if err != nil {
		return err
	}

	for _, path := range s.Changed {
		err := HardResetPath(path, s, repo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (res *Reset) ResetFiles() error {
	if res.Option.hasSoft {
		//softだったらHEADを指定したcommitまで戻すだけ、indexもworkspaceも戻さない
		return nil
	}

	if res.Option.hasHard {
		return HanldeHard(res.CommitObjId, res.repo)
	}

	return res.HandleMixed()

}

func RunReset(res *Reset) error {
	err := res.SelectCommitObjId(res.Args)
	if err != nil {
		return err
	}

	_, indexNonExist := os.Stat(res.repo.i.Path)

	l := lock.NewFileLock(res.repo.i.Path)
	l.Lock()
	defer l.Unlock()

	if indexNonExist == nil {
		//.git/indexがある場合のみLoad、newFileLockで存在しないならindexを作ってしまうのでStatの後にしなければならない
		err := res.repo.i.Load()
		if err != nil {
			return err
		}
	}

	//resetFilesでindexの調整
	err = res.ResetFiles()
	if err != nil {
		return err
	}

	err = res.repo.i.Write(res.repo.i.Path)
	if err != nil {
		return err
	}

	//ファイル単位でないすべてresetのときはHaedの位置を指定したCommitまで
	//ファイル指定したらそのファイルだけ戻す
	if len(res.Args) == 0 {
		headObjId, err := res.repo.r.UpdateHead(res.CommitObjId)
		if err != nil {
			return err
		}

		err = res.repo.r.UpdateRef(data.ORIG_HEAD, headObjId)
		if err != nil {
			return err
		}
	}

	return nil
}

func StartReset(rootPath string, args []string, option *ResetOption) error {
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)

	res := &Reset{Args: args, repo: repo, Option: option}

	return RunReset(res)
}
