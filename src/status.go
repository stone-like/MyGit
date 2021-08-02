package src

import (
	"errors"
	"io"
	"mygit/src/crypt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"mygit/src/database/lock"
	"mygit/src/database/util"
	"os"
	"path/filepath"
)

//untrackedFileか、untrackFileを含んでいるならtrue-
//indexedされているかどうかのチェックはしなくていいはず
//なぜなら、最初にここに来たときは、untrackedなはずで、もしも、
//a/b/c.txtがindexedされていて、aを検査するときはaはindexedになっているはず、
//なのでここに来るDir(とその中身)はすべてuntrackedのものしかないはず
//それでuntrackedなFileかuntrackedFileをどこかに含んでいるDirなら空DirじゃないのでOK

//ただ本家よりもっと簡単にできそうなので下記のようにした
//Dirなのであれば対象DirのすべてのFileをとって来て、その中のIndexされていないものが一件でもあればOK
func IsTrackableFile(i *data.Index, path string, isDir bool, w *WorkSpace) (bool, error) {
	if !isDir {
		//unTrackなファイルが入っている
		return !i.IsIndexed(path), nil
	} else {
		pathList, err := w.ListFiles(filepath.Join(w.Path, path))

		if err != nil {
			return false, err
		}

		var isAnyUntracked bool

		for _, p := range pathList {
			if !i.IsIndexed(filepath.Join(path, p)) {
				isAnyUntracked = true
			}
		}

		return isAnyUntracked, nil
	}

}

func ScanWorkSpace(w *WorkSpace, path string, i *data.Index, s *Status) error {
	pathList, err := w.ListDir(path)

	if err != nil {
		return err
	}

	for k, stat := range pathList {

		if err != nil {
			return err
		}

		if i.IsIndexed(k) {

			if stat.IsDir() {
				ScanWorkSpace(w, filepath.Join(w.Path, k), i, s)
			} else {
				s.Stats[k] = stat
			}
		} else {
			trackable, err := IsTrackableFile(i, k, stat.IsDir(), w)

			if err != nil {
				return err
			}

			if trackable {
				result := k

				if stat.IsDir() {
					result += "/"
				}

				s.Untracked = append(s.Untracked, result)
			}

		}
	}

	return nil

}

var ErrorObjeToEntryConvError = errors.New("conversion error object to entry")

func DetectWorkSpaceChanges(i *data.Index, s *Status, w *WorkSpace) error {
	for k, v := range i.Entries {
		e, ok := v.(*con.Entry)

		if !ok {
			return ErrorObjeToEntryConvError
		}

		err := s.CheckIndexAgainstWorkSpace(k.Path, e, i, w)

		if err != nil {
			return err
		}

	}

	return nil
}

func (s *Status) RecordChange(path string, set map[string]int, cause int) {

	s.Changed = append(s.Changed, path)
	set[path] = cause
}

func (s *Status) CheckIndexAgainstWorkSpace(path string, e *con.Entry, i *data.Index, w *WorkSpace) error {
	stat, ok := s.Stats[path]

	//WorkSpaceに存在するかどうか Index vs WorkSpace

	if !ok {
		//もしStatsに存在しないならDELETE
		s.RecordChange(path, s.WorkSpaceChanges, WORKSPACE_DELETE)
		return nil
	}

	if !i.StatMatch(e, stat) {
		//Statが一致しないならMODIFIED
		s.RecordChange(path, s.WorkSpaceChanges, WORKSPACE_MODIFIED)
		return nil
	}

	//modeとsizeがmatchしているならtimeがmatchしているか調べる
	//timeがmatchしていなくて、objIdが同じならupdate、objIdが違うならchanged
	//objId同じでTime違う->何らかの理由で時間だけ変わった?
	//objId違う->sizeは上で検査しているのでサイズが同じで内容が違うということになる
	//例 hello -> dummyとか、こうするとobjIdが違う(EntryのobjIdはBlobのobjIdと同じ、つまり実際の書き込まれた内容によって決まり、metadataのTimeとかはobjIdに関係ない)
	if !e.TimeMatch(stat) {
		d, err := w.ReadFile(path)
		if err != nil {
			return err
		}

		b := &con.Blob{
			Content: d,
		}

		content := data.GetStoreHeaderContent(b)

		objId := crypt.HexDigestBySha1(content)

		if e.ObjId == objId {
			i.UpdateEntryStat(e, stat)
		} else {
			//ObjIdが一致しないならContentが変化してModified
			s.RecordChange(path, s.WorkSpaceChanges, WORKSPACE_MODIFIED)
		}
	}

	return nil

}

