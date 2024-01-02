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
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	// Create a logger
	logger := logp.NewLogger(logName)
	if conf.Tag != "" {
		logger = logger.With("instance_id", conf.Tag)
	}
	// Measure load times
	start := time.Now()
	defer func() {
		took := time.Now().Sub(start)
		logger.Debugf("Load of javascript pipeline took %v", took)
	}()
	// Setup JS runtime.
	s := &session{
		vm:             goja.New(),
		log:            logger,
		makeEvent:      newBeatEventV0,
		timeout:        conf.Timeout,
		tagOnException: conf.TagOnException,
	}

	// Register modules.
	for _, registerModule := range sessionHooks {
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
		return fmt.Errorf("failed to export process function: %w", err)
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
		return fmt.Errorf("failed to export register function: %w", err)
	}
	if _, err := register(goja.Undefined(), s.Runtime().ToValue(params)); err != nil {
		return fmt.Errorf("failed to register script_params: %w", err)
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
			return fmt.Errorf("failed to export test function: %w", err)
		}
		_, err := test(goja.Undefined(), nil)
		if err != nil {
			return fmt.Errorf("failed in test() function: %w", err)
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
				"panic", r,
				zap.Stack("stack"))
			if !s.evt.IsCancelled() {
				out = b
			}
			err = fmt.Errorf("unexpected panic in javascript processor: %v", r)
			if s.tagOnException != "" {
				mapstr.AddTags(b.Fields, []string{s.tagOnException})
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
			mapstr.AddTags(b.Fields, []string{s.tagOnException})
		}
		appendString(b.Fields, "error.message", err.Error(), false)
		return b, fmt.Errorf("failed in process function: %w", err)
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
	// Register mapstr.M as being a simple map[string]interface{} for
	// treatment within the JS VM.
	AddSessionHook("_type_mapstr", func(s Session) {
		s.Runtime().RegisterSimpleMapType(reflect.TypeOf(mapstr.M(nil)),
			func(i interface{}) map[string]interface{} {
				return map[string]interface{}(i.(mapstr.M))
			},
		)
	})
}

type sessionPool struct {
	New func() *session
	C   chan *session
}

func newSessionPool(p *goja.Program, c Config) (*sessionPool, error) {
	s, err := newSession(p, c, true)
	if err != nil {
		return nil, err
	}

	pool := sessionPool{
		New: func() *session {
			s, _ := newSession(p, c, false)
			return s
		},
		C: make(chan *session, c.MaxCachedSessions),
	}
	pool.Put(s)

	return &pool, nil
}

func (p *sessionPool) Get() *session {
	select {
	case s := <-p.C:
		return s
	default:
		return p.New()
	}
}

func (p *sessionPool) Put(s *session) {
	if s != nil {
		select {
		case p.C <- s:
		default:
		}
	}
}
