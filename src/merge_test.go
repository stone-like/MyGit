package src

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func PrepareMerge(t *testing.T) func() {

	// A -> B   master
	//   -> C  test1

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
	err = StartCommit(tempPath, "test", "test@example.com", "commit3")
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	err = StartCheckout(tempPath, []string{"master"}, &buf)
	assert.NoError(t, err)

	f2, err := os.Create(helloPath)
	assert.NoError(t, err)
	f2.Write([]byte("changed"))

	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "commit6")
	assert.NoError(t, err)

	return func() {
		os.RemoveAll(tempPath)
	}
}

func Test_Merge(t *testing.T) {
	fn := PrepareMerge(t)
	t.Cleanup(fn)

	//masterにtest1をmerge
	err := StartMerge(tempPath, "test", "test@email.com", "merged", []string{"test1"})
	assert.NoError(t, err)

	c1, err := ioutil.ReadFile(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)
	c2, err := ioutil.ReadFile(filepath.Join(tempPath, "hello2.txt"))
	assert.NoError(t, err)

	assert.Equal(t, "changed", string(c1))
	assert.Equal(t, "changed2", string(c2))
}
