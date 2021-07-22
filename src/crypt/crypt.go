package crypt

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
)

func HexDigestBySha1(str string) string {
	sha1 := sha1.New()
	io.WriteString(sha1, str)
	return hex.EncodeToString(sha1.Sum(nil))
}

func DigestBySha1(str string) string {
	sha1 := sha1.New()
	io.WriteString(sha1, str)
	return string(sha1.Sum(nil))
}