func StartStatus(w io.Writer, rootPath string, isLong bool) error {

	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")

	repo := GenerateRepository(rootPath, gitPath, dbPath)

	// i := data.GenerateIndex(filepath.Join(gitPath, "index"))

	_, indexNonExist := os.Stat(repo.i.Path)

	l := lock.NewFileLock(repo.i.Path)
	l.Lock()
	defer l.Unlock()

	if indexNonExist == nil {
		//.git/indexがある場合のみLoad、newFileLockで存在しないならindexを作ってしまうのでStatの後にしなければならない
		err := repo.i.Load()
		if err != nil {
			return err
		}
	}

	// repo := GenerateRepository(rootPath, gitPath, dbPath)

	s := GenerateStatus()

	err := s.IntitializeStatus(repo)
	if err != nil {
		return err
	}

	//変更されていたところはindexに反映、具体的にはtimeが違ってobjIdが同じケース
	repo.i.Write(repo.i.Path)
	err = s.WriteStatus(w, isLong)
	if err != nil {
		return err
	}

	return nil
}

func (s *Status) IntitializeStatus(repo *Repository) error {

	err := ScanWorkSpace(repo.w, repo.w.Path, repo.i, s)

	if err != nil {
		return err
	}

	err = s.LoadHead(repo.r, repo.d)
	if err != nil {
		return err
	}
	err = s.CheckIndexEntry(repo.i, repo.w)
	if err != nil {
		return err
	}
	err = s.CollectDeletedHeadFiles(repo.i)
	if err != nil {
		return err
	}

	//最後にpath順にソート
	util.SortStringSlice(s.Changed)
	//mapのindexChangedとworkspaceChangedは使う側でソート

	return nil
}

func (s *Status) CollectDeletedHeadFiles(i *data.Index) error {
	for _, v := range s.HeadTree {
		if !i.IsIndexedFile(v.Path) {
			//CommitにあってIndexにないとき
			//なんでi.IsIndexedを使わないかは、IsIndexedだと、Fileが削除されていてもDirが健在ならはIndexedされてると判定してしまうので
			s.RecordChange(v.Path, s.IndexChanges, INDEX_DELETE)
		}
	}

	return nil
}

func (s *Status) LoadHead(r *data.Refs, d *data.Database) error {
	headOid, err := r.ReadHead()
	if err != nil {
		//headがなかったらstatusは読み取れないと今はしておく、もしかしたらエラーじゃない方がいいのかも
		return nil
	}

	o, err := d.ReadObject(headOid)

	if err != nil {
		return err
	}

	c, ok := o.(*con.CommitFromMem)

	if !ok {
		return data.ErrorUnexpectedObjType
	}

	s.ReadTree(d, c.Tree)

	return nil

}

func (s *Status) ReadTree(d *data.Database, objId string) error {

	o, err := d.ReadObject(objId)
	if err != nil {
		return err
	}

	t, ok := o.(*con.Tree)

	if !ok {
		return data.ErrorUnexpectedObjType
	}

	for _, v := range t.Entries {
		e, ok := v.(*con.Entry)
		if !ok {
			return data.ErrorUnexpectedObjType
		}

		if e.IsTree() {
			s.ReadTree(d, e.ObjId)
		} else {
			s.HeadTree[e.Path] = e
		}
	}

	return nil
}

func (s *Status) CheckIndexAgainstHeadTree(path string, e *con.Entry) error {
	ce, ok := s.HeadTree[e.Path]

	//Commitに存在するか Index vs Commit
	if !ok {
		//HeadTreeには存在しない->Commit内には存在しない
		//なので新しくIndexまで追加されたということ
		s.RecordChange(e.Path, s.IndexChanges, INDEX_ADDED)
	} else {
		//Indexにもあり、commitにもある
		if !(ce.Mode == e.Mode && ce.ObjId == e.ObjId) {
			//両方にあるがModeかOnjIdが違っていたらModeified
			s.RecordChange(e.Path, s.IndexChanges, INDEX_MODIFIED)
		}
	}

	return nil
}

func (s *Status) CheckIndexEntry(i *data.Index, w *WorkSpace) error {
	for k, v := range i.Entries {
		e, ok := v.(*con.Entry)

		if !ok {
			return ErrorObjeToEntryConvError
		}

		if e.GetStage() == 0 {
			err := s.CheckIndexAgainstWorkSpace(k.Path, e, i, w)

			if err != nil {
				return err
			}

			err = s.CheckIndexAgainstHeadTree(k.Path, e)
			if err != nil {
				return err
			}
		} else {
			//conflictしているときはchangedとconflictsに入れる
			s.Changed = append(s.Changed, e.Path) //Changedに入れるのはPorceinのため、longではChangedを使わないはず

			s.Conflicts[e.Path] = append(s.Conflicts[e.Path], e.GetStage())
		}

	}

	return nil
}

func GenerateStatus() *Status {
	return &Status{
		IndexChanges:     make(map[string]int),
		WorkSpaceChanges: make(map[string]int),
		Stats:            make(map[string]con.FileState),
		HeadTree:         make(map[string]*con.Entry),
		Conflicts:        make(map[string][]int),
	}
}

type Status struct {
	Untracked        []string
	Changed          []string
	Conflicts        map[string][]int
	Stats            map[string]con.FileState
	HeadTree         map[string]*con.Entry
	IndexChanges     map[string]int
	WorkSpaceChanges map[string]int
}
