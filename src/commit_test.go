package src

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func CreateFiles(t *testing.T, dir string, name string, content string) string {
	helloPath := filepath.Join(dir, name)
	f1, err := os.Create(helloPath)
	assert.NoError(t, err)
	defer f1.Close()
	_, err = f1.Write([]byte(content))
	assert.NoError(t, err)

	return f1.Name()
}

func Test_Dummy(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	// t.Cleanup(func() {
	// 	os.RemoveAll(tempPath)
	// })

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	helloName := CreateFiles(t, tempPath, "hello.txt", "test")
	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2")

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
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

}

func Test_Commit(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	// t.Cleanup(func() {
	// 	os.RemoveAll(tempPath)
	// })

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	helloName := CreateFiles(t, xxxPath, "hello.txt", "test")
	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2")

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
	err = StartCommit(tempPath, "test", "test@example.com", "test")
	assert.NoError(t, err)

	dummyPath := filepath.Join(xxxPath, "dummy.txt")

	os.RemoveAll(dummyPath)

	err = os.MkdirAll(dummyPath, os.ModePerm)
	assert.NoError(t, err)

	testName := CreateFiles(t, dummyPath, "test1.txt", "dummy")

	rel3, err := filepath.Rel(tempPath, testName)
	assert.NoError(t, err)

	ss = []string{rel3, rel1}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

}
