// +build !darwin !cgo
// +build !freebsd !cgo
// +build !linux,!windows

package instance

import (
	"github.com/elastic/beats/libbeat/logp"
)

func setupMetrics(name string) error {
	logp.Warn("Metrics not implemented for this OS.")
	return nil
}
