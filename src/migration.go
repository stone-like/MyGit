package src

import (
	"fmt"
	data "mygit/src/database"
	con "mygit/src/database/content"
	er "mygit/src/errors"
	"mygit/util"
	"os"
	"path/filepath"
)

var (
	MIGRATION_DELETE                = ":delete"
	MIGRATION_CREATE                = ":created"
	MIGRATION_UPDATE                = ":update"
	MIGRATION_STALE_FILE            = ":stale_file"
	MIGRATION_STALE_DIR             = ":stale_directory"
	MIGRATION_UNTRACKED_OVERWRITTEN = ":untracked_overwritten"
	MIGRATION_UNTRACKED_REMOVED     = ":untracked_removed"
)

type Migration struct {
	TrDiff    *TreeDiff
	repo      *Repository
	Mkdirs    map[string]struct{}
	Rmdirs    map[string]struct{}
	Changes   map[string]map[string]*con.Entry
	Conflicts map[string]map[string]struct{}
	Inspector *Inspector
}

func GenerateMigration(t *TreeDiff, repo *Repository) *Migration {
	return &Migration{
		TrDiff:  t,
		repo:    repo,
		Changes: map[string]map[string]*con.Entry{MIGRATION_CREATE: {}, MIGRATION_UPDATE: {}, MIGRATION_DELETE: {}},
		Mkdirs:  make(map[string]struct{}),
		Rmdirs:  make(map[string]struct{}),
		Conflicts: map[string]map[string]struct{}{
			MIGRATION_STALE_FILE:            {},
			MIGRATION_STALE_DIR:             {},
			MIGRATION_UNTRACKED_OVERWRITTEN: {},
			MIGRATION_UNTRACKED_REMOVED:     {},
		},
		Inspector: &Inspector{
			repo: repo,
		},
	}
}

func (m *Migration) ApplyChanges() error {
	err := m.PlanChanges()
	if err != nil {
		return err
	}
	m.UpdateWorkSpace()
	err = m.UpdateIndex()

	if err != nil {
		return err
	}

	return nil
}

func (m *Migration) UpdateIndex() error {
	//indexとworkspaceがあっているかを検査すればいい
	for k, _ := range m.Changes[MIGRATION_DELETE] {
		//indexからdelete
		m.repo.i.Remove(k)
	}

	for _, s := range []string{MIGRATION_CREATE, MIGRATION_UPDATE} {
		for path, e := range m.Changes[s] {
			stat, err := m.repo.w.StatFile(path)
			if err != nil {
				return err
			}

			m.repo.i.Add(path, e.ObjId, stat, data.CreateIndex)
		}
	}

	return nil

}

func (m *Migration) UpdateWorkSpace() {
	m.repo.w.ApplyMigration(m)
}

func (m *Migration) PlanChanges() error {
	//今回はerrorが出たら即返すわけには行かない(すべてのconflictを見たい)ので構造体にerrorsを持たせる(scalaのvalidateNelみたいなやつが欲しい)
	//Conflictではないerrorは即返す
	for k, v := range m.TrDiff.Changes {
		err := m.CheckForConflict(k, v[0], v[1])
		if err != nil {
			return err
		}
		m.RecordChange(k, v[0], v[1])
	}

	return m.CollectConflictError()
}

var (
	ErrorMessages = map[string][]string{
		MIGRATION_STALE_FILE: {
			"Your local changes to the following files would be overwritten by checkout:",
			"Please commit your changes or stash them before you switch branches.",
		},
		MIGRATION_STALE_DIR: {
			"Updating the following directories would lose untracked files in them",
			"\n",
		},
		MIGRATION_UNTRACKED_OVERWRITTEN: {
			"The following untracked working tree files would be overwritten by checkout:",
			"Please move or remove them before you switch branches.",
		},
		MIGRATION_UNTRACKED_REMOVED: {
			"The following untracked working tree files would be removed by checkout:",
			"Please move or remove them before you switch branches.",
		},
	}
)

