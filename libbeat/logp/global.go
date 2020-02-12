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

//+build !nologpglobal

package logp

import (
	"fmt"

	"go.uber.org/zap"
)

// MakeDebug returns a function that logs at debug level.
// Deprecated: Use logp.NewLogger.
func MakeDebug(selector string) func(string, ...interface{}) {
	return func(format string, v ...interface{}) {
		globalLogger().Named(selector).Debug(fmt.Sprintf(format, v...))
	}
}

// IsDebug returns true if the given selector would be logged.
// Deprecated: Use logp.NewLogger.
func IsDebug(selector string) bool {
	return globalLogger().Named(selector).Check(zap.DebugLevel, "") != nil
}

// Debug uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Debug(selector string, format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.DebugLevel) {
		log.Named(selector).Debug(fmt.Sprintf(format, v...))
	}
}

// Info uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Info(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.InfoLevel) {
		log.Info(fmt.Sprintf(format, v...))
	}
}

// Warn uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Warn(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.WarnLevel) {
		globalLogger().Warn(fmt.Sprintf(format, v...))
	}
}

// Err uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Err(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.ErrorLevel) {
		globalLogger().Error(fmt.Sprintf(format, v...))
	}
}

// Critical uses fmt.Sprintf to construct and log a message. It's an alias for
// Error.
// Deprecated: Use logp.NewLogger.
func Critical(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.ErrorLevel) {
		globalLogger().Error(fmt.Sprintf(format, v...))
	}
}

// WTF prints the message at PanicLevel and immediately panics with the same
// message.
//
// Deprecated: Use logp.NewLogger and its Panic or DPanic methods.
func WTF(format string, v ...interface{}) {
	globalLogger().Panic(fmt.Sprintf(format, v...))
}

// Recover stops a panicking goroutine and logs an Error.
func Recover(msg string) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("%s. Recovering, but please report this.", msg)
		globalLogger().WithOptions(zap.AddCallerSkip(1)).
			Error(msg, zap.Any("panic", r), zap.Stack("stack"))
	}
}
