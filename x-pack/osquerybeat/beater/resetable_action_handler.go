// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/logp"
)

const defaultTimeout = 15 * time.Second // expecting the osquery to be restarted within 15 secs for the action to be retried

var (
	errActionHandlerIsNotSet = errors.New("action handler is not set")
	errActionTimeout         = errors.New("action timeout")
	errActionCanceled        = errors.New("action canceled")
)

// resetableActionHandler implements the client.Action interface for the action handling.
// This wrapper for the action hanler:
// 1. Captures the "broken pipe" error. The osquery restart is expected after this.
// 2. Blocks the Execute call until the new action handler is set.
// 3. Retries the action with the new action handler once
//
// The lifetime of this should the a scope of the beat Run
type resetableActionHandler struct {
	pub actionResultPublisher

	log *logp.Logger

	ah client.Action

	mx sync.Mutex

	chSignals []chan struct{}
	timeout   time.Duration
}

type optionFunc func(a *resetableActionHandler)

func newResetableActionHandler(pub actionResultPublisher, log *logp.Logger, opts ...optionFunc) *resetableActionHandler {
	a := &resetableActionHandler{
		pub:     pub,
		log:     log,
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func resetableActionHandlerWithTimeout(timeout time.Duration) optionFunc {
	return func(a *resetableActionHandler) {
		a.timeout = timeout
	}
}

func (a *resetableActionHandler) Execute(ctx context.Context, req map[string]interface{}) (res map[string]interface{}, err error) {
	a.log.Debug(formatLogMessage("execute"))

	// Normalize error on exit, always return nil as error and encode it into result as per current contract
	defer func() {
		if err != nil {
			res = renderResult(res, err)
			err = nil
		}
		if a.pub != nil {
			a.pub.PublishActionResult(req, res)
		}
	}()

	res, err = a.execute(ctx, req)
	if err != nil {
		a.log.Error(formatLogMessage("execute error: %v", err))
		if !isBrokenPipeOrEOFError(err) {
			return nil, err
		}

		a.log.Info(formatLogMessage("broken pipe error, retry the action with timeout: %v", a.timeout))

		// Signal channel is used for waiting for the new action handler to be set in order to retry
		chSig := make(chan struct{}, 1)

		a.mx.Lock()
		a.chSignals = append(a.chSignals, chSig)
		a.mx.Unlock()

		// Set the timer for timeout
		t := time.NewTimer(a.timeout)
		defer t.Stop()

		// Wait for either:
		// 1. New action handler set signal
		// 2. Timeout
		// 3. Context cancelation
		select {
		case _, ok := <-chSig:
			if ok {
				a.log.Debug(formatLogMessage("got new action handler signal, retrying the action"))
				return a.execute(ctx, req)
			}
			a.log.Debug(formatLogMessage("got cancel signal, exiting"))
			return nil, errActionCanceled
		case <-t.C:
			a.log.Debug(formatLogMessage("action retry timed out, with timeout value: %v", a.timeout))
			return nil, errActionTimeout
		case <-ctx.Done():
			a.log.Debug(formatLogMessage("action retry canceled, context: %v", ctx.Err()))
			return nil, ctx.Err()
		}
	}
	return res, nil
}

func formatLogMessage(format string, args ...interface{}) string {
	return "resetable action handler: " + fmt.Sprintf(format, args...)
}

func renderResult(res map[string]interface{}, err error) map[string]interface{} {
	if res == nil {
		now := time.Now().UTC()
		res = map[string]interface{}{
			"started_at":   now.Format(time.RFC3339Nano),
			"completed_at": now.Format(time.RFC3339Nano),
		}
	}
	if err != nil {
		res["error"] = err.Error()
	}
	return res
}

func (a *resetableActionHandler) execute(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	a.mx.Lock()
	defer a.mx.Unlock()

	if a.ah == nil {
		return nil, errActionHandlerIsNotSet
	}

	// The current action handler always returns result and error set as a property in the result
	var err error
	res, _ := a.ah.Execute(ctx, req)
	if e, ok := res["error"]; ok {
		if emsg, ok := e.(string); ok {
			err = errors.New(emsg)
		}
	}
	return res, err
}

func (a *resetableActionHandler) Name() string {
	a.mx.Lock()
	defer a.mx.Unlock()
	if a.ah == nil {
		return ""
	}
	return a.ah.Name()
}

func (a *resetableActionHandler) Attach(ah client.Action) {
	a.mx.Lock()
	a.ah = ah
	chSignals := a.chSignals
	a.chSignals = make([]chan struct{}, 0)
	a.mx.Unlock()

	// signal pending
	for _, chSig := range chSignals {
		chSig <- struct{}{}
	}
}

func (a *resetableActionHandler) Clear() {
	a.mx.Lock()
	a.ah = nil
	chSignals := a.chSignals
	a.chSignals = make([]chan struct{}, 0)
	a.mx.Unlock()

	// signal pending to cancel by closing channel
	for _, chSig := range chSignals {
		close(chSig)
	}
}
