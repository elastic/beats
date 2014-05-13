package main

import (
    "fmt"
    "log"
    "os"
    "runtime/debug"
)

type Logger struct {
    toSyslog  bool
    level     Priority
    selectors map[string]bool

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

func DEBUG(selector string, format string, v ...interface{}) {
    if _log.level >= LOG_DEBUG {
        selected := _log.selectors[selector]
        if !selected {
            return
        }
        _log.logger.Output(2, fmt.Sprintf("DBG  "+format, v...))
    }
}

func IS_DEBUG(selector string) bool {
    return _log.selectors[selector]
}

func INFO(format string, v ...interface{}) {
    if _log.level >= LOG_INFO {
        _log.logger.Output(2, fmt.Sprintf("INFO "+format, v...))
    }
}

func WARN(format string, v ...interface{}) {
    if _log.level >= LOG_WARNING {
        _log.logger.Output(2, fmt.Sprintf("WARN "+format, v...))
    }
}

func ERR(format string, v ...interface{}) {
    if _log.level >= LOG_ERR {
        _log.logger.Output(2, fmt.Sprintf("ERR  "+format, v...))
    }
}

func CRIT(format string, v ...interface{}) {
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

func RECOVER(msg string) {
    if r := recover(); r != nil {
        ERR("%s. Recovering, but please report this: %s.", msg, r)
        ERR("Stacktrace: %s", debug.Stack())
    }
}

func LogInit(level Priority, prefix string, toSyslog bool, debugSelectors []string) {
    _log.level = level

    _log.selectors = make(map[string]bool)
    for _, selector := range debugSelectors {
        _log.selectors[selector] = true
    }

    _log.logger = log.New(os.Stdout, prefix, log.Lshortfile)
}
