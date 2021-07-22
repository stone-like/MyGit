package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	curDir, err := os.Getwd()
	assert.NoError(t, err)

	path := filepath.Join(curDir, "a")
	dir, file := filepath.Split(path)
	_, err = ioutil.TempFile(dir, file)
	// defer func() {
	// 	if err != nil {
	// 		// Don't leave the temp file lying around on error.
	// 		_ = os.Remove(temp.Name()) // yes, ignore the error, not much we can do about it.
	// 	}
	// }()
}
