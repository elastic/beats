// +build !windows

package main

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
    level               syslog.Priority
    selectors           map[string]bool
    debug_all_selectors bool

    logger *log.Logger
    syslog [syslog.LOG_DEBUG + 1]*log.Logger
}

var _log Logger

func DEBUG(selector string, format string, v ...interface{}) {
    if _log.level >= syslog.LOG_DEBUG {
        if !_log.debug_all_selectors {
            selected := _log.selectors[selector]
            if !selected {
                return
            }
        }
        if _log.toSyslog {
            _log.syslog[syslog.LOG_INFO].Output(2, fmt.Sprintf(format, v...))
        } else {
            _log.logger.Output(2, fmt.Sprintf("DBG  "+format, v...))
        }
    }
}

func IS_DEBUG(selector string) bool {
    return _log.selectors[selector]
}

func INFO(format string, v ...interface{}) {
    if _log.level >= syslog.LOG_INFO {
        if _log.toSyslog {
            _log.syslog[syslog.LOG_INFO].Output(2, fmt.Sprintf(format, v...))
        } else {
            _log.logger.Output(2, fmt.Sprintf("INFO "+format, v...))
        }
    }
}

func WARN(format string, v ...interface{}) {
    if _log.level >= syslog.LOG_WARNING {
        if _log.toSyslog {
            _log.syslog[syslog.LOG_WARNING].Output(2, fmt.Sprintf(format, v...))
        } else {
            _log.logger.Output(2, fmt.Sprintf("WARN "+format, v...))
        }
    }
}

func ERR(format string, v ...interface{}) {
    if _log.level >= syslog.LOG_ERR {
        if _log.toSyslog {
            _log.syslog[syslog.LOG_ERR].Output(2, fmt.Sprintf(format, v...))
        } else {
            _log.logger.Output(2, fmt.Sprintf("ERR  "+format, v...))
        }
    }
}

func CRIT(format string, v ...interface{}) {
    if _log.level >= syslog.LOG_CRIT {
        if _log.toSyslog {
            _log.syslog[syslog.LOG_CRIT].Output(2, fmt.Sprintf(format, v...))
        } else {
            _log.logger.Output(2, fmt.Sprintf("CRIT "+format, v...))
        }
    }
}

func WTF(format string, v ...interface{}) {
    if _log.level >= syslog.LOG_CRIT {
        if _log.toSyslog {
            _log.syslog[syslog.LOG_CRIT].Output(2, fmt.Sprintf(format, v...))
        } else {
            _log.logger.Output(2, fmt.Sprintf("CRIT "+format, v...))
        }
    }

    // TODO: assert here when not in production mode
}

func RECOVER(msg string) {
    if r := recover(); r != nil {
        ERR("%s. Recovering, but please report this: %s.", msg, r)
        ERR("Stacktrace: %s", debug.Stack())
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

func LogInit(level Priority, prefix string, toSyslog bool, debugSelectors []string) {
    _log.toSyslog = toSyslog
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
    } else {
        _log.logger = log.New(os.Stdout, prefix, log.Lshortfile)
    }
}
