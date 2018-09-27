package aws

import (
	"errors"

	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

var errNeverRun = errors.New("executer was never executed")
var errCannotAdd = errors.New("cannot add to an already executed executer")
var errAlreadyExecuted = errors.New("executer already executed")

type executerContext struct {
	Content     []byte
	Name        string
	FunctionArn string
	Description string
	HandleName  string
	Role        string
	Runtime     lambdaApi.Runtime
}

type executer struct {
	Context    *executerContext
	operations []doer
	undos      []undoer
	completed  bool
	log        *logp.Logger
}

type doer interface {
	Execute(*executerContext) error
}

type undoer interface {
	Rollback(*executerContext) error
}

func newExecuter(log *logp.Logger, context *executerContext) *executer {
	if log == nil {
		log = logp.NewLogger("executer")
	}

	return &executer{log: log, Context: context}
}

func (e *executer) Execute() (err error) {
	e.log.Debugf("executing %d calls", len(e.operations))
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
	e.markCompleted()
	return err
}

func (e *executer) Rollback() (err error) {
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
	return err
}

func (e *executer) Add(operation doer) error {
	if e.IsCompleted() {
		return errCannotAdd
	}
	e.operations = append(e.operations, operation)
	return nil
}

func (e *executer) markCompleted() {
	e.completed = true
}

func (e *executer) IsCompleted() bool {
	return e.completed
}
