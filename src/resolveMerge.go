package src

import (
	"fmt"
	con "mygit/src/database/content"
	"mygit/src/database/util"
	"reflect"
)

type ResolveMerge struct {
	leftDiff  *TreeDiff
	rightDiff *TreeDiff
	cleanDiff map[string][]*con.Entry
	conflicts map[string][]*con.Entry
	//untrackedはfileDirConflictの時用、このconflictした結果はindexではなくworkspaceに反映,fileはrenameされる
	untracked map[string]*con.Entry
	m         *Merge
}

//整理しておかないといけないのは、EntryはあくまでindexにObjId等を保存する媒体なだけで、
//GitObjectはBlob,Tree,Commit
//なのでIndexをロード->EntryからObjIdを取り出してBlobを取得したりすることもある
//interface ObjectでBlobもEntryも同様に扱えてしまえているのがよくないかも、Entryは切り離した方がよさそう
func GenerateResolveMerge(m *Merge) *ResolveMerge {
	return &ResolveMerge{
		cleanDiff: make(map[string][]*con.Entry),
		conflicts: make(map[string][]*con.Entry),
		untracked: make(map[string]*con.Entry),
		m:         m,
	}
}

func (rm *ResolveMerge) Resolve() error {
	err := rm.PrepareTreeDiff()
	if err != nil {
		return err
	}

	// base->mergeの差分をtargetに適用するのが3wayMergeの本質
	mig := GenerateMigration(&TreeDiff{
		Changes: rm.cleanDiff,
		repo:    rm.m.repo,
	}, rm.m.repo)
	err = mig.ApplyChanges()
	if err != nil {
		return err
	}

	err = rm.AddConflictsToIndex()
	if err != nil {
		return err
	}

	err = rm.WriteUntrackedFiles()
	if err != nil {
		return err
	}

	return nil
}

func (rm *ResolveMerge) WriteUntrackedFiles() error {
	for path, e := range rm.untracked {
		o, err := rm.m.repo.d.ReadObject(e.ObjId)
		if err != nil {
			return err
		}

		blob, ok := o.(*con.Blob)

		if !ok {
			return ErrorObjeToEntryConvError
		}

		rm.m.repo.w.WriteFile(path, blob.Content)

	}

	return nil
}

//生成したConflictをindexに書き込み
func (rm *ResolveMerge) AddConflictsToIndex() error {
	for path, entries := range rm.conflicts {
		err := rm.m.repo.i.AddConflictSet(path, entries)
		if err != nil {
			return err
		}
	}

	return nil
}

//MergeはResoveMergeの情報を持たせないようにして、Conflict関連をResoveMerge、最初のbaseObjId生成関連、leftObjId,rightObjIdの情報をMergeに集める
func (rm *ResolveMerge) PrepareTreeDiff() error {

	leftDiff, err := TreeDiffGenerateAndCompareCommit(rm.m.baseObjId, rm.m.leftObjId, rm.m.repo)
	if err != nil {
		return err
	}
	rightDiff, err := TreeDiffGenerateAndCompareCommit(rm.m.baseObjId, rm.m.rightObjId, rm.m.repo)
	if err != nil {
		return err
	}

	rm.leftDiff = leftDiff
	rm.rightDiff = rightDiff

	for path, entries := range rightDiff.Changes {

		rm.SamePathConflict(path, entries[0], entries[1])

		//変更後のdiff[1]がなかったらconflictしようがないので
		if entries[1] != nil {
			//diff適用後が存在するならfileとDirのconflictを調べる
			//fileDirConflictは、
			//left -> f.txt
			//right -> f.txt/g.txtみたいなやつ
			//samePathConflictではこれは調べられない
			rm.FileDirConflict(path, rm.m.leftName, leftDiff)
		}
	}
	//わざわざright,.leftをそれぞれrangeで回してfileDirConflictチェックをする理由は、それぞれの親起点でしかチェックできないから
	//例えば、
	//rightにaaa.txt/bbb.txt , ccc.txt
	//leftに aaa.txt         ,ccc.txt/ddd.txt
	//があったとする
	//この時rightからrangeを回しただけではaaa.txtしかチェックできない
	//leftも回せばccc.txtもチェックできる

	for path, entries := range leftDiff.Changes {
		if entries[1] != nil {
			rm.FileDirConflict(path, rm.m.rightName, rightDiff)
		}
	}

	return nil

}

