// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// Subset of logp configuration we need.
type logpConfig struct {
	Beat       string     `config:",ignore"` // Name of the Beat (for default file name).
	Level      logp.Level `config:"level"`   // Logging level (error, warning, info, debug).
	ECSEnabled bool       `config:"ecs"`     // Adds minimal ECS information using ECS conformant keys to every log line

	toObserver  bool
	toIODiscard bool
	ToStderr    bool `config:"to_stderr" yaml:"to_stderr"`
	ToFiles     bool `config:"to_files" yaml:"to_files"`

	Files logpFileConfig `config:"files"`

	environment logp.Environment
	addCaller   bool // Adds package and line number info to messages.
}

type logpFileConfig struct {
	Path            string        `config:"path"`
	Name            string        `config:"name"`
	MaxSize         uint          `config:"rotateeverybytes" validate:"min=1"`
	MaxBackups      uint          `config:"keepfiles" validate:"max=1024"`
	Permissions     uint32        `config:"permissions"`
	Interval        time.Duration `config:"interval"`
	RotateOnStartup bool          `config:"rotateonstartup"`
	RedirectStderr  bool          `config:"redirect_stderr"`
}

func defaultLogpConfig() logpConfig {
	return logpConfig{
		Level: logp.DebugLevel,
		Files: logpFileConfig{
			MaxSize:         10 * 1024 * 1024,
			MaxBackups:      5,
			Permissions:     0600,
			Interval:        0,
			RotateOnStartup: true,
		},
		environment: logp.DefaultEnvironment,
		addCaller:   true,
	}
}
