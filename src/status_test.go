package src

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Status(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	helloName := CreateFiles(t, tempPath, "hello.txt", "test")
	dummyName := CreateFiles(t, tempPath, "dummy.txt", "test2")

	// rel1, err := filepath.Rel(tempPath, helloName)
	// assert.NoError(t, err)
	// rel2, err := filepath.Rel(tempPath, dummyName)
	// assert.NoError(t, err)

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)

	expected := "?? dummy.txt\n?? hello.txt\n"

	fmt.Println(helloName, dummyName)

	bbuf := new(bytes.Buffer)
	err = StartStatus(bbuf, tempPath, false)
	assert.NoError(t, err)

	assert.Equal(t, expected, buf.String())

}

func Test_StatusOnlyUntracked(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	helloName := CreateFiles(t, tempPath, "hello.txt", "test")
	dummyName := CreateFiles(t, tempPath, "dummy.txt", "test2")

	rel1, err := filepath.Rel(tempPath, helloName)
	assert.NoError(t, err)

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)

	//dummy.txtだけuntracked
	ss := []string{rel1}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	expected := "?? dummy.txt\n"

	fmt.Println(helloName, dummyName)

	bbuf := new(bytes.Buffer)
	err = StartStatus(bbuf, tempPath, false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_UntrackedDirectories_NotTheirContent(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	helloName := CreateFiles(t, tempPath, "hello.txt", "test")
	dummyName := CreateFiles(t, xxxPath, "dummy.txt", "test2")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)

	//contentがuntrackなdirがある場合、dir/で表示する
	expected := "?? xxx/\n?? hello.txt\n"

	fmt.Println(helloName, dummyName)

	bbuf := new(bytes.Buffer)
	err = StartStatus(bbuf, tempPath, false)
	assert.NoError(t, err)
	bs := buf.String()
	assert.Equal(t, expected, bs)

}

func Test_UntrackedFiles_TrackedDir(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	yyyPath := filepath.Join(xxxPath, "yyy")
	err = os.MkdirAll(yyyPath, os.ModePerm)
	assert.NoError(t, err)

	zzzPath := filepath.Join(yyyPath, "zzz")
	err = os.MkdirAll(zzzPath, os.ModePerm)
	assert.NoError(t, err)
	innerName := CreateFiles(t, yyyPath, "inner.txt", "test2")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)

	rel1, err := filepath.Rel(tempPath, innerName)
	assert.NoError(t, err)

	ss := []string{rel1}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)
	//tempDir/xxx/yyy/inner.txtのみTrackedとして、
	//tempDir/xxx/outer.txt
	//tempDir/xxx/yyy/zzz/dummy.txtはunTracked
	CreateFiles(t, xxxPath, "outer.txt", "test2")
	CreateFiles(t, zzzPath, "dummy.txt", "test2")

	//contentがuntrackなdirがある場合、dir/で表示する
	expected := "?? xxx/yyy/zzz/\n?? xxx/outer.txt\n"
	bbuf := new(bytes.Buffer)
	err = StartStatus(bbuf, tempPath, false)
	assert.NoError(t, err)
	bs := buf.String()
	assert.Equal(t, expected, bs)

}

func Test_Does_Not_Display_Empty_Dir(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	expected := ""

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_List_OuterDir(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempPath)
	})

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	yyyPath := filepath.Join(xxxPath, "yyy")
	err = os.MkdirAll(yyyPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, yyyPath, "dummy.txt", "test2")

	expected := "?? xxx/\n"

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func PrepareFile(t *testing.T) (func(), error) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	// tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	tempPath := filepath.Join(curDir, "tempDir")
	err = os.MkdirAll(tempPath, os.ModePerm)
	assert.NoError(t, err)

	xxxPath := filepath.Join(tempPath, "xxx")
	err = os.MkdirAll(xxxPath, os.ModePerm)
	assert.NoError(t, err)

	yyyPath := filepath.Join(xxxPath, "yyy")
	err = os.MkdirAll(yyyPath, os.ModePerm)
	assert.NoError(t, err)

	CreateFiles(t, tempPath, "hello.txt", "hello")
	CreateFiles(t, xxxPath, "dummy.txt", "")
	CreateFiles(t, yyyPath, "dummy2.txt", "")

	is := []string{tempPath}
	var buf bytes.Buffer
	err = StartInit(is, &buf)
	assert.NoError(t, err)

	//dummy.txtだけuntracked
	ss := []string{"."}
	err = StartAdd(tempPath, "test", "test@example.com", "test", ss)
	assert.NoError(t, err)

	fn := func() {
		os.RemoveAll(tempPath)
	}

	return fn, nil
}

