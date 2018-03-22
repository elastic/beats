package process

import (
	"os"
	"path"
	"strconv"

	"github.com/elastic/gosigar"
)

// GetSelfPid returns the PID for this process
func GetSelfPid() (int, error) {
	pid, err := os.Readlink(path.Join(gosigar.Procd, "self"))

	if err != nil {
		return 0, err
	}

	return strconv.Atoi(pid)
}
