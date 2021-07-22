// +build windows

package database

import (
	"math"
	con "mygit/src/database/content"
	"syscall"
	"time"
)

func CreateIndex(path, objId string, state con.FileState) *con.Entry {

	var mode int
	if IsExec(uint32(state.Mode())) {
		mode = EXECUTABLE_MODE
	} else {
		mode = REGULAR_MODE
	}

	flags := int(math.Min(float64(len([]byte(path))), float64(MAX_PATH_SIZE)))

	stat, _ := state.Sys().(*syscall.Win32FileAttributeData)
	cTimeNano := time.Since(time.Unix(0, stat.CreationTime.Nanoseconds()))
	cTime := time.Since(time.Unix(0, stat.CreationTime.Nanoseconds()/1000000000))
	mTimeNano := time.Since(time.Unix(0, stat.LastWriteTime.Nanoseconds()))
	mTime := time.Since(time.Unix(0, stat.LastWriteTime.Nanoseconds()/1000000000))

	return &con.Entry{
		CTime:      int64(cTime),
		CTime_nsec: int64(cTimeNano),
		MTime:      int64(mTime),
		MTime_nsec: int64(mTimeNano),
		Mode:       mode,
		ObjId:      objId,
		Flags:      flags,
		Path:       path,
	}

}
