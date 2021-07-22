package errors

type UserError interface {
	UCause() string
}

type PathNotExists struct {
	Path  string
	Cause string
}

func (p *PathNotExists) UCause() string {
	return p.Cause
}

func (p *PathNotExists) Error() string {
	return "pathNotExists"
}

type InternalError interface {
	Cause() string
}
