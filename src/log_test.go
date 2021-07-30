package src

import (
	"bytes"
	"fmt"
	con "mygit/src/database/content"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func PrepareMultipleBranch(t *testing.T) func() {

	//A->B -> D master
	//     -> C test1のbranchをつくる
	//時間的にはA(commit1)->B(commit2)->C(commit3)->D(commit4)の順
	//Writeされるのは時間が深い順なので、
	//Commit4 -> 3 -> 2 -> 1
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "hello.txt", "test\n")

	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	CreateFiles(t, tempPath, "hello2.txt", "test\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit2")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)
	CreateFiles(t, tempPath, "hello3.txt", "test\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)
	CreateFiles(t, tempPath, "hello4.txt", "test\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit4")
	assert.NoError(t, err)

	return func() {
		os.RemoveAll(tempPath)
	}
}

//UnixTimeが一秒単位でしか認識できないので、StartLogのテストは仕方なく一秒以上時間をSleepで開けるとして、
//PriorityQueueのテストはこっちで時間を作る形で作って書いた方がよさそう
func Test_LogMultiple(t *testing.T) {
	fn := PrepareMultipleBranch(t)
	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartLog(tempPath, []string{"master", "test1"}, &LogOption{
		Format: "oneline",
	}, buf)
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("master")
	assert.NoError(t, err)
	commit4ObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	ret, err = ParseRev("test1")
	assert.NoError(t, err)
	commit3ObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	ret, err = ParseRev("master^")
	assert.NoError(t, err)
	commit2ObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	ret, err = ParseRev("master^^")
	assert.NoError(t, err)
	commit1ObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	if diff := cmp.Diff(fmt.Sprintf("%s commit4\n%s commit3\n%s commit2\n%s commit1\n", commit4ObjId, commit3ObjId, commit2ObjId, commit1ObjId), buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func Test_LogOne(t *testing.T) {
	fn := PrepareMultipleBranch(t)
	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartLog(tempPath, []string{"master"}, &LogOption{
		Format: "oneline",
	}, buf)
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("master")
	assert.NoError(t, err)
	commit4ObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	ret, err = ParseRev("master^")
	assert.NoError(t, err)
	commit2ObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	ret, err = ParseRev("master^^")
	assert.NoError(t, err)
	commit1ObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	if diff := cmp.Diff(fmt.Sprintf("%s commit4\n%s commit2\n%s commit1\n", commit4ObjId, commit2ObjId, commit1ObjId), buf.String()); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func PrepareUninteresting(t *testing.T) func() {

	// A -> B -> C  -> D master
	//        -> E -> F -> G test1でC,Dのみにできるか？
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "hello.txt", "test\n")

	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	CreateFiles(t, tempPath, "hello2.txt", "test\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit2")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, &buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test1"}, &buf)
	assert.NoError(t, err)
	CreateFiles(t, tempPath, "hello3.txt", "test\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	CreateFiles(t, tempPath, "hello4.txt", "test\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit4")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	CreateFiles(t, tempPath, "hello5.txt", "test\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit5")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)
	CreateFiles(t, tempPath, "hello6.txt", "test\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6")
	assert.NoError(t, err)
	CreateFiles(t, tempPath, "hello7.txt", "test\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit7")
	assert.NoError(t, err)

	return func() {
		os.RemoveAll(tempPath)
	}
}

func Test_LogUninterested(t *testing.T) {

	for _, d := range []struct {
		title        string
		targetBranch []string
		expectCommit string
	}{
		{
			title:        "uninterest^",
			targetBranch: []string{"^test1", "master"},
			expectCommit: "commit4",
		},
		{
			title:        "uninterest..",
			targetBranch: []string{"test1..master"},
			expectCommit: "commit4",
		},
	} {
		t.Run(d.title, func(t *testing.T) {
			fn := PrepareUninteresting(t)
			t.Cleanup(fn)

			curDir, err := os.Getwd()
			assert.NoError(t, err)
			tempPath := filepath.Join(curDir, "tempDir")
			buf := new(bytes.Buffer)
			err = StartLog(tempPath, d.targetBranch, &LogOption{
				Format: "oneline",
			}, buf)
			assert.NoError(t, err)

			gitPath := filepath.Join(tempPath, ".git")
			dbPath := filepath.Join(gitPath, "objects")
			repo := GenerateRepository(tempPath, gitPath, dbPath)

			ret, err := ParseRev("master")
			assert.NoError(t, err)
			commit7ObjId, err := ResolveRev(ret, repo)
			assert.NoError(t, err)
			ret, err = ParseRev("master^")
			assert.NoError(t, err)
			commit6ObjId, err := ResolveRev(ret, repo)
			assert.NoError(t, err)

			if diff := cmp.Diff(fmt.Sprintf("%s commit7\n%s commit6\n", commit7ObjId, commit6ObjId), buf.String()); diff != "" {
				t.Errorf("diff is %s\n", diff)
			}

		})

	}

}

// func TestCreateFileChange(t *testing.T) {

// 	//A->Bで指定したfileのchangeのみ表示できるか
// 	//A->Bでa.txt,b.txc,c.txtの三つを変化させるが、表示対象はa.txtだけとしたい
// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	tempPath := filepath.Join(curDir, "tempDir")
// 	err = os.MkdirAll(tempPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	aname := CreateFiles(t, tempPath, "a.txt", "prev\n")
// 	bname := CreateFiles(t, tempPath, "b.txt", "prev\n")
// 	cname := CreateFiles(t, tempPath, "c.txt", "prev\n")

// 	is := []string{tempPath}

// 	var buf bytes.Buffer
// 	err = StartInit(is, &buf)
// 	assert.NoError(t, err)
// 	ss := []string{"."}
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "commit1")
// 	assert.NoError(t, err)
// 	time.Sleep(1 * time.Second)

// 	f1, err := os.Create(aname)
// 	assert.NoError(t, err)
// 	defer f1.Close()
// 	f1.Write([]byte("changed"))

// 	f2, err := os.Create(bname)
// 	assert.NoError(t, err)
// 	defer f2.Close()
// 	f2.Write([]byte("changed"))

// 	f3, err := os.Create(cname)
// 	assert.NoError(t, err)
// 	defer f3.Close()
// 	f3.Write([]byte("changed"))

// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "commit2")
// 	assert.NoError(t, err)

// }

func Test_LogFile(t *testing.T) {

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "testData/logFileChange")
	buf := new(bytes.Buffer)
	err = StartLog(tempPath, []string{"a.txt", "c.txt"}, &LogOption{
		Patch: true,
	}, buf)
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	ret, err := ParseRev("master")
	assert.NoError(t, err)
	headObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	o, err := repo.d.ReadObject(headObjId)
	assert.NoError(t, err)

	c, _ := o.(*con.CommitFromMem)
	headTime := c.Author.ReadableTime()

	ret, err = ParseRev("master^")
	assert.NoError(t, err)
	prevObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	o, err = repo.d.ReadObject(prevObjId)
	assert.NoError(t, err)

	c, _ = o.(*con.CommitFromMem)
	prevTime := c.Author.ReadableTime()

	headAtxtObjId := "21fb1e"
	headCtxtObjId := "21fb1e"
	prevAtxtObjId := "7941ff"
	prevCtxtObjId := "7941ff"

	s := buf.String()
	expected := fmt.Sprintf(
		`commit %s
Author: test <test@example.com>
Date: %s

     commit2

diff --git a/a.txt b/a.txt
index %s..%s 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-prev
+changed
\ No newline at end of file
diff --git a/c.txt b/c.txt
index %s..%s 100644
--- a/c.txt
+++ b/c.txt
@@ -1 +1 @@
-prev
+changed
\ No newline at end of file
commit %s
Author: test <test@example.com>
Date: %s

     commit1

diff --git a/a.txt b/a.txt
new file mode 100644
index 000000..%s
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
+prev
diff --git a/c.txt b/c.txt
new file mode 100644
index 000000..%s
--- a/c.txt
+++ b/c.txt
@@ -1 +1 @@
+prev
`, headObjId, headTime, prevAtxtObjId, headAtxtObjId,
		prevCtxtObjId, headCtxtObjId, prevObjId, prevTime,
		prevAtxtObjId, prevCtxtObjId,
	)

	if diff := cmp.Diff(expected, s); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func PrepareMergedFileTreeSame(t *testing.T) func() {

	// A -> B  ->  D [master]
	//    \   /
	//      C   [topic]

	//Dのlog fileでhello.txtとhello2.txtをチェックするときに
	//parentsのどちらかとファイルが同じ状態ならtreesameとして表示をしないことを確かめる
	err := os.MkdirAll(tempPath, os.ModePerm)
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
	err = StartCommit(tempPath, "test", "test@example.com", "commit1")
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
	err = StartCommit(tempPath, "test", "test@example.com", "commit2")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changed"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3")
	assert.NoError(t, err)

	err = StartMerge(tempPath, "test", "test@email.com", "commit4", []string{"test1"})
	assert.NoError(t, err)

	return func() {
		os.RemoveAll(tempPath)
	}

}

func Test_LogFileTreeSame(t *testing.T) {
	fn := PrepareMergedFileTreeSame(t)
	t.Cleanup(fn)

	buf := new(bytes.Buffer)
	//logでhello.txt、yhello2.txt両方を確かめると、
	//B->D、C->Dで必ずhello.txtかhello2,txtどちらかは変化しているのでDはtreesameとならない
	//hello.txtでのみPathFilterを掛けることによりC->Dにおいてhello.txtは変化していないのでtreesameの対象となる
	err := StartLog(tempPath, []string{"hello.txt"}, &LogOption{
		Patch: true,
	}, buf)
	assert.NoError(t, err)

	ret, err := ParseRev("master^")
	assert.NoError(t, err)
	bObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	o, err := repo.d.ReadObject(bObjId)
	assert.NoError(t, err)

	c, _ := o.(*con.CommitFromMem)
	bTime := c.Author.ReadableTime()

	ret, err = ParseRev("master^^")
	assert.NoError(t, err)
	aObjId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	o, err = repo.d.ReadObject(aObjId)
	assert.NoError(t, err)

	c, _ = o.(*con.CommitFromMem)
	aTime := c.Author.ReadableTime()

	aContentObj, err := CreateObjIdFromContent("test\n")
	assert.NoError(t, err)
	bContentObj, err := CreateObjIdFromContent("changed")
	assert.NoError(t, err)

	aShortObjId := ShortOid(aContentObj, repo.d)
	bShortObjId := ShortOid(bContentObj, repo.d)

	s := buf.String()
	expected := fmt.Sprintf(`commit %s
Author: test <test@example.com>
Date: %s

     commit3

diff --git a/hello.txt b/hello.txt
index %s..%s 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1 @@
-test
+changed
\ No newline at end of file
commit %s
Author: test <test@example.com>
Date: %s

     commit1

diff --git a/hello.txt b/hello.txt
new file mode 100644
index 000000..%s
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1 @@
+test
`, bObjId, bTime, aShortObjId, bShortObjId, aObjId, aTime, aShortObjId)
	//hello.txtのみのlogをとって、Dは片方の親CとtreesameなのでDのlogがないことを確かめる
	//ついでにhello.txtのみの変化なのでA->Bのlogが出ないことも確認
	if diff := cmp.Diff(expected, s); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}
