package content

type Object interface {
	ToString() string
	Type() string
	GetObjId() string
	SetObjId(id string)
	Basename() string
	getMode() string
}

type TreeWriter interface {
	Mode() string
}
