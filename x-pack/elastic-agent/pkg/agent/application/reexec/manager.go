// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reexec

import (
	"sync"
)

var (
	execSingleton     ExecManager
	execSingletonOnce sync.Once
)

type ExecManager interface {
	// ReExec asynchronously re-executes command in the same PID and memory address
	// as the currently running application.
	ReExec()

	// ShutdownChan returns the shutdown channel the main function should use to
	// handle shutdown of the current running application.
	ShutdownChan() chan bool

	// ShutdownComplete gets called from the main function once ShutdownChan channel
	// has been closed and the running application has completely shutdown.
	ShutdownComplete()
}

func Manager(exec string) ExecManager {
	execSingletonOnce.Do(func() {
		execSingleton = newManager(exec)
	})
	return execSingleton
}

type manager struct {
	exec     string
	trigger  chan bool
	shutdown chan bool
	complete chan bool
}

func newManager(exec string) *manager {
	return &manager{
		exec:     exec,
		trigger:  make(chan bool),
		shutdown: make(chan bool),
		complete: make(chan bool),
	}
}

func (m *manager) ReExec() {
	go func() {
		close(m.trigger)
		<-m.shutdown

		if err := exec(m.exec); err != nil {
			// panic; because there is no going back, everything is shutdown
			panic(err)
		}

		close(m.complete)
	}()
}

func (m *manager) ShutdownChan() chan bool {
	return m.trigger
}

func (m *manager) ShutdownComplete() {
	close(m.shutdown)
	<-m.complete
}
