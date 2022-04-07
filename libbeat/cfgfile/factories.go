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
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
)

type multiplexedFactory []FactoryMatcher

// FactoryMatcher returns a RunnerFactory that can handle the given
// configuration if it supports it, otherwise it returns nil.
type FactoryMatcher func(cfg *common.Config) RunnerFactory

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
	return func(cfg *common.Config) RunnerFactory {
		if cfg.HasField(field) {
			return factory
		}
		return nil
	}
}

// MatchDefault return a FactoryMatcher that always returns returns the given
// RunnerFactory.
func MatchDefault(factory RunnerFactory) FactoryMatcher {
	return func(cfg *common.Config) RunnerFactory {
		return factory
	}
}

func (f multiplexedFactory) Create(
	p beat.PipelineConnector,
	config *common.Config,
) (Runner, error) {
	factory, err := f.findFactory(config)
	if err != nil {
		return nil, err
	}
	return factory.Create(p, config)
}

func (f multiplexedFactory) CheckConfig(c *common.Config) error {
	factory, err := f.findFactory(c)
	if err == nil {
		err = factory.CheckConfig(c)
	}
	return err
}

func (f multiplexedFactory) findFactory(c *common.Config) (RunnerFactory, error) {
	for _, matcher := range f {
		if factory := matcher(c); factory != nil {
			return factory, nil
		}
	}

	return nil, errConfigDoesNotMatch
}