func (rm *ResolveMerge) FileDirConflict(path, name string, diff *TreeDiff) {
	for _, p := range util.ParentDirs(path, true) {
		entries, ok := diff.Changes[p]
		if !ok {
			continue
		}

		//変更後のdiff[1]がなかったらconflictしようがないので
		if entries[1] == nil {
			continue
		}

		_, exists := rm.conflicts[p]

		if !exists {
			rm.conflicts[p] = make([]*con.Entry, 0)
		}

		switch name {
		case rm.m.leftName:
			{
				rm.conflicts[p] = append(rm.conflicts[p], []*con.Entry{entries[0], entries[1], nil}...)
			}
		case rm.m.rightName:
			{
				rm.conflicts[p] = append(rm.conflicts[p], []*con.Entry{entries[0], nil, entries[1]}...)
			}
		}

		//cleanDiffもworkSpaceに適用される対象だが、renameして適用したいのでcleanDiffからは削除
		delete(rm.cleanDiff, p)

		renamed := fmt.Sprintf("%s~%s", p, name)

		rm.untracked[renamed] = entries[1]

	}
}

//作成したcleanDiffを元にmigration、conflictsをもとにindexにcomnflict情報を書き込み
func (rm *ResolveMerge) SamePathConflict(path string, baseEntry, rightEntry *con.Entry) {

	leftEntries, ok := rm.leftDiff.Changes[path]

	if !ok {
		//rightDiffにあってleftDiffにないということはconflictじゃない
		_, cleanDiffOk := rm.cleanDiff[path]

		if !cleanDiffOk {
			rm.cleanDiff[path] = make([]*con.Entry, 0)
		}

		rm.cleanDiff[path] = append(rm.cleanDiff[path], []*con.Entry{baseEntry, rightEntry}...)

		return
	}

	leftDiffEntry := leftEntries[1] //diffのchangesのEntriesの[0]が変更前、[1]がdiff後

	//leftとrightの変更後を比べて同じだったらなにもしない
	//ポインタ同士の比較だとメモリアドレスが同じでなければ同じでないので、
	//ポインタの場合deepEqualを使う
	//両方nilの場合はここで弾ける
	if reflect.DeepEqual(leftDiffEntry, rightEntry) {
		return
	}

	objId, objIdOk := rm.MergeBlobs(
		baseEntry.GetObjIdForNormalAndNilEntry(),
		leftDiffEntry.GetObjIdForNormalAndNilEntry(),
		rightEntry.GetObjIdForNormalAndNilEntry())

	mode, modeOk := rm.MergeModes(
		baseEntry.GeModeForNormalAndNilEntry(),
		leftDiffEntry.GeModeForNormalAndNilEntry(),
		rightEntry.GeModeForNormalAndNilEntry())

	//conflictでもcleanDiffには入れるcleanDiffの役割は適切にconflict後の状況を現在のbranchから作り出せるようなdiffを保存すること
	_, exists := rm.cleanDiff[path]
	if !exists {
		rm.cleanDiff[path] = make([]*con.Entry, 0)
	}

	e := &con.Entry{ObjId: objId, Mode: mode}
	rm.cleanDiff[path] = append(rm.cleanDiff[path], []*con.Entry{leftDiffEntry, e}...)

	//objIdかmodeかどちらかOKでなかった時点でconflict
	if !objIdOk || !modeOk {
		_, exists := rm.conflicts[path]

		if !exists {
			rm.conflicts[path] = make([]*con.Entry, 0)
		}

		rm.conflicts[path] = append(rm.conflicts[path], []*con.Entry{baseEntry, leftDiffEntry, rightEntry}...)
	}

}

