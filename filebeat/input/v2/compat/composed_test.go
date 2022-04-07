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

package compat

import (
	"errors"
	"testing"

	v2 "github.com/elastic/beats/v8/filebeat/input/v2"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/cfgfile"
	"github.com/elastic/beats/v8/libbeat/common"
)

type fakeRunnerFactory struct {
	OnCheck  func(*common.Config) error
	OnCreate func(beat.PipelineConnector, *common.Config) (cfgfile.Runner, error)
}

type fakeRunner struct {
	Name    string
	OnStart func()
	OnStop  func()
}

func TestCombine_CheckConfig(t *testing.T) {
	oops1 := errors.New("oops1")
	oops2 := errors.New("oops2")

	cases := map[string]struct {
		factory, fallback cfgfile.RunnerFactory
		want              error
	}{
		"success": {
			factory:  failingRunnerFactory(nil),
			fallback: failingRunnerFactory(nil),
			want:     nil,
		},
		"fail if factory fails already": {
			factory:  failingRunnerFactory(oops1),
			fallback: failingRunnerFactory(oops2),
			want:     oops1,
		},
		"do not fail in fallback if factory is fine": {
			factory:  failingRunnerFactory(nil),
			fallback: failingRunnerFactory(oops2),
			want:     nil,
		},
		"ignore ErrUnknownInput and use check from fallback": {
			factory:  failingRunnerFactory(v2.ErrUnknownInput),
			fallback: failingRunnerFactory(oops2),
			want:     oops2,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			factory := Combine(test.factory, test.fallback)
			cfg := common.MustNewConfigFrom(struct{ Type string }{"test"})
			err := factory.CheckConfig(cfg)
			if test.want != err {
				t.Fatalf("Failed. Want: %v, Got: %v", test.want, err)
			}
		})
	}
}

func TestCombine_Create(t *testing.T) {
	type validation func(*testing.T, cfgfile.Runner, error)

	wantError := func(want error) validation {
		return func(t *testing.T, _ cfgfile.Runner, got error) {
			if want != got {
				t.Fatalf("Wrong error. Want: %v, Got: %v", want, got)
			}
		}
	}

	wantRunner := func(want cfgfile.Runner) validation {
		return func(t *testing.T, got cfgfile.Runner, err error) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != want {
				t.Fatalf("Wrong runner. Want: %v Got: %v", want, got)
			}
		}
	}

	runner1 := &fakeRunner{Name: "runner1"}
	runner2 := &fakeRunner{Name: "runner2"}
	oops1 := errors.New("oops1")
	oops2 := errors.New("oops2")

	cases := map[string]struct {
		factory  cfgfile.RunnerFactory
		fallback cfgfile.RunnerFactory
		Type     string
		check    validation
	}{
		"runner exsits in factory only": {
			factory:  constRunnerFactory(runner1),
			fallback: failingRunnerFactory(oops2),
			check:    wantRunner(runner1),
		},
		"runner exists in fallback only": {
			factory:  failingRunnerFactory(v2.ErrUnknownInput),
			fallback: constRunnerFactory(runner2),
			check:    wantRunner(runner2),
		},
		"runner from factory has higher priority": {
			factory:  constRunnerFactory(runner1),
			fallback: constRunnerFactory(runner2),
			check:    wantRunner(runner1),
		},
		"if both fail return error from factory": {
			factory:  failingRunnerFactory(oops1),
			fallback: failingRunnerFactory(oops2),
			check:    wantError(oops1),
		},
		"ignore ErrUnknown": {
			factory:  failingRunnerFactory(v2.ErrUnknownInput),
			fallback: failingRunnerFactory(oops2),
			check:    wantError(oops2),
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			factory := Combine(test.factory, test.fallback)
			cfg := common.MustNewConfigFrom(struct{ Type string }{test.Type})
			runner, err := factory.Create(nil, cfg)
			test.check(t, runner, err)
		})
	}
}

// Create creates a new Runner based on the given configuration.
func (f *fakeRunnerFactory) Create(p beat.PipelineConnector, config *common.Config) (cfgfile.Runner, error) {
	if f.OnCreate == nil {
		return nil, errors.New("not implemented")
	}
	return f.OnCreate(p, config)
}

// CheckConfig tests if a confiugation can be used to create an input. If it
// is not possible to create an input using the configuration, an error must
// be returned.
func (f *fakeRunnerFactory) CheckConfig(config *common.Config) error {
	if f.OnCheck == nil {
		return errors.New("not implemented")
	}
	return f.OnCheck(config)
}

func (f *fakeRunner) String() string { return f.Name }
func (f *fakeRunner) Start() {
	if f.OnStart != nil {
		f.OnStart()
	}
}

func (f *fakeRunner) Stop() {
	if f.OnStop != nil {
		f.OnStop()
	}
}

func constRunnerFactory(runner cfgfile.Runner) cfgfile.RunnerFactory {
	return &fakeRunnerFactory{
		OnCreate: func(_ beat.PipelineConnector, _ *common.Config) (cfgfile.Runner, error) {
			return runner, nil
		},
	}
}

func failingRunnerFactory(err error) cfgfile.RunnerFactory {
	return &fakeRunnerFactory{
		OnCheck: func(_ *common.Config) error { return err },

		OnCreate: func(_ beat.PipelineConnector, _ *common.Config) (cfgfile.Runner, error) {
			return nil, err
		},
	}
}
