// Copyright (c) 2012 VMware, Inc.

package sigar

// #include <stdlib.h>
// #include <windows.h>
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var (
	modpsapi    = syscall.NewLazyDLL("psapi.dll")
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")

	procEnumProcesses           = modpsapi.NewProc("EnumProcesses")
	procGetProcessMemoryInfo    = modpsapi.NewProc("GetProcessMemoryInfo")
	procGetProcessTimes         = modkernel32.NewProc("GetProcessTimes")
	procGetProcessImageFileName = modpsapi.NewProc("GetProcessImageFileNameA")
)

const (
	PROCESS_ALL_ACCESS = 0x001f0fff
	MAX_PATH           = 260
)

type PROCESS_MEMORY_COUNTERS_EX struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

func init() {
}

func (self *LoadAverage) Get() error {
	return nil
}

func (self *Uptime) Get() error {
	return nil
}

func (self *Mem) Get() error {
	var statex C.MEMORYSTATUSEX
	statex.dwLength = C.DWORD(unsafe.Sizeof(statex))

	succeeded := C.GlobalMemoryStatusEx(&statex)
	if succeeded == C.FALSE {
		lastError := C.GetLastError()
		return fmt.Errorf("GlobalMemoryStatusEx failed with error: %d", int(lastError))
	}

	self.Total = uint64(statex.ullTotalPhys)
	self.Free = uint64(statex.ullAvailPhys)
	self.Used = self.Total - self.Free
	vtotal := uint64(statex.ullTotalVirtual)
	self.ActualFree = uint64(statex.ullAvailVirtual)
	self.ActualUsed = vtotal - self.ActualFree

	return nil
}

func (self *Swap) Get() error {
	//return notImplemented()
	return nil
}

func (self *Cpu) Get() error {

	var lpIdleTime, lpKernelTime, lpUserTime C.FILETIME

	succeeded := C.GetSystemTimes(&lpIdleTime, &lpKernelTime, &lpUserTime)
	if succeeded == C.FALSE {
		lastError := C.GetLastError()
		return fmt.Errorf("GetSystemTime failed with error: %d", int(lastError))
	}

	LOT := float64(0.0000001)
	HIT := (LOT * 4294967296.0)

	idle := ((HIT * float64(lpIdleTime.dwHighDateTime)) + (LOT * float64(lpIdleTime.dwLowDateTime)))
	user := ((HIT * float64(lpUserTime.dwHighDateTime)) + (LOT * float64(lpUserTime.dwLowDateTime)))
	kernel := ((HIT * float64(lpKernelTime.dwHighDateTime)) + (LOT * float64(lpKernelTime.dwLowDateTime)))
	system := (kernel - idle)

	self.Idle = uint64(idle)
	self.User = uint64(user)
	self.Sys = uint64(system)
	return nil
}

func (self *CpuList) Get() error {
	return notImplemented()
}

func (self *FileSystemList) Get() error {
	return notImplemented()
}

// Retrieves the process identifier for each process object in the system.

func (self *ProcList) Get() error {

	var enumSize int
	var pids [1024]C.DWORD

	// If the function succeeds, the return value is nonzero.
	ret, _, _ := procEnumProcesses.Call(
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(unsafe.Sizeof(pids)),
		uintptr(unsafe.Pointer(&enumSize)),
	)
	if ret == 0 {
		return fmt.Errorf("error %d while reading processes", C.GetLastError())
	}

	results := []int{}

	pids_size := enumSize / int(unsafe.Sizeof(pids[0]))

	for _, pid := range pids[:pids_size] {
		results = append(results, int(pid))
	}

	self.List = results

	return nil
}

func FiletimeToDuration(ft *syscall.Filetime) time.Duration {
	n := int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime) // in 100-nanosecond intervals
	return time.Duration(n*100) * time.Nanosecond
}

func CarrayToString(c [MAX_PATH]byte) string {
	end := 0
	for {
		if c[end] == 0 {
			break
		}
		end++
	}
	return string(c[:end])
}

