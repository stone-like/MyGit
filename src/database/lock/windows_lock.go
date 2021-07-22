// +build windows

package lock

import (
	"sync"
	"syscall"
	"unsafe"
)

//windowsだとそのファイルに限らず、他プロセスとの兼ね合いでもロックされてしまうっぽい...なのでちょっとwindows版では不具合が出てしまっている状況
var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	winLockfileFailImmediately = 0x00000001
	winLockfileExclusiveLock   = 0x00000002
	winLockfileSharedLock      = 0x00000000
)

type FileLock struct {
	m  sync.Mutex
	fd syscall.Handle
}

func NewFileLock(filename string) *FileLock {
	if filename == "" {
		panic("filename needed")
	}

	fd, err := syscall.CreateFile(
		&(syscall.StringToUTF16(filename)[0]),
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0)

	if err != nil {
		panic(err)
	}

	return &FileLock{fd: fd}
}

func (m *FileLock) Lock() {
	m.m.Lock()

	r1, _, e1 := syscall.Syscall6(
		procLockFileEx.Addr(),
		6,
		uintptr(m.fd),
		uintptr(winLockfileExclusiveLock),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&syscall.Overlapped{})))
	if r1 == 0 {
		if e1 != 0 {
			panic(error(e1))
		} else {
			panic(syscall.EINVAL)
		}
	}
}

func (m *FileLock) Unlock() {
	r1, _, e1 := syscall.Syscall6(
		procUnlockFileEx.Addr(),
		5,
		uintptr(m.fd),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&syscall.Overlapped{})),
		0)
	if r1 == 0 {
		if e1 != 0 {
			panic(error(e1))
		} else {
			panic(syscall.EINVAL)
		}
	}
	m.m.Unlock()
}
