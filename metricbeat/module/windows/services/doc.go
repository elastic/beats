// Package services implements a Metricbeat metricset for reading Windows Services
package services

//go:generate go run run.go -cmd "go tool cgo -godefs defs_services_windows.go" -goarch amd64 -output defs_services_windows_amd64.go
//go:generate go run run.go -cmd "go tool cgo -godefs defs_services_windows.go" -goarch 386 -output defs_services_windows_386.go
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zservices_windows.go services_windows.go
//go:generate gofmt -w defs_services_windows_amd64.go defs_services_windows_386.go
