// +build !windows

package eventlog

import (
	"fmt"
)

type Handle struct {
	name string
}

func IsAvailable() (bool, error) {
	return false, fmt.Errorf("Event Logging is only available on Windows.")
}

func queryEventMessageFiles(eventLogName, sourceName string) ([]Handle, error) {
	return nil, nil
}

func freeLibrary(handle Handle) error {
	return nil
}

func (el *eventLog) Open(recordNumber uint64) error {
	return nil
}

func (el *eventLog) Read() ([]LogRecord, error) {
	return nil, nil
}

func (el *eventLog) Close() error {
	return nil
}
