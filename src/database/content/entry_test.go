package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func createReg(path string) error {
// 	if _, err := os.Stat(path); err != nil {
// 		perm := "0644"
// 		perm32, _ := strconv.ParseUint(perm, 8, 32)
// 		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.FileMode(perm32))
// 		f.Write([]byte("test1"))
// 		defer f.Close()
// 		if err != nil {
// 			return err
// 		}

// 		return nil
// 	}

// 	return nil

// }

// func createExec(path string) error {
// 	if _, err := os.Stat(path); err != nil {
// 		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
// 		f.Write([]byte("test1"))
// 		defer f.Close()
// 		if err != nil {
// 			return err
// 		}

// 		return nil
// 	}

// 	return nil

// }

// func Test_GenerateParent(t *testing.T) {
// 	e := &Entry{}
// 	ret := e.ParentDirs("test/aaa/ccc/ddd.txt")

// 	for i, p := range []string{
// 		"test",
// 		"test/aaa",
// 		"test/aaa/ccc",
// 	} {
// 		assert.Equal(t, p, ret[i])
// 	}
// }

// func Test_Entry(t *testing.T) {
// 	curDir, err := os.Getwd()
// 	assert.NoError(t, err)

// 	perm := "0666"
// 	perm32, _ := strconv.ParseUint(perm, 8, 32)

// 	tempDir1 := filepath.Join(curDir, "tempDir")
// 	err = os.MkdirAll(tempDir1, os.FileMode(perm32))

// 	assert.NoError(t, err)

// 	// t.Cleanup(func() {
// 	// 	os.RemoveAll(tempDir1)
// 	// })

// 	regPath := filepath.Join(tempDir1, "reg.txt")
// 	execPath := filepath.Join(tempDir1, "exec.txt")

// 	err = createReg(regPath)
// 	assert.NoError(t, err)
// 	err = createExec(execPath)
// 	assert.NoError(t, err)

// 	s1, err := os.Stat(regPath)
// 	assert.NoError(t, err)
// 	f1, err := os.Open(regPath)
// 	assert.NoError(t, err)
// 	defer f1.Close()

// 	regE := &Entry{
// 		Name:  "exec",
// 		ObjId: "test",
// 		State: s1,
// 	}

// 	assert.Equal(t, false, regE.isExec())

// 	s2, err := os.Stat(execPath)
// 	assert.NoError(t, err)
// 	f2, err := os.Open(execPath)
// 	assert.NoError(t, err)
// 	defer f2.Close()

// 	execE := &Entry{
// 		Name:  "exec",
// 		ObjId: "test",
// 		State: s2,
// 	}

// 	assert.Equal(t, true, execE.isExec())

// }

func Test_Entry_ToString(t *testing.T) {
	e := &Entry{
		CTime:      1,
		CTime_nsec: 1,
		MTime:      1,
		MTime_nsec: 1,
		Dev:        2,
		Ino:        2,
		Mode:       2,
		UId:        3,
		GId:        3,
		Size:       4,
		ObjId:      "1111",
		Flags:      1,
		Path:       "test.txt",
	}

	b := []byte(e.ToString())

	assert.Equal(t, []byte{0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 3, 0, 0, 0, 4, 17, 17, 0, 1, 116, 101, 115, 116, 46, 116, 120, 116, 0, 0, 0, 0}, b)
}
