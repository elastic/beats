/*
Package perfmon collect windows performance counters.
*/
package perfmon

// go:generate go tool cgo -godefs defs_windows.go > defs_windows_amd64.go
// go:generate go tool cgo -godefs defs_windows.go > defs_windows_386.go
