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

package logp

import (
	"go.uber.org/zap/zapcore"
)

var baseEncodingConfig = zapcore.EncoderConfig{
	TimeKey:        "timestamp",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	MessageKey:     "message",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.NanosDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
	EncodeName:     zapcore.FullNameEncoder,
}

func buildEncoder(cfg Config) zapcore.Encoder {
	if cfg.JSON {
		return zapcore.NewJSONEncoder(jsonEncoderConfig())
	} else if cfg.ToSyslog {
		return zapcore.NewConsoleEncoder(syslogEncoderConfig())
	} else {
		return zapcore.NewConsoleEncoder(consoleEncoderConfig())
	}
}

func jsonEncoderConfig() zapcore.EncoderConfig {
	return baseEncodingConfig
}

func consoleEncoderConfig() zapcore.EncoderConfig {
	c := baseEncodingConfig
	c.EncodeLevel = zapcore.CapitalLevelEncoder
	c.EncodeName = bracketedNameEncoder
	return c
}

func syslogEncoderConfig() zapcore.EncoderConfig {
	c := consoleEncoderConfig()
	// Time is added by syslog.
	c.TimeKey = ""
	return c
}

func bracketedNameEncoder(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + loggerName + "]")
}
