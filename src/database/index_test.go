package database

import (
	con "mygit/src/database/content"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func Test_DiscardConflictFileToDir(t *testing.T) {

	i := GenerateIndex("temp")

	fn := func(path, objId string, state con.FileState) *con.Entry {
		return &con.Entry{
			Path:  path,
			ObjId: objId,
		}
	}

	i.Add("alice.txt", "2b2b", nil, fn)
	i.Add("bob.txt", "2b2b", nil, fn)
	i.Add("alice.txt/nested.txt", "2b2b", nil, fn)

	for ind, p := range []string{"bob.txt", "alice.txt/nested.txt"} {
		assert.Equal(t, p, i.Keys[ind])
	}
}

func Test_DiscardConflictDirToFile(t *testing.T) {

	i := GenerateIndex("temp")

	fn := func(path, objId string, state con.FileState) *con.Entry {
		return &con.Entry{
			Path:  path,
			ObjId: objId,
		}
	}

	i.Add("alice.txt", "2b2b", nil, fn)
	i.Add("nested/bob.txt", "2b2b", nil, fn)
	i.Add("nested", "2b2b", nil, fn)

	for ind, p := range []string{"alice.txt", "nested"} {
		assert.Equal(t, p, i.Keys[ind])
	}
}

func Test_DiscardConflictDirToFile2(t *testing.T) {

	i := GenerateIndex("temp")

	fn := func(path, objId string, state con.FileState) *con.Entry {
		return &con.Entry{
			Path:  path,
			ObjId: objId,
		}
	}

	i.Add("alice.txt", "2b2b", nil, fn)
	i.Add("nested/bob.txt", "2b2b", nil, fn)
	i.Add("nested/inner/ccc.txt", "2b2b", nil, fn)

	i.Add("nested", "2b2b", nil, fn)

	for ind, p := range []string{"alice.txt", "nested"} {
		assert.Equal(t, p, i.Keys[ind])
	}
}

func Test_ReadFromFile(t *testing.T) {

	cur, err := os.Getwd()
	assert.NoError(t, err)

	i := GenerateIndex(filepath.Join(cur, "testData/testindex"))

	err = i.Load()
	assert.NoError(t, err)

	h := &con.Entry{
		CTime:      1624795630,
		CTime_nsec: 850762100,
		MTime:      1624787271,
		MTime_nsec: 760778600,
		Dev:        43,
		Ino:        230271,
		Mode:       33188,
		UId:        1000,
		GId:        1000,
		Size:       4,
		ObjId:      "30d74d258442c7c65512eafab474568dd706c430",
		Flags:      9,
		Path:       "hello.txt",
	}
	x := &con.Entry{
		CTime:      1624979048,
		CTime_nsec: 72374100,
		MTime:      1624979048,
		MTime_nsec: 72374100,
		Dev:        43,
		Ino:        206031,
		Mode:       33188,
		UId:        1000,
		GId:        1000,
		Size:       7,
		ObjId:      "56bf82ee60ad6536eed6b79095ca62d8bcae4068",
		Flags:      12,
		Path:       "xxx/test.txt",
	}

	for _, e := range []*con.Entry{h, x} {
		if diff := cmp.Diff(e, i.Entries[e.Path]); diff != "" {
			t.Errorf("diff is %s\n", diff)
		}
	}

}
