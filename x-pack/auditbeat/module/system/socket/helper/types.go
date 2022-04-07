// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package helper

import (
	"github.com/elastic/beats/v8/x-pack/auditbeat/tracing"
)

// Logger exposes logging functions.
type Logger interface {
	Errorf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

// ProbeCondition is a function that allows to filter probes.
type ProbeCondition func(probe tracing.Probe) bool

// ProbeInstaller interface allows to install and uninstall kprobes.
type ProbeInstaller interface {
	// Install installs the given kprobe, returning its format and decoder.
	Install(pdef ProbeDef) (format tracing.ProbeFormat, decoder tracing.Decoder, err error)

	// UninstallInstalled removes all kprobes that have been installed by the
	// Install method.
	UninstallInstalled() error

	// UninstallIf uninstalls all Kprobes that match a condition.
	// Works on all existing kprobes, not only those installed by Install, so
	// it allows to cleanup dangling probes from a previous run.
	UninstallIf(condition ProbeCondition) error
}
