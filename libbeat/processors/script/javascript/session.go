// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package javascript

import (
	"reflect"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	logName = "processor.javascript"

	registerFunction   = "register"
	entryPointFunction = "process"
	testFunction       = "test"

	timeoutError = "javascript processor execution timeout"
)

// Session is an instance of the processor.
type Session interface {
	// Runtime returns the Javascript runtime used for this session.
	Runtime() *goja.Runtime

	// Event returns a pointer to the current event being processed.
	Event() Event
}

// Event is the event being processed by the processor.
type Event interface {
	// Cancel marks the event as cancelled such that it will be dropped.
	Cancel()

	// IsCancelled returns true if Cancel has been invoked.
	IsCancelled() bool

	// Wrapped returns the underlying beat.Event being wrapped. The wrapped
	// event is replaced each time a new event is processed.
	Wrapped() *beat.Event

	// JSObject returns the Value that represents this object within the
	// runtime.
	JSObject() goja.Value

	// reset replaces the inner beat.Event and resets the state.
	reset(*beat.Event) error
}

// session is a javascript runtime environment used throughout the life of
// the processor instance.
type session struct {
	vm             *goja.Runtime
	log            *logp.Logger
	makeEvent      func(Session) (Event, error)
	evt            Event
	processFunc    goja.Callable
	timeout        time.Duration
	tagOnException string
}

func newSession(p *goja.Program, conf Config, test bool) (*session, error) {
	// Setup JS runtime.
	s := &session{
		vm:             goja.New(),
		log:            logp.NewLogger(logName),
		makeEvent:      newBeatEventV0,
		timeout:        conf.Timeout,
		tagOnException: conf.TagOnException,
	}
	if conf.Tag != "" {
		s.log = s.log.With("instance_id", conf.Tag)
	}

	// Register modules.
	for name, registerModule := range sessionHooks {
		s.log.Debugf("Registering module %v with the Javascript runtime.", name)
		registerModule(s)
	}

	// Register constructor for 'new Event' to enable test() to create events.
	s.vm.Set("Event", newBeatEventV0Constructor(s))

	_, err := s.vm.RunProgram(p)
	if err != nil {
		return nil, err
	}

	if err = s.setProcessFunction(); err != nil {
		return nil, err
	}

	if len(conf.Params) > 0 {
		if err = s.registerScriptParams(conf.Params); err != nil {
			return nil, err
		}
	}

	if test {
		if err = s.executeTestFunction(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// setProcessFunction validates that the process() function exists and stores
// the handle.
func (s *session) setProcessFunction() error {
	processFunc := s.vm.Get(entryPointFunction)
	if processFunc == nil {
		return errors.New("process function not found")
	}
	if processFunc.ExportType().Kind() != reflect.Func {
		return errors.New("process is not a function")
	}
	if err := s.vm.ExportTo(processFunc, &s.processFunc); err != nil {
		return errors.Wrap(err, "failed to export process function")
	}
	return nil
}

// registerScriptParams calls the register() function and passes the params.
func (s *session) registerScriptParams(params map[string]interface{}) error {
	registerFunc := s.vm.Get(registerFunction)
	if registerFunc == nil {
		return errors.New("params were provided but no register function was found")
	}
	if registerFunc.ExportType().Kind() != reflect.Func {
		return errors.New("register is not a function")
	}
	var register goja.Callable
	if err := s.vm.ExportTo(registerFunc, &register); err != nil {
		return errors.Wrap(err, "failed to export register function")
	}
	if _, err := register(goja.Undefined(), s.Runtime().ToValue(params)); err != nil {
		return errors.Wrap(err, "failed to register script_params")
	}
	s.log.Debug("Registered params with processor")
	return nil
}

// executeTestFunction executes the test() function if it exists. Any exceptions
// will cause the processor to fail to load.
func (s *session) executeTestFunction() error {
	if testFunc := s.vm.Get(testFunction); testFunc != nil {
		if testFunc.ExportType().Kind() != reflect.Func {
			return errors.New("test is not a function")
		}
		var test goja.Callable
		if err := s.vm.ExportTo(testFunc, &test); err != nil {
			return errors.Wrap(err, "failed to export test function")
		}
		_, err := test(goja.Undefined(), nil)
		if err != nil {
			return errors.Wrap(err, "failed in test() function")
		}
		s.log.Debugf("Successful test() execution for processor.")
	}
	return nil
}

// setEvent replaces the beat event handle present in the runtime.
func (s *session) setEvent(b *beat.Event) error {
	if s.evt == nil {
		var err error
		s.evt, err = s.makeEvent(s)
		if err != nil {
			return err
		}
	}

	return s.evt.reset(b)
}

// runProcessFunc executes process() from the JS script.
func (s *session) runProcessFunc(b *beat.Event) (out *beat.Event, err error) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorw("The javascript processor caused an unexpected panic "+
				"while processing an event. Recovering, but please report this.",
				"event", common.MapStr{"original": b.Fields.String()},
				"panic", r,
				zap.Stack("stack"))
			if !s.evt.IsCancelled() {
				out = b
			}
			err = errors.Errorf("unexpected panic in javascript processor: %v", r)
			if s.tagOnException != "" {
				common.AddTags(b.Fields, []string{s.tagOnException})
			}
			appendString(b.Fields, "error.message", err.Error(), false)
		}
	}()

	if err = s.setEvent(b); err != nil {
		// Always return the event even if there was an error.
		return b, err
	}

	// Interrupt the JS code if execution exceeds timeout.
	if s.timeout > 0 {
		t := time.AfterFunc(s.timeout, func() {
			s.vm.Interrupt(timeoutError)
		})
		defer t.Stop()
	}

	if _, err = s.processFunc(goja.Undefined(), s.evt.JSObject()); err != nil {
		if s.tagOnException != "" {
			common.AddTags(b.Fields, []string{s.tagOnException})
		}
		appendString(b.Fields, "error.message", err.Error(), false)
		return b, errors.Wrap(err, "failed in process function")
	}

	if s.evt.IsCancelled() {
		return nil, nil
	}
	return b, nil
}

// Runtime returns the Javascript runtime used for this session.
func (s *session) Runtime() *goja.Runtime {
	return s.vm
}

// Event returns a pointer to the current event being processed.
func (s *session) Event() Event {
	return s.evt
}

func init() {
	// Register common.MapStr as being a simple map[string]interface{} for
	// treatment within the JS VM.
	AddSessionHook("_type_mapstr", func(s Session) {
		s.Runtime().RegisterSimpleMapType(reflect.TypeOf(common.MapStr(nil)),
			func(i interface{}) map[string]interface{} {
				return map[string]interface{}(i.(common.MapStr))
			},
		)
	})
}

type sessionPool struct {
	pool *sync.Pool
}

func newSessionPool(p *goja.Program, c Config) (*sessionPool, error) {
	s, err := newSession(p, c, true)
	if err != nil {
		return nil, err
	}

	pool := &sync.Pool{
		New: func() interface{} {
			s, _ := newSession(p, c, false)
			return s
		},
	}
	pool.Put(s)

	return &sessionPool{pool}, nil
}

func (p *sessionPool) Get() *session {
	s, _ := p.pool.Get().(*session)
	return s
}

func (p *sessionPool) Put(s *session) {
	if s != nil {
		p.pool.Put(s)
	}
}
