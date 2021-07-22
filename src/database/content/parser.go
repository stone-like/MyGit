package content

import (
	"errors"
	"io"
)

const (
	BLOB   = "blob"
	TREE   = "tree"
	COMMIT = "commit"
)

var ErrorUndefinedGitObjectType = errors.New("undefinedGitObjectType")

type ParsedObj interface {
	Parse(r io.Reader) error
	GetObjId() string
	SetObjId(objId string)
	Type() string
}

func Parse(objType string, r io.Reader) (ParsedObj, error) {

	switch objType {
	case BLOB:
		return ParseBlob(r)
	case TREE:
		return ParseTree(r)
	case COMMIT:
		return ParseCommit(r)
	default:
		return nil, ErrorUndefinedGitObjectType
	}
}

func ParseBlob(r io.Reader) (ParsedObj, error) {
	b := &Blob{}
	b.Parse(r)
	return b, nil
}

func ParseTree(r io.Reader) (ParsedObj, error) {
	t := GenerateTree()
	t.Parse(r)
	return t, nil
}

func ParseCommit(r io.Reader) (ParsedObj, error) {
	c := &CommitFromMem{}
	c.Parse(r)
	return c, nil
}
