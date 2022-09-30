//go:build (darwin && cgo) || freebsd || linux || windows || aix
// +build darwin,cgo freebsd linux windows aix

package locks

import (
	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// findMatchingPID is a small wrapper to deal with cgo compat issues in libbeat's CI
func findMatchingPID(pid int) (process.PidState, error) {
	return process.GetPIDState(resolve.NewTestResolver("/"), pid)
}
