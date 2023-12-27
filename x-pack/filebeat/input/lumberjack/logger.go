// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"sync"

	"go.uber.org/zap"

	"github.com/elastic/elastic-agent-libs/logp"
	lumberlog "github.com/elastic/go-lumber/log"
)

var setGoLumberLoggerOnce sync.Once

func setGoLumberLogger(parent *logp.Logger) {
	setGoLumberLoggerOnce.Do(func() {
		lumberlog.Logger = &goLumberLogger{parent: parent.WithOptions(zap.AddCallerSkip(2))}
	})
}

// goLumberLogger implements the go-lumber/log.Logging interface to route
// log message from go-lumber to Beats logp.
type goLumberLogger struct {
	parent *logp.Logger
}

func (l *goLumberLogger) Printf(s string, i ...interface{}) {
	l.parent.Debugf(s, i...)
}

func (l *goLumberLogger) Println(i ...interface{}) {
	l.parent.Debug(i...)
}

func (l *goLumberLogger) Print(i ...interface{}) {
	l.parent.Debug(i...)
}
