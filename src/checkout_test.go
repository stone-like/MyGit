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

func Test_Index_Updated(t *testing.T) {
	path := PrepareCompareTwoCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	rootPath := filepath.Join(curDir, "tempDir")

	var buf bytes.Buffer
	//HEAD->HEAD^に戻す
	err = StartCheckout(rootPath, []string{"@^"}, &buf)
	assert.NoError(t, err)

	diffBuf := new(bytes.Buffer)
	//checkoutでwortkspaceとindexがかわらなくなっているはず
	//いずれuntrackedとかをのこすようになるとtestは増やさなければいけないが、今回用意したprepareではuntrackedだったりuncommitなものはないのでOK
	err = StartStatus(diffBuf, rootPath, true)
	assert.NoError(t, err)

	assert.Equal(t, "nothing to commit, working tree clean", diffBuf.String())
}

//Conflictの定義として、OldCommitとNewCommitから生成されたTreeDiffにあるPathとIndex,Workspaceを比較する、つまりCommitしていないIndexやWorkSpaceがDiffのファイルとコンフリクトしてしまうとうまくcheckoutできない
//対して、TreeDiffにはないファイルなら別にコンフリクトしないので別に良い
//ただTreeDiffの親にuntrackedがあるとだめ
func Test_ConflictUntrackParent(t *testing.T) {
	path := PrepareCompareTwoCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	rootPath := filepath.Join(curDir, "tempDir")

	xxxPath := filepath.Join(rootPath, "xxx")

	//xxx/addedがUntracked
	CreateFiles(t, xxxPath, "added.txt", "test")

	var buf bytes.Buffer
	//HEAD->HEAD^に戻す
	err = StartCheckout(rootPath, []string{"@^"}, &buf)

	e := err.(*er.ConflictOccurError)

	if diff := cmp.Diff("error: The following untracked working tree files would be overwritten by checkout:\n\txxx/dummy.txt\nPlease move or remove them before you switch branches.\n", e.ConflictDetail); diff != "" {
		t.Errorf("diff is :%s\n", diff)
	}

}

func TestPrintDetach(t *testing.T) {
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
	err = StartCommit(tempPath, "test", "test@example.com", "test", &buf)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "a.txt", "test\n")
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2", &buf)
	assert.NoError(t, err)

	var buf1 bytes.Buffer

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
	err = StartCommit(tempPath, "test", "test@example.com", "test3", &buf)
	assert.NoError(t, err)

	ret, err := ParseRev("master^")
	assert.NoError(t, err)
	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)
	objId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	shortObjId := ShortOid(objId, repo.d)

	var buf3 bytes.Buffer
	err = StartCheckout(tempPath, []string{"master^"}, &buf3)
	assert.NoError(t, err)
	str := buf3.String()
	assert.Equal(t, fmt.Sprintf("Note: checking out 'master^'\n\n%s\n\nHEAD is now at %s test\n", DETACHED_HEAD_MESSAGE, shortObjId), str)

}

func TestPrintPreviousHead(t *testing.T) {
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
	err = StartCommit(tempPath, "test", "test@example.com", "test", &buf)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "a.txt", "test\n")

	assert.NoError(t, err)
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test2", &buf)
	assert.NoError(t, err)

	var buf1 bytes.Buffer

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
	err = StartCommit(tempPath, "test", "test@example.com", "test3", &buf)
	assert.NoError(t, err)

	ret, err := ParseRev("master^")
	assert.NoError(t, err)
	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)
	objId, err := ResolveRev(ret, repo)
	assert.NoError(t, err)

	var buf3 bytes.Buffer
	err = StartCheckout(tempPath, []string{"master^"}, &buf3)
	assert.NoError(t, err)

	var buf4 bytes.Buffer
	err = StartCheckout(tempPath, []string{"to"}, &buf4)
	assert.NoError(t, err)
	str := buf4.String()

	var bufp bytes.Buffer
	PrintHeadPosition("Previous HEAD position was", objId, repo, &bufp)

	assert.Equal(t, bufp.String()+"Switched to branch 'to'\n", str)

}
