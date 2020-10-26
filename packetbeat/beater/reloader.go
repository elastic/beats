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

package beater

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
)

type reloader struct {
	mutex      sync.Mutex
	factory    cfgfile.RunnerFactory
	runner     cfgfile.Runner
	configHash uint64
	pipeline   beat.PipelineConnector
	logger     *logp.Logger
}

func newReloader(name string, factory cfgfile.RunnerFactory, pipeline beat.PipelineConnector) *reloader {
	return &reloader{
		factory: factory,
		logger:  logp.NewLogger(name),
	}
}

func (r *reloader) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.runner != nil {
		r.runner.Stop()
	}
}

func (r *reloader) Reload(config *reload.ConfigWithMeta) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Starting reload procedure")

	hash, err := cfgfile.HashConfig(config.Config)
	if err != nil {
		r.logger.Errorf("Unable to hash given config: %s", err)
		return errors.Wrap(err, "Unable to hash given config")
	}

	if hash == r.configHash {
		// we have the same config reloaded
		return nil
	}
	// reinitialize config hash
	r.configHash = 0

	if r.runner != nil {
		go r.runner.Stop()
	}
	// reinitialize runner
	r.runner = nil

	c, err := common.NewConfigFrom(config.Config)
	if err != nil {
		r.logger.Errorf("Unable to create new configuration for factory: %s", err)
		return errors.Wrap(err, "Unable to create new configuration for factory")
	}
	runner, err := r.factory.Create(pipetool.WithDynamicFields(r.pipeline, config.Meta), c)
	if err != nil {
		r.logger.Errorf("Unable to create new runner: %s", err)
		return errors.Wrap(err, "Unable to create new runner")
	}

	r.logger.Debugf("Starting runner: %s", runner)
	r.configHash = hash
	r.runner = runner
	runner.Start()

	return nil
}
