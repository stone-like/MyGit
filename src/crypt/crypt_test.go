package crypt

import (
	"fmt"
	"testing"
)

func Test_SHA(t *testing.T) {
	str := DigestBySha1("123")
	s := fmt.Sprint([]byte(str))

	fmt.Println(s)
}
