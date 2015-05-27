package logp

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
)

type Logger struct {
	toSyslog            bool
	toStderr            bool
	level               Priority
	selectors           map[string]bool
	debug_all_selectors bool

	logger *log.Logger
}

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

var _log Logger

func Debug(selector string, format string, v ...interface{}) {
	if _log.level >= LOG_DEBUG {
		if !_log.debug_all_selectors {
			selected := _log.selectors[selector]
			if !selected {
				return
			}
		}
		_log.logger.Output(2, fmt.Sprintf("DBG  "+format, v...))
	}
}

func IsDebug(selector string) bool {
	return _log.selectors[selector]
}

func Info(format string, v ...interface{}) {
	if _log.level >= LOG_INFO {
		_log.logger.Output(2, fmt.Sprintf("INFO "+format, v...))
	}
}

func Warn(format string, v ...interface{}) {
	if _log.level >= LOG_WARNING {
		_log.logger.Output(2, fmt.Sprintf("WARN "+format, v...))
	}
}

func Err(format string, v ...interface{}) {
	if _log.level >= LOG_ERR {
		_log.logger.Output(2, fmt.Sprintf("ERR  "+format, v...))
	}
}

func Critical(format string, v ...interface{}) {
	if _log.level >= LOG_CRIT {
		_log.logger.Output(2, fmt.Sprintf("CRIT "+format, v...))
	}
}

func WTF(format string, v ...interface{}) {
	if _log.level >= LOG_CRIT {
		_log.logger.Output(2, fmt.Sprintf("CRIT "+format, v...))
	}

	// TODO: assert here when not in production mode
}

func Recover(msg string) {
	if r := recover(); r != nil {
		Err("%s. Recovering, but please report this: %s.", msg, r)
		Err("Stacktrace: %s", debug.Stack())
	}
}

func LogInit(level Priority, prefix string, toSyslog bool, toStderr bool, debugSelectors []string) {
	_log.level = level

	_log.selectors = make(map[string]bool)
	for _, selector := range debugSelectors {
		_log.selectors[selector] = true
		if selector == "*" {
			_log.debug_all_selectors = true
		}
	}

	_log.logger = log.New(os.Stdout, prefix, log.Lshortfile)
}

func SetToStderr(toStderr bool) {
}
