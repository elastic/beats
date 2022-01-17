// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reexec

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// ExecManager is the interface that the global reexec manager implements.
type ExecManager interface {
	// ReExec asynchronously re-executes command in the same PID and memory address
	// as the currently running application.
	ReExec(callback ShutdownCallbackFn, argOverrides ...string)

	// ShutdownChan returns the shutdown channel the main function should use to
	// handle shutdown of the current running application.
	ShutdownChan() <-chan bool

	// ShutdownComplete gets called from the main function once ShutdownChan channel
	// has been closed and the running application has completely shutdown.
	ShutdownComplete()
}

type manager struct {
	logger   *logger.Logger
	exec     string
	trigger  chan bool
	shutdown chan bool
	complete chan bool
}

// ShutdownCallbackFn is called once everything is shutdown and allows cleanup during reexec process.
type ShutdownCallbackFn func() error

// NewManager returns the reexec manager.
func NewManager(log *logger.Logger, exec string) ExecManager {
	return &manager{
		logger:   log,
		exec:     exec,
		trigger:  make(chan bool),
		shutdown: make(chan bool),
		complete: make(chan bool),
	}
}

func (m *manager) ReExec(shutdownCallback ShutdownCallbackFn, argOverrides ...string) {
	go func() {
		close(m.trigger)
		<-m.shutdown

		if shutdownCallback != nil {
			if err := shutdownCallback(); err != nil {
				// panic; because there is no going back, everything is shutdown
				panic(errors.New(errors.TypeUnexpected, err, "failure occurred during shutdown cleanup"))
			}
		}

		if err := reexec(m.logger, m.exec, argOverrides...); err != nil {
			// panic; because there is no going back, everything is shutdown
			panic(err)
		}

		close(m.complete)
	}()
}

func (m *manager) ShutdownChan() <-chan bool {
	return m.trigger
}

func (m *manager) ShutdownComplete() {
	close(m.shutdown)
	<-m.complete
}
