/*
Package perfmon collect windows performance counters.
*/
package perfmon

//go:generate go run run.go -cmd "go tool cgo -godefs defs_windows.go" -goarch amd64 -output defs_windows_amd64.go
//go:generate go run run.go -cmd "go tool cgo -godefs defs_windows.go" -goarch 386   -output defs_windows_386.go
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output syscall_windows.go pdh_windows.go
//go:generate gofmt -w defs_windows_amd64.go defs_windows_386.go syscall_windows.go
