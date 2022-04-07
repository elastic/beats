// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mqtt

import (
	"sync"

	libmqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/elastic/beats/v8/libbeat/logp"
)

var setupLoggingOnce sync.Once

type loggerWrapper struct {
	log *logp.Logger
}

type (
	debugLogger loggerWrapper
	errorLogger loggerWrapper
	warnLogger  loggerWrapper
)

var (
	_ libmqtt.Logger = new(debugLogger)
	_ libmqtt.Logger = new(errorLogger)
	_ libmqtt.Logger = new(warnLogger)
)

func setupLibraryLogging() {
	setupLoggingOnce.Do(func() {
		logger := logp.NewLogger("libmqtt", zap.AddCallerSkip(1))
		libmqtt.CRITICAL = &errorLogger{log: logger}
		libmqtt.DEBUG = &debugLogger{log: logger}
		libmqtt.ERROR = &errorLogger{log: logger}
		libmqtt.WARN = &warnLogger{log: logger}
	})
}

func (l *debugLogger) Println(v ...interface{}) {
	l.log.Debug(v...)
}

func (l *debugLogger) Printf(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

func (l *errorLogger) Println(v ...interface{}) {
	l.log.Error(v...)
}

func (l *errorLogger) Printf(format string, v ...interface{}) {
	l.log.Errorf(format, v...)
}

func (l *warnLogger) Println(v ...interface{}) {
	l.log.Warn(v...)
}

func (l *warnLogger) Printf(format string, v ...interface{}) {
	l.log.Warnf(format, v...)
}
