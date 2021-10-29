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

	"go.elastic.co/ecszap"
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

type encoderCreator func(cfg zapcore.EncoderConfig) zapcore.Encoder

func buildEncoder(cfg Config) zapcore.Encoder {
	var encCfg zapcore.EncoderConfig
	var encCreator encoderCreator
	if cfg.ToSyslog {
		encCfg = SyslogEncoderConfig()
		encCreator = zapcore.NewConsoleEncoder
	} else {
		encCfg = JSONEncoderConfig()
		encCreator = zapcore.NewJSONEncoder
	}

	encCfg = ecszap.ECSCompatibleEncoderConfig(encCfg)
	return encCreator(encCfg)
}

func JSONEncoderConfig() zapcore.EncoderConfig {
	return baseEncodingConfig
}

func ConsoleEncoderConfig() zapcore.EncoderConfig {
	c := baseEncodingConfig
	c.EncodeLevel = zapcore.CapitalLevelEncoder
	c.EncodeName = bracketedNameEncoder
	return c
}

func SyslogEncoderConfig() zapcore.EncoderConfig {
	c := ConsoleEncoderConfig()
	// Time is generally added by syslog.
	// But when logging with ECS the empty TimeKey will be
	// ignored and @timestamp is still added to log line
	c.TimeKey = ""
	return c
}

func bracketedNameEncoder(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + loggerName + "]")
}
