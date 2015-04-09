// +build !windows

package logp

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"runtime/debug"
)

type Priority int

const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

type Logger struct {
	toSyslog            bool
	toStderr            bool
	level               syslog.Priority
	selectors           map[string]bool
	debug_all_selectors bool

	logger *log.Logger
	syslog [syslog.LOG_DEBUG + 1]*log.Logger
}

var _log Logger

func Debug(selector string, format string, v ...interface{}) {
	if _log.level >= syslog.LOG_DEBUG {
		if !_log.debug_all_selectors {
			selected := _log.selectors[selector]
			if !selected {
				return
			}
		}
		if _log.toSyslog {
			_log.syslog[syslog.LOG_INFO].Output(2, fmt.Sprintf(format, v...))
		}
		if _log.toStderr {
			_log.logger.Output(2, fmt.Sprintf("DBG  "+format, v...))
		}
	}
}

func IsDebug(selector string) bool {
	return _log.selectors[selector]
}

func Info(format string, v ...interface{}) {
	if _log.level >= syslog.LOG_INFO {
		if _log.toSyslog {
			_log.syslog[syslog.LOG_INFO].Output(2, fmt.Sprintf(format, v...))
		}
		if _log.toStderr {
			_log.logger.Output(2, fmt.Sprintf("INFO "+format, v...))
		}
	}
}

func Warn(format string, v ...interface{}) {
	if _log.level >= syslog.LOG_WARNING {
		if _log.toSyslog {
			_log.syslog[syslog.LOG_WARNING].Output(2, fmt.Sprintf(format, v...))
		}
		if _log.toStderr {
			_log.logger.Output(2, fmt.Sprintf("WARN "+format, v...))
		}
	}
}

func Err(format string, v ...interface{}) {
	if _log.level >= syslog.LOG_ERR {
		if _log.toSyslog {
			_log.syslog[syslog.LOG_ERR].Output(2, fmt.Sprintf(format, v...))
		}
		if _log.toStderr {
			_log.logger.Output(2, fmt.Sprintf("ERR  "+format, v...))
		}
	}
}

func Critical(format string, v ...interface{}) {
	if _log.level >= syslog.LOG_CRIT {
		if _log.toSyslog {
			_log.syslog[syslog.LOG_CRIT].Output(2, fmt.Sprintf(format, v...))
		}
		if _log.toStderr {
			_log.logger.Output(2, fmt.Sprintf("CRIT "+format, v...))
		}
	}
}

func WTF(format string, v ...interface{}) {
	if _log.level >= syslog.LOG_CRIT {
		if _log.toSyslog {
			_log.syslog[syslog.LOG_CRIT].Output(2, fmt.Sprintf(format, v...))
		}
		if _log.toStderr {
			_log.logger.Output(2, fmt.Sprintf("CRIT "+format, v...))
		}
	}

	// TODO: assert here when not in production mode
}

func Recover(msg string) {
	if r := recover(); r != nil {
		Err("%s. Recovering, but please report this: %s.", msg, r)
		Err("Stacktrace: %s", debug.Stack())
	}
}

func openSyslog(level syslog.Priority, prefix string) *log.Logger {
	logger, err := syslog.NewLogger(level, log.Lshortfile)
	if err != nil {
		fmt.Println("Error opening syslog: ", err)
		return nil
	}
	logger.SetPrefix(prefix)

	return logger
}

func LogInit(level Priority, prefix string, toSyslog bool, toStderr bool, debugSelectors []string) {
	_log.toSyslog = toSyslog
	_log.toStderr = toStderr
	_log.level = syslog.Priority(level)

	_log.selectors = make(map[string]bool)
	for _, selector := range debugSelectors {
		_log.selectors[selector] = true
		if selector == "*" {
			_log.debug_all_selectors = true
		}
	}

	if _log.toSyslog {
		for prio := syslog.LOG_EMERG; prio <= syslog.LOG_DEBUG; prio++ {
			_log.syslog[prio] = openSyslog(prio, prefix)
		}
	}
	if _log.toStderr {
		_log.logger = log.New(os.Stdout, prefix, log.Lshortfile)
	}
}

func SetToStderr(toStderr bool) {
	_log.toStderr = toStderr
}
