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
	v2 "github.com/elastic/beats/v8/filebeat/input/v2"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/cfgfile"
	"github.com/elastic/beats/v8/libbeat/common"
)

// composeFactory combines to factories. Instances are created using the Combine function.
// For each operation the configured factory will be tried first. If the
// operation failed (for example the input type is unknown) the fallback factory is tried.
type composeFactory struct {
	factory  cfgfile.RunnerFactory
	fallback cfgfile.RunnerFactory
}

// Combine takes two RunnerFactory instances and creates a new RunnerFactory.
// The new factory will first try to create an input using factory. If this operation fails fallback will be used.
//
// The new RunnerFactory will return the error of fallback only if factory did
// signal that the input type is unknown via v2.ErrUnknown.
//
// XXX: This RunnerFactory is used for combining the v2.Loader with the
// existing RunnerFactory for inputs in Filebeat. The Combine function should be removed once the old RunnerFactory is removed.
func Combine(factory, fallback cfgfile.RunnerFactory) cfgfile.RunnerFactory {
	return composeFactory{factory: factory, fallback: fallback}
}

func (f composeFactory) CheckConfig(cfg *common.Config) error {
	err := f.factory.CheckConfig(cfg)
	if !v2.IsUnknownInputError(err) {
		return err
	}
	return f.fallback.CheckConfig(cfg)
}

func (f composeFactory) Create(
	p beat.PipelineConnector,
	config *common.Config,
) (cfgfile.Runner, error) {
	var runner cfgfile.Runner
	var err1, err2 error

	runner, err1 = f.factory.Create(p, config)
	if err1 == nil {
		return runner, nil
	}

	runner, err2 = f.fallback.Create(p, config)
	if err2 == nil {
		return runner, nil
	}

	// return err2 only if err1 indicates that the input type is not known to f.factory
	if v2.IsUnknownInputError(err1) {
		return nil, err2
	}
	return nil, err1
}
