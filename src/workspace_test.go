package src

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	c "mygit/src/database/content"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestReadList(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir1)
	})

	tempDir2, err := ioutil.TempDir(tempDir1, "tempDir")
	assert.NoError(t, err)

	tempDir3, err := ioutil.TempDir(tempDir2, "tempDir")
	assert.NoError(t, err)

	f1, err := ioutil.TempFile(tempDir1, "test")
	assert.NoError(t, err)
	defer f1.Close()
	f2, err := ioutil.TempFile(tempDir1, "test")
	assert.NoError(t, err)
	defer f2.Close()
	f3, err := ioutil.TempFile(tempDir2, "test")
	assert.NoError(t, err)
	defer f3.Close()
	f4, err := ioutil.TempFile(tempDir3, "test")
	assert.NoError(t, err)
	defer f4.Close()

	w := &WorkSpace{
		Path: tempDir1,
	}

	fs, err := w.ListFiles(tempDir1)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(fs))

}

func TestReadFilet(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir1)
	})

	f0, err := os.Create(filepath.Join(tempDir1, "ok.txt"))
	assert.NoError(t, err)
	defer f0.Close()

	f1, err := os.Create(filepath.Join(tempDir1, "temp.yml"))
	assert.NoError(t, err)
	defer f1.Close()

	srcDir := filepath.Join(tempDir1, "src")
	err = os.MkdirAll(srcDir, os.ModePerm)
	assert.NoError(t, err)
	f2, err := ioutil.TempFile(srcDir, "test")
	assert.NoError(t, err)
	defer f2.Close()
	f3, err := ioutil.TempFile(srcDir, "test")
	assert.NoError(t, err)
	defer f3.Close()

	w := &WorkSpace{
		Path: tempDir1,
	}

	ignoreList := []string{".", "src", "temp.yml"}

	fs, err := w.FilePathWalkDir(tempDir1, ignoreList)
	assert.NoError(t, err)

	//src/*とtemp.yamlは除外されているので、ok.txtだけになるはず1
	assert.Equal(t, 1, len(fs))

}

func Test_RecursiveTree(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir1)
	})

	//tempDir1/foo/bar/test.txt
	//                /test2.txt
	//        /foo/bar.txt
	//        /foo.txt

	fooDir := filepath.Join(tempDir1, "foo")
	err = os.MkdirAll(fooDir, os.ModePerm)
	assert.NoError(t, err)

	barDir := filepath.Join(fooDir, "bar")

	err = os.MkdirAll(barDir, os.ModePerm)
	assert.NoError(t, err)

	barPath := filepath.Join(fooDir, "bar.txt")

	f0, err := os.Create(barPath)
	assert.NoError(t, err)
	defer f0.Close()

	testPath := filepath.Join(barDir, "test.txt")
	f1, err := os.Create(testPath)
	assert.NoError(t, err)
	defer f1.Close()

	fooPath := filepath.Join(tempDir1, "foo.txt")
	f2, err := os.Create(fooPath)
	assert.NoError(t, err)
	defer f2.Close()

	test2Path := filepath.Join(barDir, "test2.txt")
	f3, err := os.Create(test2Path)
	assert.NoError(t, err)
	defer f3.Close()

	w := &WorkSpace{
		Path: tempDir1,
	}
	pathList, err := w.ListFiles(tempDir1)
	assert.NoError(t, err)
	var entryList []*c.Entry

	for _, path := range pathList {
		con, err := w.ReadFile(path)

		assert.NoError(t, err)
		b := &c.Blob{
			Content: con,
		}

		stat, err := w.StatFile(path)
		assert.NoError(t, err)
		entryList = append(entryList, &c.Entry{
			Mode:  int(stat.Mode()),
			Path:  path,
			ObjId: b.ObjId,
		})
	}

	tr := c.GenerateTree()
	tr.Build(entryList)

	fooState, _ := os.Stat(fooPath)
	fooEntry := &c.Entry{
		Path:  "foo.txt",
		ObjId: "",
		Mode:  int(fooState.Mode()),
	}
	//windowsでは\\,linuxでは/になるがこれをテストでどうすればいいか悩む
	barState, _ := os.Stat(barPath)
	barEntry := &c.Entry{
		Path:  "foo/bar.txt",
		ObjId: "",
		Mode:  int(barState.Mode()),
	}

	testState, _ := os.Stat(testPath)
	testEntry := &c.Entry{
		Path:  "foo/bar/test.txt",
		ObjId: "",
		Mode:  int(testState.Mode()),
	}
	test2State, _ := os.Stat(test2Path)
	test2Entry := &c.Entry{
		Path:  "foo/bar/test2.txt",
		ObjId: "",
		Mode:  int(test2State.Mode()),
	}

	fooBarTreeMap := map[string]c.Object{
		"foo/bar/test.txt":  testEntry,
		"foo/bar/test2.txt": test2Entry,
	}

	fooBarTree := &c.Tree{
		Entries: fooBarTreeMap,
	}

	fooTreeMap := map[string]c.Object{
		"foo/bar.txt": barEntry,
		"foo/bar":     fooBarTree,
	}

	fooTree := &c.Tree{
		Entries: fooTreeMap,
	}

	treeMap := map[string]c.Object{
		"foo.txt": fooEntry,
		"foo":     fooTree,
	}

	// opt := cmpopts.IgnoreFields(c.Entry{}, "State")

	if diff := cmp.Diff(treeMap, tr.Entries); diff != "" {
		t.Errorf("diff is %s\n", diff)
	}
}
