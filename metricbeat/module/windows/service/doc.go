// Package service implements a Metricbeat metricset for reading Windows Services
package service

//go:generate go run ../run.go -cmd "go tool cgo -godefs defs_service_windows.go" -goarch amd64 -output defs_service_windows_amd64.go
//go:generate go run ../run.go -cmd "go tool cgo -godefs defs_service_windows.go" -goarch 386 -output defs_service_windows_386.go
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zservice_windows.go service_windows.go
//go:generate gofmt -w defs_service_windows_amd64.go defs_service_windows_386.go
