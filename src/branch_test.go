package src

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BranchUpdateRef(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	buf := new(bytes.Buffer)
	err = StartBranch(tempPath, []string{"master"}, buf)
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

	err = StartInit(is)
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
	err = StartBranch(tempPath, []string{"master", "@^^"}, buf)
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
