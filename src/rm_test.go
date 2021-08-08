package src

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func PrepareCommit(t *testing.T) string {

	cur, err := os.Getwd()
	assert.NoError(t, err)
	tempPath, err := ioutil.TempDir(cur, "")
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
	err = StartCommit(tempPath, "test", "test@example.com", "commit1", &buf)
	assert.NoError(t, err)

	return tempPath
}

func TestIndexDir(t *testing.T) {
	tempPath := PrepareCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	var buf bytes.Buffer
	err := StartRm(tempPath, []string{"xxx"}, &RmOption{}, &buf)
	assert.NoError(t, err)

	str := buf.String()
	expected := "not removing 'xxx' recursively without -r"

	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func TestIndexFile(t *testing.T) {
	tempPath := PrepareCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	var buf bytes.Buffer
	err := StartRm(tempPath, []string{"notexistsFilename"}, &RmOption{}, &buf)
	assert.NoError(t, err)

	str := buf.String()
	expected := "pathspec 'notexistsFilename' did not match any files"

	if diff := cmp.Diff(expected, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func TestIndexDirRecur(t *testing.T) {
	tempPath := PrepareCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)
	repo.i.Load()

	strs, err := RunExamine("xxx", true, repo)
	assert.NoError(t, err)

	if diff := cmp.Diff([]string{"xxx/dummy.txt"}, strs); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func TestIndexFileRecur(t *testing.T) {
	tempPath := PrepareCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	gitPath := filepath.Join(tempPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(tempPath, gitPath, dbPath)
	repo.i.Load()

	strs, err := RunExamine("xxx/dummy.txt", true, repo)
	assert.NoError(t, err)

	if diff := cmp.Diff([]string{"xxx/dummy.txt"}, strs); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}

func TestRunRm_WORKSPACE_CHANGED(t *testing.T) {
	tempPath := PrepareCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	f, err := os.Create(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)
	f.Write([]byte("temp"))
	f.Close()

	var buf bytes.Buffer
	err = StartRm(tempPath, []string{"hello.txt"}, &RmOption{}, &buf)
	assert.NoError(t, err)

	str := buf.String()
	expect := `error: the following file has local modifications:
   hello.txt
`

	if diff := cmp.Diff(expect, str); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}

}

func TestRunRm_SuccessFullyDeleted(t *testing.T) {
	tempPath := PrepareCommit(t)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})
	helloPath := filepath.Join(tempPath, "hello.txt")
	var buf bytes.Buffer
	err := StartRm(tempPath, []string{"hello.txt"}, &RmOption{}, &buf)
	assert.NoError(t, err)

	stat, _ := os.Stat(helloPath)
	assert.Nil(t, stat)

}
