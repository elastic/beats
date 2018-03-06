// +build !darwin,!linux,!cgo darwin,!cgo freebsd,!cgo openbsd

package instance

import "github.com/elastic/beats/libbeat/logp"

func setupMetrics(name string) error {
	logp.Warn("Metrics not implemented for this OS.")
	return nil
}
