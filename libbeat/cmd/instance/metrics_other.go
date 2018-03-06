// +build !darwin,!linux,!cgo darwin,!cgo freebsd,!cgo openbsd

package instance

import (
	"github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/logp"
)

var (
	ephemeralID uuid.UUID
)

func init() {
	ephemeralID = uuid.NewV4()
}

func setupMetrics(name string) error {
	logp.Warn("Metrics not implemented for this OS.")
	return nil
}
