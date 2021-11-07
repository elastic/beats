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

package ecszap

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.elastic.co/ecszap/internal"
)

const version = "1.6.0"

// NewCore creates a zapcore.Core that uses an ECS conformant JSON encoder.
// This is the safest way to create an ECS compatible core.
func NewCore(cfg EncoderConfig, ws zapcore.WriteSyncer, enab zapcore.LevelEnabler) zapcore.Core {
	enc := zapcore.NewJSONEncoder(cfg.ToZapCoreEncoderConfig())
	return WrapCore(zapcore.NewCore(enc, ws, enab))
}

// WrapCore wraps a given core with ECS core functionality. For ECS
// compatibility, ensure that the wrapped zapcore.Core uses an encoder
// that is created from an ECS compatible configuration. For further details
// check out ecszap.EncoderConfig or ecszap.ECSCompatibleEncoderConfig.
func WrapCore(c zapcore.Core) zapcore.Core {
	return &core{c}
}

type core struct {
	zapcore.Core
}

// With converts error fields into ECS compliant errors
// before adding them to the logger.
func (c core) With(fields []zapcore.Field) zapcore.Core {
	convertToECSFields(fields)
	return &core{c.Core.With(fields)}
}

// Check verifies whether or not the provided entry should be logged,
// by comparing the log level with the configured log level in the core.
// If it should be logged the core is added to the returned entry.
func (c core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write converts error fields into ECS compliant errors
// before serializing the entry and fields.
func (c core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	convertToECSFields(fields)
	fields = append(fields, zap.String("ecs.version", version))
	return c.Core.Write(ent, fields)
}

func convertToECSFields(fields []zapcore.Field) {
	for i, f := range fields {
		if f.Type == zapcore.ErrorType {
			fields[i] = zapcore.Field{Key: "error",
				Type:      zapcore.ObjectMarshalerType,
				Interface: internal.NewError(f.Interface.(error)),
			}
		}
	}
}
