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
	toFile              bool
	level               syslog.Priority
	selectors           map[string]bool
	debug_all_selectors bool

	logger  *log.Logger
	syslog  [syslog.LOG_DEBUG + 1]*log.Logger
	rotator *FileRotator
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
		if _log.toFile {
			_log.rotator.WriteLine([]byte(fmt.Sprintf("DBG  "+format+"\n", v...)))
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
		if _log.toFile {
			_log.rotator.WriteLine([]byte(fmt.Sprintf("INFO  "+format+"\n", v...)))
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
		if _log.toFile {
			_log.rotator.WriteLine([]byte(fmt.Sprintf("WARN  "+format+"\n", v...)))
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
		if _log.toFile {
			_log.rotator.WriteLine([]byte(fmt.Sprintf("ERR  "+format+"\n", v...)))
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
		if _log.toFile {
			_log.rotator.WriteLine([]byte(fmt.Sprintf("CRIT  "+format+"\n", v...)))
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
		if _log.toFile {
			_log.rotator.WriteLine([]byte(fmt.Sprintf("CRIT  "+format+"\n", v...)))
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

// TODO: remove toSyslog and toStderr from the init function
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

func SetToStderr(toStderr bool, prefix string) {
	_log.toStderr = toStderr
	if _log.toStderr {
		_log.logger = log.New(os.Stdout, prefix, log.Lshortfile)
	}
}

func SetToSyslog(toSyslog bool, prefix string) {
	_log.toSyslog = toSyslog
	if _log.toSyslog {
		for prio := syslog.LOG_EMERG; prio <= syslog.LOG_DEBUG; prio++ {
			_log.syslog[prio] = openSyslog(prio, prefix)
		}
	}
}

func SetToFile(toFile bool, rotator *FileRotator) error {
	_log.toFile = toFile
	if _log.toFile {
		_log.rotator = rotator

		err := rotator.CreateDirectory()
		if err != nil {
			return err
		}
		err = rotator.CheckIfConfigSane()
		if err != nil {
			return err
		}
	}
	return nil
}
