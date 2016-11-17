package system

import (
	"os"
	"path/filepath"

	"github.com/elastic/gosigar"
)

func initModule() {
	configureHostFS()
}

func configureHostFS() {
	dir := *HostFS
	if dir == "" {
		dir = "/"
	}

	// Set environment variables for gopsutil.
	os.Setenv("HOST_PROC", filepath.Join(dir, "/proc"))
	os.Setenv("HOST_SYS", filepath.Join(dir, "/sys"))
	os.Setenv("HOST_ETC", filepath.Join(dir, "/etc"))

	// Set proc location for gosigar.
	gosigar.Procd = filepath.Join(dir, "/proc")
}
