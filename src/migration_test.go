package src

import (
	"bytes"
	"fmt"
	er "mygit/src/errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func PrepareCompareTwoCommit(t *testing.T) string {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "hello.txt", "test\n")
	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	os.Remove(dummyName)
	CreateFiles(t, xxxPath, "dummy2.txt", "test\n")
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2")
	assert.NoError(t, err)

	return tempPath //こうしているのはなぜかcleanUp関数を返そうとすると最後のコミットまでいかない
}

//ForMigrationの方はexists.txtを変更なしのファイルとしてのちのちこれもうまくmigrationできるように作っている
func PrepareCompareTwoCommitForMigartion(t *testing.T) string {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "hello.txt", "test\n")
	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2\n")
	CreateFiles(t, xxxPath, "exists.txt", "test2\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	os.Remove(dummyName)
	CreateFiles(t, xxxPath, "dummy2.txt", "test\n")
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2")
	assert.NoError(t, err)

	return tempPath //こうしているのはなぜかcleanUp関数を返そうとすると最後のコミットまでいかない
}

//現在のMigrationだと二つのコミット間で関係のないexists.txtが消えてしまう
// 	//(一回xxxを削除しているので)なのでコメントアウトしている、治ったらコメント解除

// func Test_MigateWorkspace(t *testing.T) {
// 	path := PrepareCompareTwoCommitForMigartion(t)
// 	t.Cleanup(func() {
// 		os.RemoveAll(path)
// 	})

// 	//現在Commit2の状況からworkSpaceがCommit1の状況に戻っていればいい、つまり
// 	//hello.txt,xxx/dummy2.txt -> hello.txt,xxx/dummy.txtに戻る
// 	//だからtreeDiffではdummy2がdelete,dummyがaddとなる
// 	//またxxx内のmigration間で変化しないxxx/exists.txtは、
// 	//一回xxxがdeleteされても当然きちんと残る

// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)
// 	rootPath := filepath.Join(curDir, "tempDir")
// 	gitPath := filepath.Join(rootPath, ".git")
// 	dbPath := filepath.Join(gitPath, "objects")
// 	repo := GenerateRepository(rootPath, gitPath, dbPath)

// 	head, err := ParseRev("@")
// 	assert.NoError(t, err)
// 	headId, err := ResolveRev(head, repo)
// 	assert.NoError(t, err)
// 	parent, err := ParseRev("@^")
// 	assert.NoError(t, err)
// 	parentId, err := ResolveRev(parent, repo)
// 	assert.NoError(t, err)

// 	err = repo.i.Load()
// 	assert.NoError(t, err)

// 	td := GenerateTreeDiff(repo)
// 	td.CompareObjId(headId, parentId)
// 	m := GenerateMigration(td, repo)
// 	err = m.ApplyChanges()
// 	assert.NoError(t, err)

// 	lists, err := repo.w.ListFiles(rootPath)
// 	assert.NoError(t, err)
// 	//現在のMigrationだと二つのコミット間で関係のないexists.txtが消えてしまう
// 	//(一回xxxを削除しているので)<-17まで見てこれが解決しているか見る
// 	expectedList := []string{"hello.txt", "xxx/dummy.txt", "xxx/exists.txt"}

// 	if diff := cmp.Diff(expectedList, lists); diff != "" {
// 		t.Errorf("diff is: %s\n", diff)
// 	}

// }

func TestUntrackedOverWritten(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(tempPath)

	CreateFiles(t, tempPath, "hello.txt", "test\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	var buf1 bytes.Buffer

	err = StartBranch(tempPath, []string{"from"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"to"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)

	var buf2 bytes.Buffer
	err = StartCheckout(tempPath, []string{"to"}, &buf2)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "dup.txt", "test\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2")
	assert.NoError(t, err)

	var buf3 bytes.Buffer
	err = StartCheckout(tempPath, []string{"from"}, &buf3)
	assert.NoError(t, err)
	CreateFiles(t, tempPath, "dup.txt", "untracked\n")

	// 	error: The following untracked working tree files would be overwritten by checkout:
	//         add.txt
	// Please move or remove them before you switch branches.
	// 状況再現としてはコミットＡ->コミットＢ(to)をつくる(dup.txt
	// コミットＡからコミットＣ(from)をつくる(dup.txt(untrackedとする)

	// コミットＣ->コミットＢにマージ
	err = StartCheckout(tempPath, []string{"to"}, &buf3)
	if diff := cmp.Diff(&er.ConflictOccurError{
		ConflictDetail: fmt.Sprint("error: The following untracked working tree files would be overwritten by checkout:\n\tdup.txt\nPlease move or remove them before you switch branches.\n"),
	}, err); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}

}

