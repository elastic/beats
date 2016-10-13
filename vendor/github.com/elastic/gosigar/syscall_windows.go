package gosigar

// Process-specific access rights. Others are declared in the syscall package.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684880(v=vs.85).aspx
const (
	PROCESS_QUERY_LIMITED_INFORMATION uint32 = 0x1000
	PROCESS_VM_READ                   uint32 = 0x0010
)

// Use "GOOS=windows go generate -v -x ." to generate the source.

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys   _GetSystemTimes(idleTime *syscall.Filetime, kernelTime *syscall.Filetime, userTime *syscall.Filetime) (err error) = kernel32.GetSystemTimes
