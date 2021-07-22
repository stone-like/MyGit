package src

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Detect_File_Added_To_Tracked_Dir_Long(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	xxxPath := filepath.Join(tempPath, "xxx")

	CreateFiles(t, xxxPath, "added.txt", "test")
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, true)
	assert.NoError(t, err)

	expected := fmt.Sprintf("Changes to be Commited:\n\t%*s%s\n", 20, "new file:", "xxx/added.txt")

	bs := buf.String()

	assert.Equal(t, expected, bs)

}
func Test_Detect_File_Added_To_UnTracked_Dir_Long(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	yyyPath := filepath.Join(tempPath, "yyy")
	err = os.MkdirAll(yyyPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, yyyPath, "added.txt", "test")
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, true)
	assert.NoError(t, err)

	expected := fmt.Sprintf("Changes to be Commited:\n\t%*s%s\n", 20, "new file:", "yyy/added.txt")
	bs := buf.String()
	assert.Equal(t, expected, bs)

}

func Test_Detect_WorkSpaceAdded(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	xxxPath := filepath.Join(tempPath, "xxx")

	//workSpaceに作っただけでAddはしていない
	CreateFiles(t, xxxPath, "added.txt", "test")

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, true)
	assert.NoError(t, err)

	expected := fmt.Sprintf("Untracked files:\n\t %s\nnothing added to commit but untracked files present", "xxx/added.txt")

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_Detect_WorkSpaceModified(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")

	//workSpaceを更新しただけでAddはしていない
	err = os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, true)
	assert.NoError(t, err)

	expected := fmt.Sprintf("Changes not staged for commit:\n\t%*s%s\nno changes added to commit", 20, "modified:", "hello.txt")

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_Detect_WorkSpaceDeleted(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")

	//workSpaceを更新しただけでAddはしていない
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/hello.txt"))
	assert.NoError(t, err)
	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, true)
	assert.NoError(t, err)

	expected := fmt.Sprintf("Changes not staged for commit:\n\t%*s%s\nno changes added to commit", 20, "deleted:", "hello.txt")

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_PrintNothing(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, true)
	assert.NoError(t, err)

	expected := fmt.Sprintf("nothing to commit, working tree clean")

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_Print_Ordered(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	yyyPath := filepath.Join(tempPath, "yyy")
	err = os.MkdirAll(yyyPath, os.ModePerm)
	assert.NoError(t, err)

	err = os.RemoveAll(filepath.Join(curDir, "tempDir/xxx/dummy.txt"))
	assert.NoError(t, err)
	err = os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)
	xxxPath := filepath.Join(tempPath, "xxx")
	CreateFiles(t, xxxPath, "dummz.txt", "test")

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, true)
	assert.NoError(t, err)

	expected := fmt.Sprintf("Changes to be Commited:\n\t%*s%s\t%*s%s\t%*s%s\n", 20, "modified:", "hello.txt", 20, "deleted:", "xxx/dummy.txt", 20, "new file:", "xxx/dummz.txt")
	bs := buf.String()
	assert.Equal(t, expected, bs)

}
