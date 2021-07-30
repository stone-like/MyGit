package src

import (
	con "mygit/src/database/content"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

// func Test_CompareTwoCommit(t *testing.T) {
// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	tempPath := filepath.Join(curDir, "tempDir")
// 	err = os.MkdirAll(tempPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	xxxPath := filepath.Join(tempPath, "xxx")
// 	err = os.MkdirAll(xxxPath, os.ModePerm)
// 	assert.NoError(t, err)

// 	CreateFiles(t, tempPath, "hello.txt", "test\n")
// 	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2\n")

// 	is := []string{tempPath}
// 	var buf bytes.Buffer
// 	err = StartInit(is, &buf)
// 	assert.NoError(t, err)
// 	ss := []string{"."}
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "test")
// 	assert.NoError(t, err)

// 	os.Remove(dummyName)
// 	CreateFiles(t, xxxPath, "dummy2.txt", "test\n")
// 	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
// 	assert.NoError(t, err)
// 	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
// 	assert.NoError(t, err)
// 	err = StartCommit(tempPath, "test", "test@example.com", "test2")
// 	assert.NoError(t, err)
// }

func Test_TreeDiff(t *testing.T) {
	a := "ea7c3064d57fa1b07b4da70bb7a534d42da7bd7f"
	b := "3c340347598d1070b988e0b77cd2e8f14aa501e1"

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	rootPath := filepath.Join(curDir, "testData/treeDiffDir")
	gitPath := filepath.Join(rootPath, ".git")
	dbPath := filepath.Join(gitPath, "objects")
	repo := GenerateRepository(rootPath, gitPath, dbPath)
	td := GenerateTreeDiff(repo)

	td.CompareObjId(a, b)

	d1 := &con.Entry{
		ObjId: "180cf8328022becee9aaa2577a8f84ea2b9f3827",
		Path:  "xxx/dummy.txt",
		Mode:  33188,
	}
	d2 := &con.Entry{
		ObjId: "9daeafb9864cf43055ae93beb0afd6c7d144bfa4",
		Path:  "xxx/dummy2.txt",
		Mode:  33188,
	}

	e1 := []*con.Entry{d1, nil}
	e2 := []*con.Entry{nil, d2}

	expectedMap := map[string][]*con.Entry{"xxx/dummy.txt": e1, "xxx/dummy2.txt": e2}

	if diff := cmp.Diff(expectedMap, td.Changes); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}
