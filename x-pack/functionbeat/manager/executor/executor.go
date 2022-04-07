// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package executor

import (
	"errors"

	"github.com/elastic/beats/v8/libbeat/logp"
)

var (
	// ErrNeverRun is returned if the step was never run.
	ErrNeverRun = errors.New("executor was never executed")
	// ErrCannotAdd is returned if the executor had already ran and a new operation is added.
	ErrCannotAdd = errors.New("cannot add to an already executed executor")
	// ErrAlreadyExecuted is returned if it has already run.
	ErrAlreadyExecuted = errors.New("executor already executed")
)

// Context holds the information of each execution step.
type Context interface{}

// Executor tries to execute operations. If an operation fails, everything is rolled back.
type Executor struct {
	operations []doer
	undos      []undoer
	completed  bool
	log        *logp.Logger
}

type doer interface {
	Execute(Context) error
}

type undoer interface {
	Rollback(Context) error
}

// NewExecutor return a new executor.
func NewExecutor(log *logp.Logger) *Executor {
	if log == nil {
		log = logp.NewLogger("")
	}

	log = log.Named("executor")
	return &Executor{log: log}
}

// Execute executes all operations. If something fail it rolls back.
func (e *Executor) Execute(ctx Context) (err error) {
	e.log.Debugf("The executor is executing '%d' operations for converging state", len(e.operations))
	if e.IsCompleted() {
		return ErrAlreadyExecuted
	}
	for _, operation := range e.operations {
		err = operation.Execute(ctx)
		if err != nil {
			break
		}
		v, ok := operation.(undoer)
		if ok {
			e.undos = append(e.undos, v)
		}
	}
	if err == nil {
		e.log.Debug("All operations successful")
	}
	e.markCompleted()
	return err
}

// Rollback rolls back executed operations.
func (e *Executor) Rollback(ctx Context) (err error) {
	e.log.Debugf("The executor is rolling back previous execution, '%d' operations to rollback", len(e.undos))
	if !e.IsCompleted() {
		return ErrNeverRun
	}
	for i := len(e.undos) - 1; i >= 0; i-- {
		operation := e.undos[i]
		err = operation.Rollback(ctx)
		if err != nil {
			break
		}
	}

	if err == nil {
		e.log.Debug("The rollback is successful")
	} else {
		e.log.Debug("The rollback is incomplete")
	}
	return err
}

// Add adds new operation to execute.
func (e *Executor) Add(operation ...doer) error {
	if e.IsCompleted() {
		return ErrCannotAdd
	}
	e.operations = append(e.operations, operation...)
	return nil
}

func (e *Executor) markCompleted() {
	e.completed = true
}

// IsCompleted returns if all operations are completed.
func (e *Executor) IsCompleted() bool {
	return e.completed
}
