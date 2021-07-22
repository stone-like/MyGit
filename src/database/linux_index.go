// +build !windows

package database

import (
	"math"
	con "mygit/src/database/content"
	"syscall"
)

func CreateIndex(path, objId string, state con.FileState) *con.Entry {

	flags := int(math.Min(float64(len([]byte(path))), float64(MAX_PATH_SIZE)))

	stat, _ := state.Sys().(*syscall.Stat_t)

	return &con.Entry{
		CTime:      stat.Ctim.Sec,
		CTime_nsec: stat.Ctim.Nsec,
		MTime:      stat.Mtim.Sec,
		MTime_nsec: stat.Mtim.Nsec,
		Dev:        stat.Dev,
		Ino:        stat.Ino,
		Mode:       ModeForStat(state),
		UId:        stat.Uid,
		GId:        stat.Gid,
		Size:       stat.Size,
		ObjId:      objId,
		Flags:      flags,
		Path:       path,
	}

}

func UpdateStat(e *con.Entry, state con.FileState) {
	stat, _ := state.Sys().(*syscall.Stat_t)
	e.CTime = stat.Ctim.Sec
	e.CTime_nsec = stat.Ctim.Nsec
	e.MTime = stat.Mtim.Sec
	e.MTime_nsec = stat.Mtim.Nsec
	e.Dev = stat.Dev
	e.Ino = stat.Ino
	e.Mode = ModeForStat(state)
	e.UId = stat.Uid
	e.GId = stat.Gid
	e.Size = stat.Size
}
