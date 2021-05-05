// +build freebsd linux

package metrics

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

// Get returns a metrics object for CPU data
func Get(procfs string) (MetricMap, error) {
	path := filepath.Join(procfs, "stat")
	fd, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening file %s", path)
	}

	return scanStatFile(bufio.NewScanner(fd))

}

func isCPUGlobalLine(line string) bool {
	if len(line) > 3 && line[0:3] == "cpu " {
		return true
	}
	return false
}

func isCPULine(line string) bool {
	if len(line) > 3 && line[0:3] == "cpu" && line[3] != ' ' {
		return true
	}
	return false
}

func touint(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}
