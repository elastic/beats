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

package cfgfile

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/pkg/config"
)

type multiplexedFactory []FactoryMatcher

// FactoryMatcher returns a RunnerFactory that can handle the given
// configuration if it supports it, otherwise it returns nil.
type FactoryMatcher func(cfg *config.C) RunnerFactory

var errConfigDoesNotMatch = errors.New("config does not match accepted configurations")

// MultiplexedRunnerFactory is a RunnerFactory that uses a list of
// FactoryMatcher to choose which RunnerFactory should handle the configuration.
// When presented a Config object, MultiplexedRunnerFactory will query the
// matchers in the order given. The first RunnerFactory returned will be used
// to create the runner.
// Creating a runner or checking a configuration will return an error if no
// matcher was found. Use MatchDefault as last argument to
// MultiplexedRunnerFactory to configure a default RunnerFactory that shall
// always be used if no other factory was matched.
func MultiplexedRunnerFactory(matchers ...FactoryMatcher) RunnerFactory {
	return multiplexedFactory(matchers)
}

// MatchHasField returns a FactoryMatcher that returns the given RunnerFactory
// when the input config contains the given field name.
func MatchHasField(field string, factory RunnerFactory) FactoryMatcher {
	return func(cfg *config.C) RunnerFactory {
		if cfg.HasField(field) {
			return factory
		}
		return nil
	}
}

// MatchDefault return a FactoryMatcher that always returns returns the given
// RunnerFactory.
func MatchDefault(factory RunnerFactory) FactoryMatcher {
	return func(cfg *config.C) RunnerFactory {
		return factory
	}
}

func (f multiplexedFactory) Create(
	p beat.PipelineConnector,
	config *config.C,
) (Runner, error) {
	factory, err := f.findFactory(config)
	if err != nil {
		return nil, err
	}
	return factory.Create(p, config)
}

func (f multiplexedFactory) CheckConfig(c *config.C) error {
	factory, err := f.findFactory(c)
	if err == nil {
		err = factory.CheckConfig(c)
	}
	return err
}

func (f multiplexedFactory) findFactory(c *config.C) (RunnerFactory, error) {
	for _, matcher := range f {
		if factory := matcher(c); factory != nil {
			return factory, nil
		}
	}

	return nil, errConfigDoesNotMatch
}

type FactoryWrapper func(RunnerFactory) RunnerFactory

type WithOtelFactoryWrapper interface {
	WithOtelFactoryWrapper(FactoryWrapper)
}

// RunnerConfigsSetter is implemented by runner factories that need to know
// every config their callers want running, not only the runners they create.
// The OTel status wrapper uses it to report a status only for configured
// runners, so the statuses of removed runners don't accumulate.
type RunnerConfigsSetter interface {
	// SetRunnerConfigs sets the configs source wants running.
	SetRunnerConfigs(source any, configs []*config.C)
}

// SetRunnerConfigs tells factory, or the first factory it wraps that
// implements RunnerConfigsSetter, which configs source wants running. Callers
// that run a set of runners call it with the whole set before they create or
// stop runners, and with no configs once they stop them all. source identifies
// the caller and must be comparable.
func SetRunnerConfigs(factory RunnerFactory, source any, configs []*config.C) {
	for {
		if s, ok := factory.(RunnerConfigsSetter); ok {
			s.SetRunnerConfigs(source, configs)
			return
		}
		w, ok := factory.(interface{ Unwrap() RunnerFactory })
		if !ok {
			return
		}
		factory = w.Unwrap()
	}
}