//ここに送られてくるleft,rightはtreeDiffをとったのちの話なので、存在していればmodified,nilならdeletedということになる
func (rm *ResolveMerge) Merge3ObjId(baseObjId, leftObjId, rightObjId string) (string, bool) {

	//leftとright両方nilの場合は事前に弾けている
	if leftObjId == "" {
		return rightObjId, false //<- left deleted,right modifiedのconflict
	}

	if rightObjId == "" {
		return leftObjId, false //<- left modified,right deletedのconflict
	}

	if leftObjId == baseObjId || leftObjId == rightObjId { // base -> rightとmergeすればいいだけなのでconflictなし
		return rightObjId, true
	} else if rightObjId == baseObjId { //そのままleftを使えばいいだけ
		return leftObjId, true
	} else {
		return "", false
	}
}

//modeは100644,100755,040000しかないはずなので0なら存在しない
func (rm *ResolveMerge) Merge3Mode(baseMode, leftMode, rightMode int) (int, bool) {

	if leftMode == 0 {
		return rightMode, false //<- left deleted,right modifiedのconflict
	}

	if rightMode == 0 {
		return leftMode, false //<- left modified,right deletedのconflict
	}

	if leftMode == baseMode || leftMode == rightMode { // base -> rightとmergeすればいいだけなのでconflictなし
		return rightMode, true
	} else if rightMode == baseMode { //そのままleftを使えばいいだけ
		return leftMode, true
	} else {
		return 0, false
	}
}

func (rm *ResolveMerge) MergeModes(baseMode, leftMode, rightMode int) (int, bool) {
	ret, canMerge := rm.Merge3Mode(baseMode, leftMode, rightMode)

	if ret > 0 {
		return ret, canMerge
	} else {
		return leftMode, false
	}
}

//merge3のresultが""ということは、deleted modifiedのconflictではない
//delete,deleteの組は問題ないのでsamePathConflictのleft == rightの部分でreturnしている]
//ということはここに来るのはmodifed,modifiedのconflictなので
//gitのようにv
// <<<<<<<<<<
//  .....
// ======
//  .....
//  >>>>>>>>>>
//をあたらしいblobに書き込むことになる
func (rm *ResolveMerge) MergeBlobs(baseObjId, leftObjId, rightObjId string) (string, bool) {
	retObjId, canMerge := rm.Merge3ObjId(baseObjId, leftObjId, rightObjId)
	if retObjId != "" {
		return retObjId, canMerge
	}

	//modifed,modifiedのconflict
	content, err := rm.MergedData(leftObjId, rightObjId)

	if err != nil {
		return "", false
	}
	blob := &con.Blob{
		Content: content,
	}

	rm.m.repo.d.Store(blob)
	return blob.ObjId, false

}

func (rm *ResolveMerge) MergedData(leftObjId, rightObjId string) (string, error) {
	leftObj, err := rm.m.repo.d.ReadObject(leftObjId)
	if err != nil {
		return "", err
	}
	leftBlob, ok := leftObj.(*con.Blob)
	if !ok {
		return "", ErrorObjeToEntryConvError
	}

	rightObj, err := rm.m.repo.d.ReadObject(rightObjId)
	if err != nil {
		return "", err
	}
	rightBlob, ok := rightObj.(*con.Blob)
	if !ok {
		return "", ErrorObjeToEntryConvError
	}

	str := fmt.Sprintf("<<<<<<< %s\n%s\n=======\n%s\n>>>>>>> %s\n", rm.m.leftName, leftBlob.Content, rightBlob.Content, rm.m.rightName)

	return str, nil

}
