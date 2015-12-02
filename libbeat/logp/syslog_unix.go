// +build !windows,!nacl,!plan9

package logp

import (
	"fmt"
	"log"
	"log/syslog"
)

func openSyslog(level Priority, prefix string) *log.Logger {
	logger, err := syslog.NewLogger(syslog.Priority(level), log.Lshortfile)
	if err != nil {
		fmt.Println("Error opening syslog: ", err)
		return nil
	}
	logger.SetPrefix(prefix)

	return logger
}
