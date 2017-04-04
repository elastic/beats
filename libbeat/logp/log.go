package logp

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"
)

type Priority int

const (
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
	toSyslog          bool
	toStderr          bool
	toFile            bool
	level             Priority
	selectors         map[string]struct{}
	debugAllSelectors bool

	logger  *log.Logger
	syslog  [LOG_DEBUG + 1]*log.Logger
	rotator *FileRotator
}

// pre-init logger to debug mode + stderr before init

const stderrLogFlags = log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC | log.Lshortfile

var _log = Logger{}

// TODO: remove toSyslog and toStderr from the init function
func LogInit(level Priority, prefix string, toSyslog bool, toStderr bool, debugSelectors []string) {
	_log.toSyslog = toSyslog
	_log.toStderr = toStderr
	_log.level = level

	_log.selectors, _log.debugAllSelectors = parseSelectors(debugSelectors)

	if _log.toSyslog {
		SetToSyslog(true, prefix)
	}

	if _log.toStderr {
		SetToStderr(true, prefix)
	}
}

func parseSelectors(selectors []string) (map[string]struct{}, bool) {
	all := false
	set := map[string]struct{}{}
	for _, selector := range selectors {
		set[selector] = struct{}{}
		if selector == "*" {
			all = true
		}
	}
	return set, all
}

func debugMessage(calldepth int, selector, format string, v ...interface{}) {
	if _log.level >= LOG_DEBUG && IsDebug(selector) {
		send(calldepth+1, LOG_DEBUG, "DBG  ", format, v...)
	}
}

func send(calldepth int, level Priority, prefix string, format string, v ...interface{}) {
	if _log.toSyslog {
		_log.syslog[level].Output(calldepth, fmt.Sprintf(format, v...))
	}
	if _log.toStderr {
		_log.logger.Output(calldepth, fmt.Sprintf(prefix+format, v...))
	}
	if _log.toFile {
		// Creates a timestamp for the file log message and formats it
		prefix = time.Now().Format(time.RFC3339) + " " + prefix
		_log.rotator.WriteLine([]byte(fmt.Sprintf(prefix+format, v...)))
	}
}

func Debug(selector string, format string, v ...interface{}) {
	debugMessage(3, selector, format, v...)
}

func MakeDebug(selector string) func(string, ...interface{}) {
	return func(msg string, v ...interface{}) {
		debugMessage(3, selector, msg, v...)
	}
}

func IsDebug(selector string) bool {
	return _log.debugAllSelectors || HasSelector(selector)
}

func HasSelector(selector string) bool {
	_, selected := _log.selectors[selector]
	return selected
}

func msg(level Priority, prefix string, format string, v ...interface{}) {
	if _log.level >= level {
		send(4, level, prefix, format, v...)
	}
}

func Info(format string, v ...interface{}) {
	msg(LOG_INFO, "INFO ", format, v...)
}

func Warn(format string, v ...interface{}) {
	msg(LOG_WARNING, "WARN ", format, v...)
}

func Err(format string, v ...interface{}) {
	msg(LOG_ERR, "ERR ", format, v...)
}

func Critical(format string, v ...interface{}) {
	msg(LOG_CRIT, "CRIT ", format, v...)
}

// Deprecate logs a deprecation message.
// The version string contains the version when the future will be removed
func Deprecate(version string, format string, v ...interface{}) {
	postfix := fmt.Sprintf(" Will be removed in version: %s", version)
	Warn("DEPRECATED: "+format+postfix, v...)
}

// Experimental logs the usage of an experimental feature.
func Experimental(format string, v ...interface{}) {
	Warn("EXPERIMENTAL: "+format, v...)
}

// Beta logs the usage of an beta feature.
func Beta(format string, v ...interface{}) {
	Warn("BETA: "+format, v...)
}

// WTF prints the message at CRIT level and panics immediately with the same
// message
func WTF(format string, v ...interface{}) {
	msg(LOG_CRIT, "CRIT ", format, v)
	panic(fmt.Sprintf(format, v...))
}

func Recover(msg string) {
	if r := recover(); r != nil {
		Err("%s. Recovering, but please report this: %s.", msg, r)
		Err("Stacktrace: %s", debug.Stack())
	}
}

func SetToStderr(toStderr bool, prefix string) {
	_log.toStderr = toStderr
	if _log.toStderr {
		// Add timestamp
		_log.logger = log.New(os.Stderr, prefix, stderrLogFlags)
	}
}

func SetToSyslog(toSyslog bool, prefix string) {
	_log.toSyslog = toSyslog
	if _log.toSyslog {
		for prio := LOG_EMERG; prio <= LOG_DEBUG; prio++ {
			_log.syslog[prio] = openSyslog(prio, prefix)
			if _log.syslog[prio] == nil {
				// syslog not available
				_log.toSyslog = false
				break
			}
		}
	}
}

func SetToFile(toFile bool, rotator *FileRotator) error {
	if toFile {
		err := rotator.CreateDirectory()
		if err != nil {
			return err
		}
		err = rotator.CheckIfConfigSane()
		if err != nil {
			return err
		}

		// Only assign rotator on no errors
		_log.rotator = rotator
	}

	_log.toFile = toFile

	return nil
}
