package errors

import "errors"

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

type InternalError interface {
	Cause() string
}
