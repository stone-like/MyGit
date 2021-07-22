package src

import (
	"mygit/src/crypt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	"path/filepath"
)

type Inspector struct {
	repo *Repository
}

var (
	INSPECTOR_UNTRACKED = ":untracked"
	INSPECTOR_DELETED   = ":deleted"
	INSPECTOR_MODIFIED  = ":modified"
	INSPECTOR_ADDED     = ":added"
)

//untrackedFileか、untrackFileを含んでいるならtrue-
//indexedされているかどうかのチェックはしなくていいはず
//なぜなら、最初にここに来たときは、untrackedなはずで、もしも、
//a/b/c.txtがindexedされていて、aを検査するときはaはindexedになっているはず、
//なのでここに来るDir(とその中身)はすべてuntrackedのものしかないはず
//それでuntrackedなFileかuntrackedFileをどこかに含んでいるDirなら空DirじゃないのでOK

//ただ本家よりもっと簡単にできそうなので下記のようにした
//Dirなのであれば対象DirのすべてのFileをとって来て、その中のIndexされていないものが一件でもあればOK

//Untrackableの定義はIndexに入っていないこと
//なので、IsTrackableはIndexに入っていなければtrue
func (in *Inspector) IsTrackableFile(path string, isDir bool) (bool, error) {
	if !isDir {
		//unTrackなファイルが入っている
		return !in.repo.i.IsIndexed(path), nil
	} else {
		pathList, err := in.repo.w.ListFiles(filepath.Join(in.repo.w.Path, path))

		if err != nil {
			return false, err
		}

		var isAnyUntracked bool

		for _, p := range pathList {
			if !in.repo.i.IsIndexed(filepath.Join(path, p)) {
				isAnyUntracked = true
			}
		}

		return isAnyUntracked, nil
	}

}

func (in *Inspector) CompareIndextoWorkSpace(entry *con.Entry, stat con.FileState) (string, error) {
	if entry == nil {
		//indexとworkspaceを比較して、indexに存在しない
		return INSPECTOR_UNTRACKED, nil
	}

	if stat == nil {
		//indexとworkspaceを比較して、workSpaceに存在しない
		return INSPECTOR_DELETED, nil
	}

	if !in.repo.i.StatMatch(entry, stat) {
		//両方に存在するがStatがMatchしない(Modeが違うかSizeが違う)(ただ内容が違ってSizeが同じ場合は弾けない)
		return INSPECTOR_MODIFIED, nil
	}

	if entry.TimeMatch(stat) {
		//内容が変わっていれば時間も変更されているはずなのでここで時間が変わっていないことを見るだけで高速で確認できる
		return "", nil
	}

	d, err := in.repo.w.ReadFile(entry.Path)

	if err != nil {
		return "", err
	}

	b := &con.Blob{
		Content: d,
	}

	content := data.GetStoreHeaderContent(b)

	objId := crypt.HexDigestBySha1(content)

	if entry.ObjId != objId {
		//中身が変更されていたら
		return INSPECTOR_MODIFIED, nil
	}

	return "", nil
}

func (in *Inspector) CompareTreeToIndex(treeEntry, indexEntry *con.Entry) string {
	if treeEntry == nil && indexEntry == nil {
		return ""
	}

	if treeEntry == nil {
		//TargetCommitとIndexを比べてIndexのみに存在
		return INSPECTOR_ADDED
	}

	if indexEntry == nil {
		//TargetCommitとIndexを比べてTargetCommitのみに存在
		return INSPECTOR_DELETED
	}

	if treeEntry.Mode != indexEntry.Mode || treeEntry.ObjId != indexEntry.ObjId {
		return INSPECTOR_MODIFIED
	}

	return ""
}
