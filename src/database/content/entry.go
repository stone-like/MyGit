package content

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"mygit/src/database/crypt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"
)

var (
	REGULAR_MODE    = "100644"
	EXECUTABLE_MODE = "100755"
	ENTRY_BLOCK     = 8
)

type FileState = os.FileInfo

type Entry struct {
	// Name  string
	// ObjId string
	// State FileState

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
	ObjId      string
	Flags      int
	Path       string
}

type EntryStateBin struct {
	CTime      uint32
	CTime_nsec uint32
	MTime      uint32
	MTime_nsec uint32
	Dev        uint32
	Ino        uint32
	Mode       uint32
	UId        uint32
	GId        uint32
	Size       uint32
}

type EntryFlagsBin struct {
	Flags uint16
}

type EntryFromMem struct {
	CTime      uint32
	CTime_nsec uint32
	MTime      uint32
	MTime_nsec uint32
	Dev        uint32
	Ino        uint32
	Mode       uint32
	UId        uint32
	GId        uint32
	Size       uint32
	ObjId      [20]byte
	Flags      uint16
}

func (em *EntryFromMem) ConvertToEntity(path string) *Entry {
	hexString := hex.EncodeToString(em.ObjId[:])

	return &Entry{
		CTime:      int64(em.CTime),
		CTime_nsec: int64(em.CTime_nsec),
		MTime:      int64(em.MTime),
		MTime_nsec: int64(em.MTime_nsec),
		Dev:        uint64(em.Dev),
		Ino:        uint64(em.Ino),
		Mode:       int(em.Mode),
		UId:        em.UId,
		GId:        em.GId,
		Size:       int64(em.Size),
		ObjId:      hexString,
		Flags:      int(em.Flags),
		Path:       path,
	}
}

func IsExec(mode uint32) bool {
	return mode&0111 != 0
}

//これはtree,entryとかの書き込みで使う 040000 tree 100644 hello.txtで使うのでstring
func (e *Entry) getMode() string {
	if IsExec(uint32(e.Mode)) {
		return EXECUTABLE_MODE
	} else {
		return REGULAR_MODE
	}
}

func (e *Entry) IsTree() bool {
	i, _ := strconv.ParseInt(DIRECTORY_MODE, 8, 64)
	return e.Mode == int(i)
}

func ModeToString(mode int) string {
	return fmt.Sprintf("%o", mode)
}
func (e *Entry) GetObjId() string {
	return e.ObjId
}

func (e *Entry) SetObjId(objId string) {
	e.ObjId = objId
}

func (e *Entry) ToString() string {
	var tempStr string
	buf := new(bytes.Buffer)

	eb := &EntryStateBin{
		CTime:      uint32(e.CTime),
		CTime_nsec: uint32(e.CTime_nsec),
		MTime:      uint32(e.MTime),
		MTime_nsec: uint32(e.MTime_nsec),
		Dev:        uint32(e.Dev),
		Ino:        uint32(e.Ino),
		Mode:       uint32(e.Mode),
		UId:        e.UId,
		GId:        e.GId,
		Size:       uint32(e.Size),
	}

	binary.Write(buf, binary.BigEndian, eb)

	tempStr += buf.String()
	buf.Reset()

	ret, _ := crypt.CreateH40(e.ObjId)

	tempStr += ret

	fb := &EntryFlagsBin{
		Flags: uint16(e.Flags),
	}
	binary.Write(buf, binary.BigEndian, fb)

	tempStr += buf.String()
	buf.Reset()

	tempStr += fmt.Sprintf("%s\x00", e.Path)

	return PaddingAlign8(tempStr)

}

func PaddingAlign8(str string) string {

	tempStr := str
	for {

		if len([]byte(tempStr))%ENTRY_BLOCK == 0 {
			break
		} else {
			tempStr += "\x00"
		}

	}

	return tempStr
}

var MAX_PATH_SIZE = 0xfff

func MinPathSize(path string) int {
	return int(math.Min(float64(len([]byte(path))), float64(MAX_PATH_SIZE)))
}

func CreateFlags(stage int, path string) int {
	return (stage << 12) | MinPathSize(path)
}

func CreateEntryFromDB(stage int, path string, e *Entry) *Entry {
	flags := CreateFlags(stage, path)
	return &Entry{
		Mode:  e.Mode,
		ObjId: e.ObjId,
		Flags: flags,
		Path:  path,
	}
}

func (e *Entry) GeModeForNormalAndNilEntry() int {
	if e == nil {
		return 0
	} else {
		return e.Mode
	}
}

func (e *Entry) GetObjIdForNormalAndNilEntry() string {
	if e == nil {
		return ""
	} else {
		return e.ObjId
	}
}

func (e *Entry) Type() string {
	return ""
}

func (e *Entry) Basename() string {
	return e.Path
}

func (e *Entry) GetStage() int {
	return (e.Flags >> 12) & 0x3
}

//あとで共通化
func createParentDirs(path string) []string {
	var parents []string
	dir := filepath.Dir(path)

	if dir != "." {
		ret := createParentDirs(dir)
		parents = append(parents, dir)
		parents = append(parents, ret...)
	}

	return parents

}

func (e *Entry) ParentDirs(path string) []string {
	ret := createParentDirs(path)
	sort.Slice(ret, func(i, j int) bool {
		return len(ret[i]) < len(ret[j])
	})

	return ret

}

func (e *Entry) TimeMatch(stat FileState) bool {
	s, _ := stat.Sys().(*syscall.Stat_t)
	return e.CTime == s.Ctim.Sec && e.CTime_nsec == s.Ctim.Nsec &&
		e.MTime == s.Mtim.Sec && e.MTime_nsec == s.Mtim.Nsec
}