func (m *Migration) CollectConflictError() error {

	var isConflictOccured bool
	var totalErrorMessage string
	for errorType, m := range m.Conflicts {
		//errorTypeごとに
		//header
		//conflictedPaths
		//footer
		//の組を作る
		//そのtypeごとの奴をerror: で表示する

		if len(m) == 0 {
			continue
		}

		var conflictedPaths string
		for p := range m {
			isConflictOccured = true
			conflictedPaths += fmt.Sprintf("\t%s\n", p)
		}

		errorHeaderAndFooter := ErrorMessages[errorType]

		messagePerErrorType :=
			"error: " +
				fmt.Sprintf("%s\n", errorHeaderAndFooter[0]) +
				conflictedPaths +
				fmt.Sprintf("%s\n", errorHeaderAndFooter[1])

		totalErrorMessage += messagePerErrorType

	}

	if isConflictOccured {
		return &er.ConflictOccurError{
			ConflictDetail: totalErrorMessage,
		}
	}

	return nil
}

func (m *Migration) CheckForConflict(path string, oldEntry, newEntry *con.Entry) error {
	//ここindexEntryがnilの場合があってもいいんじゃないか<-indexがない場合もあるのでそれも考慮する
	//indexに存在しない場合indexEntryはnil
	indexEntry, _ := m.repo.i.EntryForPath(path)

	//indexとold,newで両方とも変化がある場合
	if m.IndexDifferFromTrees(indexEntry, oldEntry, newEntry) {
		m.Conflicts[MIGRATION_STALE_FILE][path] = struct{}{}
		return nil
	}

	//ここからはindexとold,newでどちらか片方は少なくとも変化がない場合
	stat, nonExist := m.repo.w.StatFile(path)
	errorType := m.GetErrorType(stat, indexEntry, newEntry)

	if nonExist != nil {
		//workSpaceに存在しない
		//parentにuntrackableなものがないか探す
		//lib/app.goがあるとして、これがdeleteされているとすると、migrationのときlibまでdeleteするので親のuntrackedなやつまでdeleteされてしまうのでチェック

		//ここの条件としては、
		//targetPathがtreeDiffにあってかつ、
		//workSpaceから消えていて、
		//targetPathの親にuntrackedFileが含まれている<-なぜ困るかがよくわからない、普通にuntrackedを残しておけばいい...とおもうがまだuntrackedやtreediffに関係ない普通のファイルを残すことが現段階のMyGitではできないための策なのかもしれない、うまくそこら辺を扱えるようになったらここはいらなさそう
		untrackableParent, err := m.UntrackedParent(path)

		if err != nil {
			return err
		}

		if untrackableParent != "" {
			//indexEntryがなければそれを、nilならworkSpaceの存在しないPathのuntrackedな親を使う
			var targetPath string

			if indexEntry != nil {
				targetPath = indexEntry.Path
			} else {
				targetPath = untrackableParent
			}

			m.Conflicts[errorType][targetPath] = struct{}{}
		}
	} else if nonExist == nil && !stat.IsDir() {
		//workspaceに存在してFileの時
		//IndexとWorkSpaceに違いがあるならUntracked、もしくはUnStaged(UnstagedはworkSpaceにもIndexにもあるが修正されている、UntrackedはIndexにあってWorkspaceにない)
		//Indexと違わないならここに来る時はIndexとOld,Newどちらか片方とは違わないので違いなしということになる

		//変更予定のファイルがWorkSpaceにもあってIndexと違うなら結局コンフリクトということ

		changed, err := m.Inspector.CompareIndextoWorkSpace(indexEntry, stat)
		if err != nil {
			return err
		}

		if changed != "" {
			m.Conflicts[errorType][path] = struct{}{}
		}

	} else if nonExist == nil && stat.IsDir() {
		//workspaceに存在してDirの時
		hasUntrackable, err := m.Inspector.IsTrackableFile(path, true)
		if err != nil {
			return err
		}

		if hasUntrackable {
			m.Conflicts[errorType][path] = struct{}{}
		}

	}
	return nil

}

