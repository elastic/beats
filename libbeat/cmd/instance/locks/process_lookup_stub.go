//go:build (!darwin || !cgo) && !freebsd && !linux && !windows && !aix
// +build !darwin !cgo
// +build !freebsd
// +build !linux
// +build !windows
// +build !aix

package locks

import (
	"fmt"

	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
)

func findMatchingPID(pid int) (process.PidState, error) {
	return process.Dead, fmt.Errorf("findMatchingPID not supported on platform")
}