func Test_Print_Nothing_When_Nothing_Changed(t *testing.T) {
	cleanup, err := PrepareFile(t)
	t.Cleanup(
		cleanup,
	)
	assert.NoError(t, err)

	curDir, err := os.Getwd()
	assert.NoError(t, err)

	expected := ""

	buf := new(bytes.Buffer)
	err = StartStatus(buf, filepath.Join(curDir, "tempDir"), false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_Detect_Change_Indexed_File(t *testing.T) {
	cleanup, err := PrepareFile(t)
	t.Cleanup(
		cleanup,
	)
	assert.NoError(t, err)

	curDir, err := os.Getwd()
	assert.NoError(t, err)

	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/hello.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("change1"))
	f2, _ := os.OpenFile(filepath.Join(curDir, "tempDir/xxx/dummy.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f2.Close()
	f2.Write([]byte("change2"))

	expected := "M hello.txt\nM xxx/dummy.txt\n"

	buf := new(bytes.Buffer)
	err = StartStatus(buf, filepath.Join(curDir, "tempDir"), false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_Detect_Size_Preserving_Change(t *testing.T) {
	cleanup, err := PrepareFile(t)
	t.Cleanup(
		cleanup,
	)
	assert.NoError(t, err)

	curDir, err := os.Getwd()
	assert.NoError(t, err)

	f1, _ := os.OpenFile(filepath.Join(curDir, "tempDir/hello.txt"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f1.Close()
	f1.Write([]byte("dummy"))

	expected := "M hello.txt\n"

	buf := new(bytes.Buffer)
	err = StartStatus(buf, filepath.Join(curDir, "tempDir"), false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_Detect_ModChange(t *testing.T) {
	cleanup, err := PrepareFile(t)
	t.Cleanup(
		cleanup,
	)
	assert.NoError(t, err)

	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")

	err = os.Chmod(filepath.Join(tempPath, "hello.txt"), 0777)
	assert.NoError(t, err)

	expected := "M hello.txt\n"

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

func Test_TimeChange_is_Not_Detect_But_Update_Index(t *testing.T) {
	cleanup, err := PrepareFile(t)
	t.Cleanup(
		cleanup,
	)
	assert.NoError(t, err)

	curDir, err := os.Getwd()
	assert.NoError(t, err)

	tempPath := filepath.Join(curDir, "tempDir")

	currenttime := time.Now().Local()

	err = os.Chtimes(filepath.Join(tempPath, "hello.txt"), currenttime, currenttime)
	assert.NoError(t, err)

	expected := ""

	buf := new(bytes.Buffer)
	err = StartStatus(buf, tempPath, false)
	assert.NoError(t, err)

	bs := buf.String()

	assert.Equal(t, expected, bs)

}

// func Test_Detect_Deleted_File(t *testing.T) {
// 	cleanup, err := PrepareFile(t)
// 	t.Cleanup(
// 		cleanup,
// 	)
// 	assert.NoError(t, err)

// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	err = os.RemoveAll(filepath.Join(curDir, "tempDir/xxx/dummy.txt"))
// 	assert.NoError(t, err)
// 	expected := "D xxx/dummy.txt\n"

// 	buf := new(bytes.Buffer)
// 	err = StartStatus(buf, filepath.Join(curDir, "tempDir"))
// 	assert.NoError(t, err)

// 	bs := buf.String()

// 	assert.Equal(t, expected, bs)

// }

// func Test_Detect_Deleted_Dir(t *testing.T) {
// 	cleanup, err := PrepareFile(t)
// 	t.Cleanup(
// 		cleanup,
// 	)
// 	assert.NoError(t, err)

// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	err = os.RemoveAll(filepath.Join(curDir, "tempDir/xxx"))
// 	assert.NoError(t, err)
// 	expected := "D xxx/dummy.txt\nD xxx/yyy/dummy2.txt\n"

// 	buf := new(bytes.Buffer)
// 	err = StartStatus(buf, filepath.Join(curDir, "tempDir"))
// 	assert.NoError(t, err)

// 	bs := buf.String()

// 	assert.Equal(t, expected, bs)

// }

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
	err = StartCommit(tempPath, "test", "test@example.com", "test")
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
	//indexにはないが、commitにはあることを検知したい
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
	//indexにはないが、commitにはあることを検知したい
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