//refs/heads以下をcommitで更新できるようになったらtest可能
func TestLocalChangeOverWritten(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(tempPath)

	CreateFiles(t, tempPath, "hello.txt", "test\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	var buf1 bytes.Buffer

	err = StartBranch(tempPath, []string{"from"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"to"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)

	var buf2 bytes.Buffer
	err = StartCheckout(tempPath, []string{"to"}, &buf2)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "dup.txt", "test\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2")
	assert.NoError(t, err)

	var buf3 bytes.Buffer
	err = StartCheckout(tempPath, []string{"from"}, &buf3)
	assert.NoError(t, err)
	CreateFiles(t, tempPath, "dup.txt", "modfied\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	// 	error: Your local changes to the following files would be overwritten by checkout:
	//         add.txt
	// Please commit your changes or stash them before you switch branches.
	// 状況再現としてはコミットＡ->コミットＢをつくる(dum.txt
	// コミットＡからコミットＣをつくる(dum.txt(indexedでコミットＢとは内容を変える(内容が同じならswitchされる))
	// 内容が同じならswitchされるがtest1(コミットCからはdeleteされてしまう)(ただコミットBで同じ内容があるから内容は失われないからセーフということだろう)

	// コミットＣ->コミットＢにマージ
	err = StartCheckout(tempPath, []string{"to"}, &buf3)
	if diff := cmp.Diff(&er.ConflictOccurError{
		ConflictDetail: fmt.Sprint("error: Your local changes to the following files would be overwritten by checkout:\n\tdup.txt\nPlease commit your changes or stash them before you switch branches.\n"),
	}, err); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}
}

//refs/heads以下をcommitで更新できるようになったらtest可能
func TestUntrackedRemoved(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(tempPath)

	CreateFiles(t, tempPath, "hello.txt", "test\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	var buf1 bytes.Buffer

	err = StartBranch(tempPath, []string{"from"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"to"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)

	var buf3 bytes.Buffer
	err = StartCheckout(tempPath, []string{"from"}, &buf3)
	assert.NoError(t, err)
	addName := CreateFiles(t, tempPath, "added.txt", "a\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	os.Remove(addName)
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "added.txt", "a\n")

	// 	removedの条件
	// treeDiffのold(checkout元)にあって(commit済み)、workspaceにもあってIndexにないが条件
	//fromで一回ファイルを作ってcommitまでして、削除してAddでIndexから除外し、
	//また作成する(addしないでworkspaceにとどめる)

	// コミットＣ->コミットＢにマージ
	err = StartCheckout(tempPath, []string{"to"}, &buf3)
	if diff := cmp.Diff(&er.ConflictOccurError{
		ConflictDetail: fmt.Sprint("error: The following untracked working tree files would be removed by checkout:\n\tadded.txt\nPlease move or remove them before you switch branches.\n"),
	}, err); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}
}

//refs/heads以下をcommitで更新できるようになったらtest可能
func TestUpdatingFollowingDirectoriesLose(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	defer os.RemoveAll(tempPath)

	CreateFiles(t, tempPath, "hello.txt", "test\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	var buf1 bytes.Buffer

	err = StartBranch(tempPath, []string{"from"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"to"}, &BranchOption{}, &buf1)
	assert.NoError(t, err)

	var buf2 bytes.Buffer
	err = StartCheckout(tempPath, []string{"to"}, &buf2)
	assert.NoError(t, err)

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, xxxPath, "dup.txt", "test\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2")
	assert.NoError(t, err)

	var buf3 bytes.Buffer
	err = StartCheckout(tempPath, []string{"from"}, &buf3)
	assert.NoError(t, err)

	xxxDupPath := filepath.Join(tempPath, "xxx", "dup.txt")
	err = os.MkdirAll(xxxDupPath, os.ModePerm)
	assert.NoError(t, err)
	CreateFiles(t, xxxDupPath, "a.txt", "untracked\n")

	// treeDiffにDirがないのになんでDirのConflict？と思ったけど、
	// まずDiffPathがlib/app.txtとする
	// それでWorkspaceであたらしくlib/app.txtというDirをつくったとする
	// そうするとisDir条件に引っかかる

	// これはコミットではファイルだったがワークスペースではdirになってさらにその中にuntrackedなものがあるとき起こる
	// (checkoutによってdir -> fileになってしまい、untrackedが消えるから警告ということ)

	// 再現方法
	// checkout先でxxx/dum.txt(ファイル)
	// ccheckout元でxxx/dum.txt/a.txtとする、この時a.txtはuntracked

	// コミットＣ->コミットＢにマージ
	err = StartCheckout(tempPath, []string{"to"}, &buf3)
	if diff := cmp.Diff(&er.ConflictOccurError{
		ConflictDetail: fmt.Sprint("error: Updating the following directories would lose untracked files in them\n\txxx/dup.txt\n\n\n"),
	}, err); diff != "" {
		t.Errorf("diff is: %s\n", diff)
	}
}

func PrepareParentUntracked(t *testing.T) string {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "hello.txt", "test\n")
	CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/xxx/dummy.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("change1"))
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2")
	assert.NoError(t, err)

	return tempPath //こうしているのはなぜかcleanUp関数を返そうとすると最後のコミットまでいかない
}

//本家と違う挙動？、Untrackedをのちにうまく扱えるようになったら修正？
// func Test_DetectParentUntracked(t *testing.T) {
// 	path := PrepareParentUntracked(t)
// 	t.Cleanup(func() {
// 		os.RemoveAll(path)
// 	})

// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)
// 	rootPath := filepath.Join(curDir, "tempDir")
// 	gitPath := filepath.Join(rootPath, ".git")
// 	dbPath := filepath.Join(gitPath, "objects")
// 	repo := GenerateRepository(rootPath, gitPath, dbPath)

// 	head, err := ParseRev("@")
// 	assert.NoError(t, err)
// 	headId, err := ResolveRev(head, repo)
// 	assert.NoError(t, err)
// 	parent, err := ParseRev("@^")
// 	assert.NoError(t, err)
// 	parentId, err := ResolveRev(parent, repo)
// 	assert.NoError(t, err)

// 	err = repo.i.Load()
// 	assert.NoError(t, err)
// 	//WorkSpaceから消す
// 	err = os.RemoveAll(filepath.Join(curDir, "tempDir/xxx/dummy.txt"))
// 	assert.NoError(t, err)
// 	xxxPath := filepath.Join(rootPath, "xxx")
// 	CreateFiles(t, xxxPath, "untracked.txt", "test2\n")

// 	td := GenerateTreeDiff(repo)
// 	td.CompareObjId(headId, parentId)
// 	m := GenerateMigration(td, repo)
// 	err = m.ApplyChanges()
// 	assert.NoError(t, err)

// }
