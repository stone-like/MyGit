package content

import (
	"io"
	"io/ioutil"
)

type Blob struct {
	Content string
	ObjId   string
}

func (b *Blob) Type() string {
	return "blob"
}

func (b *Blob) ToString() string {
	return b.Content
}

func (b *Blob) GetObjId() string {
	return b.ObjId
}

func (b *Blob) SetObjId(id string) {
	b.ObjId = id
}

func (b *Blob) Basename() string {
	return ""
}

func (b *Blob) getMode() string {
	return ""
}

func (b *Blob) Parse(r io.Reader) error {
	data, err := ioutil.ReadAll(r)

	if err != nil {
		return err
	}

	b.Content = string(data)
	return nil
}
