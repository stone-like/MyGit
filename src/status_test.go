package src

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Prepare(t *testing.T) func() {
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
	dummy2Name := CreateFiles(t, xxxPath, "dummy2.txt", "require multipleline\nfunc simulate()\n\n\n\n\n\n\n\n\nSimulate End\n")

	rel1, err := filepath.Rel(tempPath, helloName)
	assert.NoError(t, err)
	rel2, err := filepath.Rel(tempPath, dummyName)
	assert.NoError(t, err)
	rel3, err := filepath.Rel(tempPath, dummy2Name)
	assert.NoError(t, err)

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)
	ss := []string{rel1, rel2, rel3}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	err = StartCommit(tempPath, "test", "test@example.com", "test", &buf)
	assert.NoError(t, err)

	return func() {
		os.RemoveAll(tempPath)
	}
}

func Test_Detect_File_Added_To_Tracked_Dir(t *testing.T) {
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
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "A  xxx/added.txt\n"

	assert.Equal(t, expected, buf.String())

}
func Test_Detect_File_Added_To_UnTracked_Dir(t *testing.T) {
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
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "A  yyy/added.txt\n"

	assert.Equal(t, expected, buf.String())

}

func Test_Detect_ModifiedMode(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "M  hello.txt\n"

	assert.Equal(t, expected, buf.String())

}

func Test_Detect_ModifiedContent(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/hello.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("change1"))

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "M  hello.txt\n"

	assert.Equal(t, expected, buf.String())

}

func Test_Detect_Deleted_File(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	//index??????????????????commit????????????????????????????????????
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/xxx/dummy.txt"))
	assert.NoError(t, err)
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	expected := "D  xxx/dummy.txt\n"

	buf := new(bytes.Buffer)
	err = StartStatus(buf, filepath.Join(curDir, "tempDir"), false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_Detect_Deleted_Dir(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)

	curDir, err := os.Getwd()
	assert.NoError(t, err)
	//index??????????????????commit????????????????????????????????????
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/xxx"))
	assert.NoError(t, err)
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	expected := "D  xxx/dummy.txt\nD  xxx/dummy2.txt\n"

	buf := new(bytes.Buffer)
	err = StartStatus(buf, filepath.Join(curDir, "tempDir"), false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}