//""でerrorがnillならuntrackedのやつは親に存在しない
func (m *Migration) UntrackedParent(path string) (string, error) {
	//workspaceに存在しないファイルの親を調べる(親にuntrackedが残っていたらアウトなので)
	for _, d := range util.ParentDirs(path, false) {
		//treediffに存在するPathで、worksapaceに存在せず、その親もworkspaceに存在しない場合があるので、
		//まずとってきた親候補が存在するかチェックする
		stat, _ := os.Stat(filepath.Join(m.repo.w.Path, d))

		if stat == nil {
			//親も存在しないなら
			continue
		}

		//Dirの想定だがFileが来ることもある(例えばconflictの結果、HEADではccc.txtが存在し、cleanDiffでccc.txt/ddd.txtをチェックするときにdirとしてccc.txtをチェックするとき、
		// この時cleandiff上ではccc.txtはDirだが、HEADのworkSpaceではccc.txtはFile
		if !stat.IsDir() {
			continue
		}

		lists, err := m.repo.w.ListDir(filepath.Join(m.repo.w.Path, d))
		if err != nil {
			return "", err
		}
		for p, stat := range lists {

			if stat.IsDir() {
				continue
			}

			untracked, err := m.Inspector.IsTrackableFile(p, false)

			if err != nil {
				return "", err
			}

			if untracked {
				return path, nil
			}
		}

	}

	return "", nil
}

func (m *Migration) GetErrorType(stat con.FileState, indexEntry, newEntry *con.Entry) string {
	if indexEntry != nil {
		return MIGRATION_STALE_FILE
	} else if stat != nil && stat.IsDir() {
		return MIGRATION_STALE_DIR
	} else if newEntry != nil {
		return MIGRATION_UNTRACKED_OVERWRITTEN
	} else {
		return MIGRATION_UNTRACKED_REMOVED
	}
}

func (m *Migration) IndexDifferFromTrees(indexEntry, oldEntry, newEntry *con.Entry) bool {
	//indexと比べてnewもoldも変化がなければok、下記では変化がない場合""なので、
	//IndexDifferFromTressとしてはoldとnewどちらとも違えばtrueで、変化あり
	if m.Inspector.CompareTreeToIndex(oldEntry, indexEntry) != "" && m.Inspector.CompareTreeToIndex(newEntry, indexEntry) != "" {
		return true
	} else {
		return false
	}
}

func (m *Migration) RecordChange(path string, oldItem, newItem *con.Entry) {

	var action string
	if oldItem == nil {
		//ここではpathの親も一気に入れる(既存のやつとは被らないようにする)
		// /some/dummyだったら、/,/some,/some/dumnmyが追加
		//あくまで候補としてpathの親をdeleteionに追加するだけ

		for _, p := range util.Descend(path) {
			if p == path {
				//自分は除く、pathの親のみ入れる
				continue
			}
			m.Mkdirs[p] = struct{}{}
		}
		action = MIGRATION_CREATE
	} else if newItem == nil {
		for _, p := range util.Descend(path) {
			if p == path {
				//自分は除く、pathの親のみ入れる
				continue
			}
			m.Rmdirs[p] = struct{}{}
		}
		action = MIGRATION_DELETE
	} else {
		//両方あるとき
		for _, p := range util.Descend(path) {
			if p == path {
				//自分は除く、pathの親のみ入れる
				continue
			}
			m.Mkdirs[p] = struct{}{}
		}
		action = MIGRATION_UPDATE
	}

	m.Changes[action][path] = newItem
}

func (m *Migration) BlobContent(objId string) (string, error) {
	//treeDiffからとってきたやつは全部blobの想定なので
	o, err := m.repo.d.ReadObject(objId)

	if err != nil {
		return "", err
	}

	b, ok := o.(*con.Blob)

	if !ok {
		return "", ErrorObjeToEntryConvError
	}

	return b.Content, nil

}
