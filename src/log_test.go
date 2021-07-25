package src

import (
	"bytes"
	"fmt"
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
