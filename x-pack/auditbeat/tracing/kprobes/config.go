// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package kprobes

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

// ConfigFn is a function type that configures the kprobe tracing engine.
type ConfigFn func(*Engine) error

// Logger is an interface to abstract access to an underlying logger.
type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

// WithLogger configures the tracing engine to use the given logger.
func WithLogger(logger Logger) ConfigFn {
	return func(engine *Engine) error {
		engine.log = logger
		return nil
	}
}

// WithTraceFSPath sets a custom path for tracefs access.
func WithTraceFSPath(path string) ConfigFn {
	return func(engine *Engine) error {
		engine.traceFSpath = &path
		return nil
	}
}

// WithAutoMount toggles automatic mounting of tracefs / debugfs at their expected locations. Mounted directories
// will be unmounted after termination.
func WithAutoMount(automount bool) ConfigFn {
	return func(engine *Engine) error {
		engine.autoMount = automount
		return nil
	}
}

// WithTemplateVars adds custom template variables.
func WithTemplateVars(vars common.MapStr) ConfigFn {
	return func(engine *Engine) error {
		engine.vars.Update(vars)
		return nil
	}
}

// WithProbes adds probes to use for tracing. This option can be passed multiple times.
func WithProbes(array ...ProbeDef) ConfigFn {
	return func(engine *Engine) error {
		engine.probes = append(engine.probes, array...)
		return nil
	}
}

// WithGuesses adds guesses to run. This option can be passed multiple times.
func WithGuesses(guesses ...Guesser) ConfigFn {
	return func(engine *Engine) error {
		engine.guesses = append(engine.guesses, guesses...)
		return nil
	}
}

// WithTransform adds a new transform to apply to installed probes.
func WithTransform(t ProbeTransform) ConfigFn {
	return func(engine *Engine) error {
		engine.transforms = append(engine.transforms, t)
		return nil
	}
}

// WithSymbolResolution will lookup the first symbol in the list that is available for tracing
// and set the named variable to it.
func WithSymbolResolution(variable string, symbols []string) ConfigFn {
	return func(engine *Engine) error {
		if engine.resolveSymbols == nil {
			engine.resolveSymbols = make(map[string][]string)
		}
		if _, exists := engine.resolveSymbols[variable]; exists {
			return errors.Errorf("template variable %s already in use for symbol resolution", variable)
		}
		engine.resolveSymbols[variable] = symbols
		return nil
	}
}

var syscallFnPrefixes = []string{
	"SyS_",
	"sys_",
	"__x64_sys_",
}

func makeSyscallAlternatives(name string) []string {
	result := make([]string, len(syscallFnPrefixes))
	for idx, prefix := range syscallFnPrefixes {
		result[idx] = prefix + name
	}
	return result
}

// WithSyscall adds a syscall name to be resolved to the appropriate function in the given variable.
// For example:
//  WithSyscall("SYS_EXECVE", "execve")
// will result in the SYS_EXECVE template variable to be ``SyS_execve'', ``sys_execve'' or ``__x64_sys_execve''
// depending on which symbol the kernel exports.
func WithSyscall(variable, syscallName string) ConfigFn {
	return WithSymbolResolution(variable, makeSyscallAlternatives(syscallName))
}

// WithPerfChannelConf sets the configuration for the underlying perf channel.
func WithPerfChannelConf(cfg ...tracing.PerfChannelConf) ConfigFn {
	return func(engine *Engine) error {
		engine.perfChannelConf = append(engine.perfChannelConf, cfg...)
		return nil
	}
}
