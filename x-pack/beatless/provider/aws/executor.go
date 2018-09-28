// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"

	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

var errNeverRun = errors.New("executer was never executed")
var errCannotAdd = errors.New("cannot add to an already executed executer")
var errAlreadyExecuted = errors.New("executer already executed")

type executorContext struct {
	Content     []byte
	Name        string
	FunctionArn string
	AliasArn    string
	Description string
	HandleName  string
	Role        string
	Runtime     lambdaApi.Runtime
}

type executor struct {
	Context    *executorContext
	operations []doer
	undos      []undoer
	completed  bool
	log        *logp.Logger
}

type doer interface {
	Execute(*executorContext) error
}

type undoer interface {
	Rollback(*executorContext) error
}

func newExecutor(log *logp.Logger, context *executorContext) *executor {
	if log == nil {
		log = logp.NewLogger("executer")
	}

	return &executor{log: log, Context: context}
}

func (e *executor) Execute() (err error) {
	e.log.Debugf("executing %d operations", len(e.operations))
	if e.IsCompleted() {
		return errAlreadyExecuted
	}
	context := e.Context
	for _, operation := range e.operations {
		err = operation.Execute(context)
		if err != nil {
			break
		}
		v, ok := operation.(undoer)
		if ok {
			e.undos = append(e.undos, v)
		}
	}
	if err == nil {
		e.log.Debug("all operations successful")
	}
	e.markCompleted()
	return err
}

func (e *executor) Rollback() (err error) {
	e.log.Debugf("rolling back previous execution, %d operations", len(e.undos))
	if !e.IsCompleted() {
		return errNeverRun
	}
	context := e.Context
	for i := len(e.undos) - 1; i >= 0; i-- {
		operation := e.undos[i]
		err = operation.Rollback(context)
		if err != nil {
			break
		}
	}

	if err == nil {
		e.log.Debug("rollback successful")
	} else {
		e.log.Debug("rollback incomplete")
	}
	return err
}

func (e *executor) Add(operation ...doer) error {
	if e.IsCompleted() {
		return errCannotAdd
	}
	e.operations = append(e.operations, operation...)
	return nil
}

func (e *executor) markCompleted() {
	e.completed = true
}

func (e *executor) IsCompleted() bool {
	return e.completed
}
