// +build !windows

package wineventlog

import (
	"fmt"

	"github.com/elastic/beats/winlogbeat/eventlog"
)

func IsAvailable() (bool, error) {
	return false, fmt.Errorf("Windows Event Log is only available on Windows.")
}

func Channels() ([]string, error) {
	return nil, nil
}

func (l *eventLog) Open(recordNumber uint64) error {
	return nil
}

func (l *eventLog) Read() ([]eventlog.LogRecord, error) {
	return nil, nil
}

func (l *eventLog) Close() error {
	return nil
}
