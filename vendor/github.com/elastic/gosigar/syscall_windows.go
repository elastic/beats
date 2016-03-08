package gosigar

// Use "GOOS=windows go generate -v -x ." to generate the source.

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys   _GetSystemTimes(idleTime *syscall.Filetime, kernelTime *syscall.Filetime, userTime *syscall.Filetime) (err error) = kernel32.GetSystemTimes
