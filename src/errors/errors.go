package errors

import (
	"errors"
	"fmt"
	"io"

	ers "github.com/pkg/errors"
)

type ParseError interface {
	ParseMessage() string
}

type InvalidObjectError struct {
	Message      string
	Hint         []string
	CriticalInfo string
}

func (i *InvalidObjectError) ParseMessage() string {
	return "aaa"
}

func (i *InvalidObjectError) Error() string {
	return "invalidObjError"
}

type ConvertError interface {
	ConvertMessage() string
}
type ObjConvertionError struct {
	Type         string
	Message      string
	CriticalInfo string
}

func (o *ObjConvertionError) ConvertMessage() string {
	return o.Type
}

func (o *ObjConvertionError) Error() string {
	return "objConvertionError"
}

var (
	ErrorFileNonExists = errors.New("file not exists")
)

type UserError interface {
	UserCause() string
}

type ConflictOccurError struct {
	ConflictDetail string
}

func (c *ConflictOccurError) UserCause() string {
	return c.ConflictDetail
}

func (c *ConflictOccurError) Error() string {
	return "conflictOccurError"
}

func (c *ConflictOccurError) GetContent() string {
	return c.ConflictDetail
}

type InvalidIndexPathOnRemovalError struct {
	Message string
}

func (i *InvalidIndexPathOnRemovalError) UserCause() string {
	return i.Message
}

func (i *InvalidIndexPathOnRemovalError) Error() string {
	return "invalidIndexPathOnRemovalError"
}

func (i *InvalidIndexPathOnRemovalError) GetContent() string {
	return i.Message
}

type FileNotExistOnConflictError struct {
	Message string
}

func (f *FileNotExistOnConflictError) UserCause() string {
	return f.Message
}

func (f *FileNotExistOnConflictError) Error() string {
	return "FileNotExistOnConflictError"
}

func (f *FileNotExistOnConflictError) GetContent() string {
	return f.Message
}

type InternalError interface {
	Cause() string
}

type InvalidFormatError struct {
	FormatName string
}

func (i *InvalidFormatError) UserCause() string {
	return fmt.Sprintf("%s is invalid format name\n", i.FormatName)
}

func (i *InvalidFormatError) Error() string {
	return fmt.Sprintf("%s is invalid format name\n", i.FormatName)
}

type WillWriteError interface {
	GetContent() string
}

func HandleWillWriteError(err error, w io.Writer) error {
	//WillWriteErrorだったらwriteに書いて終わり、それ以上errorは伝播させず
	//nilとする

	willWrite, ok := ers.Cause(err).(WillWriteError)
	if !ok {
		return err
	}

	w.Write([]byte(willWrite.GetContent()))

	return nil

}
