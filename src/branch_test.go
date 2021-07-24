package src

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func Test_BranchUpdateRef(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartBranch(tempPath, []string{"master"}, &BranchOption{}, buf)
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")

	expectedContent, err := ioutil.ReadFile(gitPath + "/HEAD")
	assert.NoError(t, err)

	targetPath := filepath.Join(gitPath, "refs/heads/master")
	_, err = os.Stat(targetPath)
	assert.NoError(t, err)
	targetContent, err := ioutil.ReadFile(targetPath)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, targetContent)
}

func PrepareThreeCommit(t *testing.T) func() {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	helloName := CreateFiles(t, tempPath, "hello.txt", "test\n")
	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

	rel1, err := filepath.Rel(tempPath, helloName)
	assert.NoError(t, err)
	rel2, err := filepath.Rel(tempPath, dummyName)
	assert.NoError(t, err)
	is := []string{tempPath}

	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{rel1, rel2}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit1")
	assert.NoError(t, err)

	CreateFiles(t, xxxPath, "dummy2.txt", "test2\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit2")
	assert.NoError(t, err)

	CreateFiles(t, xxxPath, "dummy3.txt", "test2\n")
	ss = []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit3")
	assert.NoError(t, err)

	return func() {
		os.RemoveAll(tempPath)
	}
}

func Test_CreateBranch(t *testing.T) {
	fn := PrepareThreeCommit(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartBranch(tempPath, []string{"master", "@^^"}, &BranchOption{}, buf)
	assert.NoError(t, err)

	ret, err := ParseRev("@^^")
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)
	objId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	targetPath := filepath.Join(gitPath, "refs/heads/master")
	_, err = os.Stat(targetPath)
	assert.NoError(t, err)
	targetContent, err := ioutil.ReadFile(targetPath)
	assert.NoError(t, err)
	assert.Equal(t, objId, strings.TrimSpace(string(targetContent)))
}

func Test_DeleteBranch(t *testing.T) {
	fn := PrepareThreeCommit(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, buf)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"test2"}, &BranchOption{}, buf)
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	targetPath := filepath.Join(gitPath, "refs/heads")

	lists, err := repo.w.ListDir(targetPath)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(lists)) //master,test1,test2

	err = StartBranch(tempPath, []string{"test1", "test2"}, &BranchOption{
		HasD: true,
		HasF: true,
	}, buf)
	assert.NoError(t, err)

	deletedlists, err := repo.w.ListDir(targetPath)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(deletedlists)) //master

}

func Test_DeleteBranchAndParentDir(t *testing.T) {
	fn := PrepareThreeCommit(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartBranch(tempPath, []string{"xxx/yyy/test1"}, &BranchOption{}, buf)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"xxx/yyy/test2"}, &BranchOption{}, buf)
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)

	targetPath := filepath.Join(gitPath, "refs/heads")

	lists, err := repo.w.ListDir(targetPath)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(lists)) //master,xxx(Dir)

	err = StartBranch(tempPath, []string{"xxx/yyy/test1", "xxx/yyy/test2"}, &BranchOption{
		HasD: true,
		HasF: true,
	}, buf)
	assert.NoError(t, err)

	xstat, _ := os.Stat(filepath.Join(targetPath, "xxx"))
	assert.Nil(t, xstat)
	ystat, _ := os.Stat(filepath.Join(targetPath, "xxx", "yyy"))
	assert.Nil(t, ystat)

	//テスト内容としてrefs/heads/xxx/yyy/testブランチを作って削除した後、
	//refs/heads/xxxとyyyが存在しなければOK
}

func Test_ListBranch(t *testing.T) {
	fn := PrepareThreeCommit(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, buf)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"test2"}, &BranchOption{}, buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test2"}, buf)
	assert.NoError(t, err)

	var listBuf bytes.Buffer
	err = StartBranch(tempPath, []string{}, &BranchOption{}, &listBuf)
	assert.NoError(t, err)

	str := listBuf.String()
	if diff := cmp.Diff("  master\n  test1\n* test2\n", str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func Test_ListBranchVerbose(t *testing.T) {
	fn := PrepareThreeCommit(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartBranch(tempPath, []string{"test1"}, &BranchOption{}, buf)
	assert.NoError(t, err)
	err = StartBranch(tempPath, []string{"test2"}, &BranchOption{}, buf)
	assert.NoError(t, err)

	err = StartCheckout(tempPath, []string{"test2"}, buf)
	assert.NoError(t, err)

	var listBuf bytes.Buffer
	err = StartBranch(tempPath, []string{}, &BranchOption{
		HasV: true,
	}, &listBuf)
	assert.NoError(t, err)

	ret, err := ParseRev("master")
	assert.NoError(t, err)

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)
	objId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	shortObjId := ShortOid(objId, repo.d)

	//今回master,test1,test2のobjIdは全部同じとしている

	str := listBuf.String()
	if diff := cmp.Diff(fmt.Sprintf("  master %s commit3\n  test1  %s commit3\n* test2  %s commit3\n", shortObjId, shortObjId, shortObjId), str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}