func (self *ProcState) Get(pid int) error {

	var err error

	self.Name, err = GetProcName(pid)
	if err != nil {
		return err
	}

	self.State, err = GetProcStatus(pid)
	if err != nil {
		return err
	}

	// TODO: ppid
	return nil
}

func GetProcName(pid int) (string, error) {

	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))

	defer syscall.CloseHandle(handle)

	if err != nil {
		return "", fmt.Errorf("OpenProcess fails with %v", err)
	}

	var nameProc [MAX_PATH]byte

	ret, _, _ := procGetProcessImageFileName.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&nameProc)),
		uintptr(MAX_PATH),
	)
	if ret == 0 {
		return "", fmt.Errorf("error %d while getting process name", C.GetLastError())
	}

	return filepath.Base(CarrayToString(nameProc)), nil

}

func GetProcStatus(pid int) (RunState, error) {

	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))

	defer syscall.CloseHandle(handle)

	if err != nil {
		return RunStateUnknown, fmt.Errorf("OpenProcess fails with %v", err)
	}

	var ec uint32
	e := syscall.GetExitCodeProcess(syscall.Handle(handle), &ec)
	if e != nil {
		return RunStateUnknown, os.NewSyscallError("GetExitCodeProcess", e)
	}
	if ec == 259 { //still active
		return RunStateRun, nil
	}
	return RunStateSleep, nil
}

func (self *ProcMem) Get(pid int) error {
	handle, err := syscall.OpenProcess(PROCESS_ALL_ACCESS, false, uint32(pid))

	defer syscall.CloseHandle(handle)

	if err != nil {
		return fmt.Errorf("OpenProcess fails with %v", err)
	}

	var mem PROCESS_MEMORY_COUNTERS_EX
	mem.CB = uint32(unsafe.Sizeof(mem))

	r1, _, e1 := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&mem)),
		uintptr(mem.CB),
	)
	if r1 == 0 {
		if e1 != nil {
			return error(e1)
		} else {
			return syscall.EINVAL
		}
	}

	self.Resident = uint64(mem.WorkingSetSize)
	self.Size = uint64(mem.PrivateUsage)
	// Size contains only to the Private Bytes
	// Virtual Bytes are the Working Set plus paged Private Bytes and standby list.
	return nil
}

func (self *ProcTime) Get(pid int) error {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))

	defer syscall.CloseHandle(handle)

	if err != nil {
		return fmt.Errorf("OpenProcess fails with %v", err)

	}
	var CPU syscall.Rusage
	if err := syscall.GetProcessTimes(handle, &CPU.CreationTime, &CPU.ExitTime, &CPU.KernelTime, &CPU.UserTime); err != nil {
		return fmt.Errorf("GetProcessTimes fails with %v", err)
	}

	// convert to millis
	self.StartTime = uint64(FiletimeToDuration(&CPU.CreationTime).Nanoseconds() / 1e6)

	self.User = uint64(FiletimeToDuration(&CPU.UserTime).Nanoseconds() / 1e6)

	self.Sys = uint64(FiletimeToDuration(&CPU.KernelTime).Nanoseconds() / 1e6)

	self.Total = self.User + self.Sys

	return nil
}

func (self *ProcArgs) Get(pid int) error {
	return notImplemented()
}

func (self *ProcExe) Get(pid int) error {
	return notImplemented()
}

func (self *FileSystemUsage) Get(path string) error {
	var availableBytes C.ULARGE_INTEGER
	var totalBytes C.ULARGE_INTEGER
	var totalFreeBytes C.ULARGE_INTEGER

	pathChars := C.CString(path)
	defer C.free(unsafe.Pointer(pathChars))

	succeeded := C.GetDiskFreeSpaceEx((*C.CHAR)(pathChars), &availableBytes, &totalBytes, &totalFreeBytes)
	if succeeded == C.FALSE {
		lastError := C.GetLastError()
		return fmt.Errorf("GetDiskFreeSpaceEx failed with error: %d", int(lastError))
	}

	self.Total = *(*uint64)(unsafe.Pointer(&totalBytes))
	return nil
}

func notImplemented() error {
	panic("Not Implemented")
	return nil
}
