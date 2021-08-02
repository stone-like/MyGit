package src

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

//一律tempDir直下にテスト用の一時フォルダを作るのではなく、tempDirでランダムなディレクトリ名を使うことによって並列にテストできるようになった
func PrepareMerge(t *testing.T) string {

	// A -> B   master
	//   -> C  test1

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	hello2Path := CreateFiles(t, tempPath, "hello2.txt", "test\n")

	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	f1, err := os.Create(hello2Path)
	assert.NoError(t, err)
	f1.Write([]byte("changed2"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changed"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	return tempPath
}

func Test_Merge(t *testing.T) {
	tempPath := PrepareMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	var buf bytes.Buffer

	//masterにtest1をmerge
	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}
	err := StartMerge(mc, &buf)
	assert.NoError(t, err)

	c1, err := ioutil.ReadFile(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)
	c2, err := ioutil.ReadFile(filepath.Join(tempPath, "hello2.txt"))
	assert.NoError(t, err)

	assert.Equal(t, "changed", string(c1))
	assert.Equal(t, "changed2", string(c2))
}

func PrepareNULLMerge(t *testing.T) string {

	// A -> B -> D master
	//   \    /
	//     C   test1

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	hello2Path := CreateFiles(t, tempPath, "hello2.txt", "test\n")

	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	f1, err := os.Create(hello2Path)
	assert.NoError(t, err)
	f1.Write([]byte("changed2"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changed"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}
	err = StartMerge(mc, &buf)
	assert.NoError(t, err)

	return tempPath
}

func TestNULLMerge(t *testing.T) {
	tempPath := PrepareNULLMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	var buf bytes.Buffer
	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	//masterにtest1をmerge
	err := StartMerge(mc, &buf)
	assert.NoError(t, err)

	assert.Equal(t, AlreadyMergedMessage, buf.String())
}

func PrepareFastForwardMerge(t *testing.T) string {

	// A <- B [master] <= HEAD
	//       \
	//         C <- D    [test1]
	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	hello2Path := CreateFiles(t, tempPath, "hello2.txt", "test\n")

	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commitA", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	f1, err := os.Create(hello2Path)
	assert.NoError(t, err)
	f1.Write([]byte("changed2"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commitB", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changed"))
	f2.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commitC", &buf)
	assert.NoError(t, err)
	//一回Createで開き直さないと追記になる
	f3, err := os.Create(helloPath)
	assert.NoError(t, err)
	f3.Write([]byte("change2"))
	f3.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commitD", &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	return tempPath
}

func TestFastForwardMerge(t *testing.T) {
	tempPath := PrepareFastForwardMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	m, err := GenerateMerge("HEAD", "test1", repo)
	assert.NoError(t, err)

	var buf bytes.Buffer
	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	//masterにtest1をmerge
	err = RunMerge(mc, m, &buf)
	assert.NoError(t, err)

	//headがrightObjIdにupdateされていることを確認
	headContent, err := m.repo.r.ReadHead()
	assert.NoError(t, err)

	assert.Equal(t, m.rightObjId, headContent)

	//mergeされてhello.txtの内容が変わっていることを確認
	bytes, err := ioutil.ReadFile(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "change2", string(bytes))

	str := buf.String()
	expected := fmt.Sprintf("Updating %s..%s\nFast-forward\n", m.repo.d.ShortObjId(m.leftObjId), m.repo.d.ShortObjId(m.rightObjId))

	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func PrepareSamePathConflictContent(t *testing.T) string {

	// A -> B   master   hello.txtをconflictされる(contentで)
	//   -> C  test1

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")
	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	f1, err := os.Create(helloPath)
	assert.NoError(t, err)
	f1.Write([]byte("test1"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("master"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	return tempPath
}

//SamePathConflict(Content)
//テスト項目は
//commitまで行われていない->refs/heads/masterのObjIdが変わっていない
//index,workspaceがそれぞれ変更されていること
func Test_SamePathConflictContent(t *testing.T) {
	tempPath := PrepareSamePathConflictContent(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("@")
	assert.NoError(t, err)
	beforeMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	var buf bytes.Buffer

	//masterにtest1をmerge
	m, err := GenerateMerge("HEAD", "test1", repo)
	assert.NoError(t, err)

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	err = RunMerge(mc, m, &buf)
	assert.NoError(t, err)

	//workspace更新を確認
	c1, err := ioutil.ReadFile(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)
	str := string(c1)
	if diff := cmp.Diff(`<<<<<<< HEAD
master
=======
test1
>>>>>>> test1
`, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

	//index更新を確認
	assert.Equal(t, 3, len(m.repo.i.Entries))

	var num = 1
	for _, k := range m.repo.i.Entries.GetSortedkey() {
		//conflict状態なのでlenがhello.txtのstage1~3で3つあってほしい
		assert.Equal(t, "hello.txt", k.Path)
		assert.Equal(t, num, k.Stage)
		num += 1
	}

	//commitされてないことを確認
	ret, err = ParseRev("@")
	assert.NoError(t, err)
	afterMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	assert.Equal(t, beforeMergedObjId, afterMergedObjId)

}

func PrepareSamePathConflictMod(t *testing.T) string {

	// A -> B   master   hello.txtをconflictされる(mod)
	//   -> C  test1
	//A -> nil
	//B -> 100644
	//C -> 100755

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	//add,commitするために適当にdummyFileを作っている
	CreateFiles(t, tempPath, "dummy.txt", "test\n")
	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "test\n")

	os.Chmod(helloPath, 0755)

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	helloPath = CreateFiles(t, tempPath, "hello.txt", "test\n")

	os.Chmod(helloPath, 0644)

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	return tempPath
}

//SamePathConflict(Content)
//テスト項目は
//commitまで行われていない->refs/heads/masterのObjIdが変わっていない
//index,workspaceがそれぞれ変更されていること
func Test_SamePathConflictMod(t *testing.T) {
	tempPath := PrepareSamePathConflictMod(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("@")
	assert.NoError(t, err)
	beforeMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	var buf bytes.Buffer

	//masterにtest1をmerge
	m, err := GenerateMerge("HEAD", "test1", repo)
	assert.NoError(t, err)

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	err = RunMerge(mc, m, &buf)
	assert.NoError(t, err)

	//workspace更新を確認
	//今回はchmodなのでcontentの変化、objIdの変化はなしなので<<< === みたいのはなし

	//index更新を確認
	assert.Equal(t, 3, len(m.repo.i.Entries))

	for ind, k := range []struct {
		path  string
		stage int
	}{
		{"dummy.txt", 0},
		{"hello.txt", 2},
		{"hello.txt", 3},
	} {
		sortedKeys := m.repo.i.Entries.GetSortedkey()
		//conflict状態で今回はbaseがnilなのでstage2,3、stage 0のdummy.txt
		assert.Equal(t, k.path, sortedKeys[ind].Path)
		assert.Equal(t, k.stage, sortedKeys[ind].Stage)
	}

	//commitされてないことを確認
	ret, err = ParseRev("@")
	assert.NoError(t, err)
	afterMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	assert.Equal(t, beforeMergedObjId, afterMergedObjId)

}

//FileDirをテストしてテスト項目は
//commitまで行われていない->refs/heads/masterのObjIdが変わっていない
//index,workspaceがそれぞれ変更されていること
func PrepareFileDirConflict(t *testing.T) string {

	// A -> B   master   hello.txtをconflictされる(mod)
	//   -> C  test1
	//A -> nil
	//B -> 100644
	//C -> 100755

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	//add,commitするために適当にdummyFileを作っている
	CreateFiles(t, tempPath, "dummy.txt", "test\n")
	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "aaa.txt", "test\n")
	cccPath := filepath.Join(tempPath, "ccc.txt")
	err = os.MkdirAll(cccPath, os.ModePerm)
	assert.NoError(t, err)
	CreateFiles(t, cccPath, "ddd.txt", "test\n")

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	aaaPath := filepath.Join(tempPath, "aaa.txt")
	err = os.MkdirAll(aaaPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, aaaPath, "bbb.txt", "test\n")
	CreateFiles(t, tempPath, "ccc.txt", "test\n")

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	return tempPath
}

//FileDirConflictは最終的にfileの方をaaa.txt~test1とrenameして
//Dirの方をマージするようにするのが目標
//なので上記の例だと
//conflictMergeの結果は
// aaa.txt~test1
// ccc.txt~HEAD
// aaa.txt/bbb.txt
// ccc.txt/ddd.txtとなる
//なのでrccc.txtをrenameし、
//aaa.txtをcleanDiffから除外しrenameしたものを同じく書き込み
//さらにtest1からcleanDiffを使ってccc.txt/ddd.txtを追加
func Test_FileDirConflict(t *testing.T) {
	tempPath := PrepareFileDirConflict(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("@")
	assert.NoError(t, err)
	beforeMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	var buf bytes.Buffer

	//masterにtest1をmerge
	m, err := GenerateMerge("HEAD", "test1", repo)
	assert.NoError(t, err)

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	err = RunMerge(mc, m, &buf)
	assert.NoError(t, err)

	//workspace更新を確認
	//renameされていることを確認
	//aaa.txt~right(test1)
	//ccc.txt~left(HEAD)となっていることを確認
	//aaa.txt/bbb.txt,ccc.txt/ddd.txtがあることを確認

	c1, err := ioutil.ReadFile(filepath.Join(tempPath, "aaa.txt~test1"))
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(c1))
	c2, err := ioutil.ReadFile(filepath.Join(tempPath, "ccc.txt~HEAD"))
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(c2))
	c3, err := ioutil.ReadFile(filepath.Join(tempPath, "aaa.txt/bbb.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(c3))
	c4, err := ioutil.ReadFile(filepath.Join(tempPath, "ccc.txt/ddd.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(c4))

	//index更新を確認 aaa.txt,ccc.txtが書き込まれていることを確認
	//dirの方をコンフリクトとする、なのでaaa.txtはright,ccc.txtはleft
	assert.Equal(t, 5, len(m.repo.i.Entries))

	for ind, k := range []struct {
		path  string
		stage int
	}{
		{"aaa.txt", 3},
		{"aaa.txt/bbb.txt", 0},
		{"ccc.txt", 2},
		{"ccc.txt/ddd.txt", 0},
		{"dummy.txt", 0},
	} {
		sortedKeys := m.repo.i.Entries.GetSortedkey()
		//conflict状態で今回はbaseがnilなのでstage2,3、stage 0のdummy.txt
		assert.Equal(t, k.path, sortedKeys[ind].Path)
		assert.Equal(t, k.stage, sortedKeys[ind].Stage)
	}

	//commitされてないことを確認
	ret, err = ParseRev("@")
	assert.NoError(t, err)
	afterMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	assert.Equal(t, beforeMergedObjId, afterMergedObjId)

}

func PrepareBeforeConflictMerge(t *testing.T) string {

	// A -> B  master
	//   \
	//     C   test1

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
	assert.NoError(t, err)

	helloPath := CreateFiles(t, tempPath, "hello.txt", "initial\n")

	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)

	f1, err := os.Create(helloPath)
	assert.NoError(t, err)
	f1.Write([]byte("test1Changed\n"))
	f1.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3", &buf)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("masterChanged\n"))
	f2.Close()

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6", &buf)
	assert.NoError(t, err)

	return tempPath
}

// testはMergeMsgができているかと、
// merge終了したときにしっかりno mergeConflictがでるか
// と
// それぞれマージのメッセージがしっかり出るか
func TestCreateMergeHeadAndMessage(t *testing.T) {
	tempPath := PrepareBeforeConflictMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	var buf bytes.Buffer

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	err := StartMerge(mc, &buf)
	assert.NoError(t, err)
	gitPath := filepath.Join(tempPath, ".git")

	stat, _ := os.Stat(filepath.Join(gitPath, Merge_HEAD))
	assert.NotNil(t, stat)
	stat, _ = os.Stat(filepath.Join(gitPath, Merge_MSG))
	assert.NotNil(t, stat)

	str := buf.String()

	if diff := cmp.Diff(`Auto-merging hello.txt
CONFLICT (content): Merge conflict in hello.txt
Automatic merge failed: fix conflicts and then commit the result.
`, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func TestAlreadyMergedMessageInProgress(t *testing.T) {
	//すでにマージしてコンフリクト済みのときに、マージとコミットをしたときのメッセージ
	tempPath := PrepareConflictMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	var buf bytes.Buffer

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"}}

	err := StartMerge(mc, &buf)
	assert.NoError(t, err)

	str := buf.String()

	if diff := cmp.Diff(`error: Merging is not possible because youy unmerged files
hint: Fix them up in the work tree, and then use 'mygit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.
`, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func TestCommitWithoutAddingIndex(t *testing.T) {
	//すでにマージしてコンフリクト済みのときに、マージとコミットをしたときのメッセージ
	tempPath := PrepareConflictMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	expect := `error: Commiting is not possible because you have unmerged files/nhint: Fix them up in the work tree, and then use 'mygit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.
`

	for _, d := range []struct {
		title string
		fn    func() string
	}{
		{"mergeCommit --continue", func() string {
			var buf bytes.Buffer

			mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"},
				Option: MergeOption{hasContinue: true},
			}

			err := StartMerge(mc, &buf)
			assert.NoError(t, err)

			return buf.String()
		}},
		{"commit", func() string {
			var buf bytes.Buffer
			err := StartCommit(tempPath, "test", "test@email.com", "test", &buf)
			assert.NoError(t, err)
			return buf.String()
		}},
	} {
		t.Run(d.title, func(t *testing.T) {
			if diff := cmp.Diff(expect, d.fn()); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}
		})
	}

}

func TestMergeContinueWithAddingIndex(t *testing.T) {
	//すでにマージしてコンフリクト済みのときに、マージとコミットをしたときのメッセージ

	tempPath := PrepareConflictMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("@")
	assert.NoError(t, err)
	beforeMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	var buf bytes.Buffer

	helloPath := filepath.Join(tempPath, "hello.txt")

	f1, err := os.Create(helloPath)
	assert.NoError(t, err)
	f1.Write([]byte("conflictFixed\n"))
	defer f1.Close()

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"},
		Option: MergeOption{hasContinue: true},
	}

	err = StartMerge(mc, &buf)
	assert.NoError(t, err)

	stat, _ := os.Stat(filepath.Join(gitPath, Merge_HEAD))
	assert.Nil(t, stat)
	stat, _ = os.Stat(filepath.Join(gitPath, Merge_MSG))
	assert.Nil(t, stat)

	ret, err = ParseRev("@")
	assert.NoError(t, err)
	afterMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	assert.NotEqual(t, beforeMergedObjId, afterMergedObjId)

}

func TestCommitWithAddingIndex(t *testing.T) {
	//すでにマージしてコンフリクト済みのときに、マージとコミットをしたときのメッセージ

	tempPath := PrepareConflictMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("@")
	assert.NoError(t, err)
	beforeMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	var buf bytes.Buffer

	helloPath := filepath.Join(tempPath, "hello.txt")

	f1, err := os.Create(helloPath)
	assert.NoError(t, err)
	f1.Write([]byte("conflictFixed\n"))
	defer f1.Close()

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	err = StartCommit(tempPath, "test", "test@email.com", "test", &buf)
	assert.NoError(t, err)

	stat, _ := os.Stat(filepath.Join(gitPath, Merge_HEAD))
	assert.Nil(t, stat)
	stat, _ = os.Stat(filepath.Join(gitPath, Merge_MSG))
	assert.Nil(t, stat)

	ret, err = ParseRev("@")
	assert.NoError(t, err)
	afterMergedObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)
	assert.NotEqual(t, beforeMergedObjId, afterMergedObjId)

}

func TestNoMergeMessageAfterCommitWithAddingIndex(t *testing.T) {
	//すでにマージしてコンフリクト済みのときに、マージとコミットをしたときのメッセージ

	tempPath := PrepareConflictMerge(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	var buf bytes.Buffer

	helloPath := filepath.Join(tempPath, "hello.txt")

	f1, err := os.Create(helloPath)
	assert.NoError(t, err)
	f1.Write([]byte("conflictFixed\n"))
	defer f1.Close()

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	err = StartCommit(tempPath, "test", "test@email.com", "test", &buf)
	assert.NoError(t, err)

	mc := MergeCommand{RootPath: tempPath, Name: "test", Email: "test@email.com", Message: "merged", Args: []string{"test1"},
		Option: MergeOption{hasContinue: true},
	}

	var newBuf bytes.Buffer

	StartMerge(mc, &newBuf)

	str := newBuf.String()

	if diff := cmp.Diff("There is no merge in progress (Merge_HEAD missng).\n", str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}
