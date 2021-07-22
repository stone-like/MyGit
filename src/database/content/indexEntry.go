package content

type IndexEntry struct {
	CTime      int64
	CTime_nsec int64
	MTime      int64
	MTime_nsec int64
	Dev        uint64
	Ino        uint64
	Mode       int
	UId        uint32
	GId        uint32
	Size       int64
	OId        string
	Flags      int
	Path       string
}

func (i *IndexEntry) isExec() bool {
	return i.Mode&0111 != 0
}

func (i *IndexEntry) getMode() string {
	if i.isExec() {
		return EXECUTABLE_MODE
	} else {
		return REGULAR_MODE
	}
}
