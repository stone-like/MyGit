package src

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/stretchr/testify/assert"
)

//diffの原理は大体理解したのでいったん既存の奴を使う、ブランチまで終わったら自力で実装
func TestDiff(t *testing.T) {
	cur, err := os.Getwd()
	assert.NoError(t, err)
	testDataPath := filepath.Join(cur, "testdata")
	prevPath := filepath.Join(testDataPath, "prev.txt")
	diffPath := filepath.Join(testDataPath, "diff.txt")
	s1, err := ioutil.ReadFile(prevPath)
	assert.NoError(t, err)
	s2, err := ioutil.ReadFile(diffPath)
	assert.NoError(t, err)
	edits := myers.ComputeEdits(span.URIFromPath("a.txt"), string(s1), string(s2))
	diff := fmt.Sprint(gotextdiff.ToUnified("a.txt", "b.txt", string(s1), edits))

	fmt.Print(diff)
}

func TestDiffStatusModContent_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/hello.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("change1\n"))

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "diff --git a/hello.txt b/hello.txt\nindex 30d74d..981663 100644\n"

	assert.Equal(t, expected, buf.String())
}

func TestDiffStatusModMode_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "diff --git a/hello.txt b/hello.txt\nold mode 100644\nnew mode 100755\n"

	assert.Equal(t, expected, buf.String())
}

func TestDiffStatusModModeAndContent_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)
	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/hello.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("change1\n"))

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "diff --git a/hello.txt b/hello.txt\nold mode 100644\nnew mode 100755\nindex 30d74d..981663\n"

	assert.Equal(t, expected, buf.String())
}

func TestDiffStatusDeleted_Index_And_WorkSpace(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")

	err = os.RemoveAll(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, false)
	assert.NoError(t, err)

	expected := "diff --git a/hello.txt b/hello.txt\ndeleted file mode 100644\nindex 30d74d..000000\n"

	assert.Equal(t, expected, buf.String())
}

func TestDiffStatusModModeAndContent_Index_And_Commit(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)
	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/xxx/dummy2.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("require multipleline\nfunc changed()\nchanged executed\n\n\n\n\n\n\n\n\nSimulate End\nSimulate Restart\n"))

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, true)
	assert.NoError(t, err)

	expected := "diff --git a/hello.txt b/hello.txt\nold mode 100644\nnew mode 100755\nindex 30d74d..981663\n"

	assert.Equal(t, expected, buf.String())
}

func TestDiffStatusDeleted_Index_And_Commit(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")

	err = os.RemoveAll(filepath.Join(tempPath, "hello.txt"))
	assert.NoError(t, err)

	//git rmを実装するまではindexを削除しなければindexからdeleteできない
	err = os.RemoveAll(filepath.Join(curDir, "tempDir/.git/index"))
	assert.NoError(t, err)

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, true)
	assert.NoError(t, err)

	expected := "diff --git a/hello.txt b/hello.txt\ndeleted file mode 100644\nindex 30d74d..000000\n"

	assert.Equal(t, expected, buf.String())
}

func TestDiffStatusAdded_Index_And_Commit(t *testing.T) {
	fn := Prepare(t)

	t.Cleanup(fn)
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempPath := filepath.Join(curDir, "tempDir")

	CreateFiles(t, tempPath, "added.txt", "added\n")

	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	err = StartDiff(buf, tempPath, true)
	assert.NoError(t, err)

	expected := "diff --git a/added.txt b/added.txt\nnew file mode 100644\nindex 000000..d5f7fc\n"

	assert.Equal(t, expected, buf.String())
}
