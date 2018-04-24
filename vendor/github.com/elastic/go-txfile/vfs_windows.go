package txfile

import (
	"fmt"
	"os"
	"reflect"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/theckman/go-flock"
)

type osFileState struct {
	mmapHandle windows.Handle
	lock       *flock.Flock
}

const (
	lockExt = ".lock"
)

func (f *osFile) MMap(sz int) ([]byte, error) {
	szhi, szlo := uint32(sz>>32), uint32(sz)
	hdl, err := windows.CreateFileMapping(windows.Handle(f.Fd()), nil, windows.PAGE_READONLY, szhi, szlo, nil)
	if hdl == 0 {
		return nil, os.NewSyscallError("CreateFileMapping", err)
	}

	// map memory
	addr, err := windows.MapViewOfFile(hdl, windows.FILE_MAP_READ, 0, 0, uintptr(sz))
	if addr == 0 {
		windows.CloseHandle(hdl)
		return nil, os.NewSyscallError("MapViewOfFile", err)
	}

	f.state.mmapHandle = hdl

	slice := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(addr),
		Len:  sz,
		Cap:  sz}))
	return slice, nil
}

func (f *osFile) MUnmap(b []byte) error {
	err1 := windows.UnmapViewOfFile(uintptr(unsafe.Pointer(&b[0])))
	b = nil

	err2 := windows.CloseHandle(f.state.mmapHandle)
	f.state.mmapHandle = 0

	if err1 != nil {
		return os.NewSyscallError("UnmapViewOfFile", err1)
	} else if err2 != nil {
		return os.NewSyscallError("CloseHandle", err2)
	}
	return nil
}

func (f *osFile) Lock(exclusive, blocking bool) error {
	if f.state.lock != nil {
		return fmt.Errorf("file %v is already locked", f.Name())
	}

	var ok bool
	var err error
	lock := flock.NewFlock(f.Name() + lockExt)
	if blocking {
		err = lock.Lock()
		ok = err != nil
	} else {
		ok, err = lock.TryLock()
	}

	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("file %v can not be locked right now", f.Name())
	}

	f.state.lock = lock
	return nil
}

func (f *osFile) Unlock() error {
	if f.state.lock == nil {
		return fmt.Errorf("file %v is not locked", f.Name())
	}

	err := f.state.lock.Unlock()
	if err == nil {
		f.state.lock = nil
	}

	return err
}
