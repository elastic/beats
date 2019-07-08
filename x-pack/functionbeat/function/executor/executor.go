// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package executor

import (
	"errors"

	"github.com/elastic/beats/libbeat/logp"
)

var (
	ErrNeverRun        = errors.New("executor was never executed")
	ErrCannotAdd       = errors.New("cannot add to an already executed executor")
	ErrAlreadyExecuted = errors.New("executor already executed")
)

type Context interface{}

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

func NewExecutor(log *logp.Logger) *Executor {
	if log == nil {
		log = logp.NewLogger("")
	}

	log = log.Named("executor")
	return &Executor{log: log}
}

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

func (e *Executor) IsCompleted() bool {
	return e.completed
}
