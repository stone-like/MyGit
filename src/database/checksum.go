package database

import (
	"encoding/binary"
	"io"
	"mygit/src/crypt"
)

type CheckSum struct {
	reader  io.Reader
	Content string
}

func (c *CheckSum) Read(r io.Reader, size int) ([]byte, error) {
	//io.Reader -> bytes.Bufferに読み取り、bytes.Bufferをbinaryreadで読み取って構造体へ
	bs := make([]byte, size)
	err := binary.Read(r, binary.BigEndian, bs)

	if err != nil {
		return nil, err
	}

	c.Content += string(bs)

	return bs, nil

}

func (c *CheckSum) GenerateHash() string {
	return crypt.DigestBySha1(c.Content)
}
