package crypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_H40(t *testing.T) {
	ret, _ := CreateH40("1111")

	assert.Equal(t, []byte{17, 17}, []byte(ret))
}
