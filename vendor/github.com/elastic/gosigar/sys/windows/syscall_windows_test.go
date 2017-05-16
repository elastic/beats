package windows

import (
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetProcessImageFileName(t *testing.T) {
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(syscall.Getpid()))
	if err != nil {
		t.Fatal(err)
	}

	filename, err := GetProcessImageFileName(h)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("GetProcessImageFileName: %v", filename)
}

func TestGetProcessMemoryInfo(t *testing.T) {
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(syscall.Getpid()))
	if err != nil {
		t.Fatal(err)
	}

	counters, err := GetProcessMemoryInfo(h)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("GetProcessMemoryInfo: ProcessMemoryCountersEx=%+v", counters)
}

func TestGetLogicalDriveStrings(t *testing.T) {
	drives, err := GetLogicalDriveStrings()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("GetLogicalDriveStrings: %v", drives)
}

func TestGetDriveType(t *testing.T) {
	drives, err := GetLogicalDriveStrings()
	if err != nil {
		t.Fatal(err)
	}

	for _, drive := range drives {
		dt, err := GetDriveType(drive)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("GetDriveType: drive=%v, type=%v", drive, dt)
	}
}

func TestGetSystemTimes(t *testing.T) {
	idle, kernel, user, err := GetSystemTimes()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("GetSystemTimes: idle=%v, kernel=%v, user=%v", idle, kernel, user)
}

func TestGlobalMemoryStatusEx(t *testing.T) {
	mem, err := GlobalMemoryStatusEx()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("GlobalMemoryStatusEx: %+v", mem)
}

func TestEnumProcesses(t *testing.T) {
	pids, err := EnumProcesses()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("EnumProcesses: %v", pids)
}

func TestGetDiskFreeSpaceEx(t *testing.T) {
	drives, err := GetLogicalDriveStrings()
	if err != nil {
		t.Fatal(err)
	}

	for _, drive := range drives {
		dt, err := GetDriveType(drive)
		if err != nil {
			t.Fatal(err)
		}

		// Ignore CDROM drives. They return an error if the drive is emtpy.
		if dt != DRIVE_CDROM {
			free, total, totalFree, err := GetDiskFreeSpaceEx(drive)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("GetDiskFreeSpaceEx: %v, %v, %v", free, total, totalFree)
		}
	}
}

func TestGetWindowsVersion(t *testing.T) {
	ver := GetWindowsVersion()
	assert.True(t, ver.Major >= 5)
	t.Logf("GetWindowsVersion: %+v", ver)
}

func TestCreateToolhelp32Snapshot(t *testing.T) {
	handle, err := CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	// Iterate over the snapshots until our PID is found.
	pid := uint32(syscall.Getpid())
	for {
		process, err := Process32Next(handle)
		if errors.Cause(err) == syscall.ERROR_NO_MORE_FILES {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("CreateToolhelp32Snapshot: ProcessEntry32=%v", process)

		if process.ProcessID == pid {
			assert.EqualValues(t, syscall.Getppid(), process.ParentProcessID)
			return
		}
	}

	assert.Fail(t, "Snapshot not found for PID=%v", pid)
}

func TestNtQuerySystemProcessorPerformanceInformation(t *testing.T) {
	cpus, err := NtQuerySystemProcessorPerformanceInformation()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cpus, runtime.NumCPU())

	for i, cpu := range cpus {
		assert.NotZero(t, cpu.IdleTime)
		assert.NotZero(t, cpu.KernelTime)
		assert.NotZero(t, cpu.UserTime)

		t.Logf("CPU=%v SystemProcessorPerformanceInformation=%v", i, cpu)
	}
}

func TestNtQueryProcessBasicInformation(t *testing.T) {
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(syscall.Getpid()))
	if err != nil {
		t.Fatal(err)
	}

	info, err := NtQueryProcessBasicInformation(h)
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, os.Getpid(), info.UniqueProcessID)
	assert.EqualValues(t, os.Getppid(), info.InheritedFromUniqueProcessID)

	t.Logf("NtQueryProcessBasicInformation: %+v", info)
}
