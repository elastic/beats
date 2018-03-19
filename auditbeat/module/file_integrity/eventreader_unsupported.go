// +build !linux,!freebsd,!openbsd,!netbsd,!windows,!darwin

package file_integrity

import "github.com/pkg/errors"

func NewEventReader(c Config) (EventProducer, error) {
	return errors.New("file auditing metricset is not implemented on this system")
}
