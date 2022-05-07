// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	logName = "awss3.script"

	entryPointFunction = "parse"
	registerFunction   = "register"
	testFunction       = "test"

	timeoutError = "javascript parser execution timeout"
)

// session is a javascript runtime environment used throughout the life of
// the input instance.
type session struct {
	vm        *goja.Runtime
	log       *logp.Logger
	parseFunc goja.Callable
	timeout   time.Duration
}

func newSession(p *goja.Program, conf scriptConfig, test bool) (*session, error) {
	// Create a logger
	logger := logp.NewLogger(logName)

	// Setup JS runtime.
	s := &session{
		vm:      goja.New(),
		log:     logger,
		timeout: conf.Timeout,
	}

	// Register mapstr.M as being a simple map[string]interface{} for
	// treatment within the JS VM.
	s.vm.RegisterSimpleMapType(reflect.TypeOf(mapstr.M(nil)),
		func(i interface{}) map[string]interface{} {
			return map[string]interface{}(i.(mapstr.M))
		},
	)

	// Register constructors for 'new S3EventV2' to enable creating them from the JS code.
	s.vm.Set("S3EventV2", newJSS3EventV2Constructor(s))
	s.vm.Set("XMLDecoder", newXMLDecoderConstructor(s))

	if _, err := s.vm.RunProgram(p); err != nil {
		return nil, err
	}

	if err := s.setParseFunction(); err != nil {
		return nil, err
	}

	if len(conf.Params) > 0 {
		if err := s.registerScriptParams(conf.Params); err != nil {
			return nil, err
		}
	}

	if test {
		if err := s.executeTestFunction(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// setParseFunction validates that the parse() function exists and stores
// the handle.
func (s *session) setParseFunction() error {
	parseFunc := s.vm.Get(entryPointFunction)
	if parseFunc == nil {
		return errors.New("parse function not found")
	}
	if parseFunc.ExportType().Kind() != reflect.Func {
		return errors.New("parse is not a function")
	}
	if err := s.vm.ExportTo(parseFunc, &s.parseFunc); err != nil {
		return errors.Wrap(err, "failed to export parse function")
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
	if _, err := register(goja.Undefined(), s.vm.ToValue(params)); err != nil {
		return errors.Wrap(err, "failed to register script_params")
	}
	s.log.Debug("Registered params with script")
	return nil
}

// executeTestFunction executes the test() function if it exists. Any exceptions
// will cause the script to fail to load.
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
		s.log.Debugf("Successful test() execution for script.")
	}
	return nil
}

// runParseFunc executes parse() from the JS script.
func (s *session) runParseFunc(n string) (out []s3EventV2, err error) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorw("The javascript script caused an unexpected panic "+
				"while parsing a notification. Recovering, but please report this.",
				"notification", mapstr.M{"original": n},
				"panic", r,
				zap.Stack("stack"))
			err = fmt.Errorf("unexpected panic in javascript script: %v", r)
		}
	}()

	// Interrupt the JS code if execution exceeds timeout.
	if s.timeout > 0 {
		t := time.AfterFunc(s.timeout, func() {
			s.vm.Interrupt(timeoutError)
		})
		defer t.Stop()
	}

	v, err := s.parseFunc(goja.Undefined(), s.vm.ToValue(n))
	if err != nil {
		return nil, fmt.Errorf("failed in parse function: %w", err)
	}

	if v.Equals(goja.Undefined()) {
		return out, nil
	}

	if err := s.vm.ExportTo(v, &out); err != nil {
		return nil, fmt.Errorf("can't export returned value: %w", err)
	}

	return out, nil
}

type sessionPool struct {
	New func() *session
	C   chan *session
}

func newSessionPool(p *goja.Program, c scriptConfig) (*sessionPool, error) {
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
