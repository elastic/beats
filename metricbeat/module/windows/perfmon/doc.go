// Package perfmon implements a Metricbeat metricset for reading Windows
// performance counters.
package perfmon

//go:generate go run mkpdh_defs.go
//go:generate go run ../run.go -cmd "go tool cgo -godefs defs_pdh_windows.go" -goarch amd64 -output defs_pdh_windows_amd64.go
//go:generate go run ../run.go -cmd "go tool cgo -godefs defs_pdh_windows.go" -goarch 386 -output defs_pdh_windows_386.go
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zpdh_windows.go pdh_windows.go
//go:generate gofmt -w defs_pdh_windows_amd64.go defs_pdh_windows_386.go zpdh_windows.go
