// +build windows

package logp

import "log"

func openSyslog(level Priority, prefix string) *log.Logger {
	return nil
}
