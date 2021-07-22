package src

import (
	"io/ioutil"
	"mygit/src/database/lock"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func createPath(path string) error {
// 	if _, err := os.Stat(path); err != nil {
// 		f, err := os.Create(path)
// 		defer f.Close()
// 		if err != nil {
// 			return err
// 		}

// 		return nil
// 	}

// 	return nil

// }

func Test_Flock(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir1)
	})

	// err = createPath(filepath.Join(tempDir1, "ok.txt"))
	// assert.NoError(t, err)

	okPath := filepath.Join(tempDir1, "ok.txt")

	f, _ := os.OpenFile(okPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	defer f.Close()
	// lock.Flock(okPath, func() {
	// 	f.Write([]byte("test1"))
	// })
	l := lock.NewFileLock(okPath)
	l.Lock()
	f.Write([]byte("test1"))
	defer l.Unlock()

	by, err := ioutil.ReadFile(okPath)
	assert.NoError(t, err)
	assert.Equal(t, "test1", string(by))
}

func Test_Flock_CREATE_NON_EXISTS_PATH(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)
	tempDir1, err := ioutil.TempDir(curDir, "tempDir")
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir1)
	})

	// err = createPath(filepath.Join(tempDir1, "ok.txt"))
	// assert.NoError(t, err)

	okPath := filepath.Join(tempDir1, "ok.txt")

	l := lock.NewFileLock(okPath)
	l.Lock()

	defer l.Unlock()

}
