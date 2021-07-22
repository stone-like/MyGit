package database

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	c "mygit/src/database/content"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeflate(t *testing.T) {
	var in bytes.Buffer
	CompressWithDeflate(&in, "testabcd")
	zr, err := zlib.NewReader(&in)
	assert.NoError(t, err)
	var out bytes.Buffer
	io.Copy(&out, zr)

	str := out.String()
	fmt.Println(str)
}

func TestReadObjectTypeAndSize(t *testing.T) {
	cur, err := os.Getwd()
	assert.NoError(t, err)

	dataPath := filepath.Join(cur, "testData/TestGitData/.git/objects")
	d := &Database{
		Path: dataPath,
	}
	//hello.txt blob 4testを取得
	r, err := d.GetContent("9daeafb9864cf43055ae93beb0afd6c7d144bfa4")
	assert.NoError(t, err)
	hAndR, err := d.ScanObjectHeader(r)
	assert.NoError(t, err)
	assert.Equal(t, "blob", hAndR.ObjType)
	assert.Equal(t, "5", hAndR.Size)
}

func TestReadBlob(t *testing.T) {
	cur, err := os.Getwd()
	assert.NoError(t, err)

	dataPath := filepath.Join(cur, "testData/TestGitData/.git/objects")
	d := &Database{
		Path: dataPath,
	}
	//hello.txt blob 4testを取得
	r, err := d.GetContent("9daeafb9864cf43055ae93beb0afd6c7d144bfa4")
	assert.NoError(t, err)
	hAndR, err := d.ScanObjectHeader(r)
	assert.NoError(t, err)
	assert.Equal(t, "blob", hAndR.ObjType)

	obj, err := c.Parse(hAndR.ObjType, hAndR.Reader)
	assert.NoError(t, err)

	blob, ok := obj.(*c.Blob)

	assert.Equal(t, ok, true)

	assert.Equal(t, "test\n", blob.Content)
}

func TestReadCommit(t *testing.T) {
	cur, err := os.Getwd()
	assert.NoError(t, err)

	dataPath := filepath.Join(cur, "testData/TestGitData/.git/objects")
	d := &Database{
		Path: dataPath,
	}
	//commitを取得
	r, err := d.GetContent("03fb89c2c5c6ad1c0d21c4ac77595175eeba6b27")
	assert.NoError(t, err)
	hAndR, err := d.ScanObjectHeader(r)
	assert.NoError(t, err)
	assert.Equal(t, "commit", hAndR.ObjType)

	o, err := c.Parse(hAndR.ObjType, hAndR.Reader)
	assert.NoError(t, err)
	com, ok := o.(*c.CommitFromMem)

	assert.Equal(t, ok, true)

	assert.Equal(t, "this is commitMessage\n\nhardToRead", com.Message)
	assert.Equal(t, "this is commitMessage", com.GetFirstLineMessage())

}

func TestReadTree(t *testing.T) {
	cur, err := os.Getwd()
	assert.NoError(t, err)

	dataPath := filepath.Join(cur, "testData/TestGitData/.git/objects")
	d := &Database{
		Path: dataPath,
	}
	//treeを取得
	r, err := d.GetContent("b5d32e66a5ff2ad54762006aa676c6d255dc7864")
	assert.NoError(t, err)
	hAndR, err := d.ScanObjectHeader(r)
	assert.NoError(t, err)
	assert.Equal(t, "tree", hAndR.ObjType)

	o, err := c.Parse(hAndR.ObjType, hAndR.Reader)
	assert.NoError(t, err)
	tree, ok := o.(*c.Tree)

	assert.Equal(t, ok, true)

	assert.Equal(t, 2, len(tree.Entries))

}

func TestPrintCurrentTree(t *testing.T) {
	cur, err := os.Getwd()
	assert.NoError(t, err)

	dataPath := filepath.Join(cur, "testData/TestGitData/.git/objects")
	d := &Database{
		Path: dataPath,
	}

	co, err := d.ReadObject("03fb89c2c5c6ad1c0d21c4ac77595175eeba6b27")
	assert.NoError(t, err)
	buf := new(bytes.Buffer)

	cc, ok := co.(*c.CommitFromMem)
	assert.Equal(t, ok, true)
	//treeを取得
	err = d.ShowTree(cc.Tree, buf)
	assert.NoError(t, err)

	str := buf.String()

	expected := "100644 180cf8328022becee9aaa2577a8f84ea2b9f3827 xxx/dummy.txt\n100644 9daeafb9864cf43055ae93beb0afd6c7d144bfa4 hello.txt\n"

	assert.Equal(t, expected, str)

}
