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
	"sync"

	"github.com/joeshaw/multierror"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/elastic-agent-libs/config"
)

// RunnerList implements a reloadable.List of Runners
type RunnerList struct {
	runners  map[uint64]Runner
	mutex    sync.RWMutex
	factory  RunnerFactory
	pipeline beat.PipelineConnector
	logger   *logp.Logger
}

// NewRunnerList builds and returns a RunnerList
func NewRunnerList(name string, factory RunnerFactory, pipeline beat.PipelineConnector) *RunnerList {
	return &RunnerList{
		runners:  map[uint64]Runner{},
		factory:  factory,
		pipeline: pipeline,
		logger:   logp.NewLogger(name),
	}
}

// Reload the list of runners to match the given state
func (r *RunnerList) Reload(configs []*reload.ConfigWithMeta) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errs multierror.Errors

	startList := map[uint64]*reload.ConfigWithMeta{}
	stopList := r.copyRunnerList()

	r.logger.Debugf("Starting reload procedure, current runners: %d", len(stopList))

	// diff current & desired state, create action lists
	for _, config := range configs {
		hash, err := HashConfig(config.Config)
		if err != nil {
			r.logger.Errorf("Unable to hash given config: %s", err)
			errs = append(errs, errors.Wrap(err, "Unable to hash given config"))
			continue
		}

		if _, ok := r.runners[hash]; ok {
			delete(stopList, hash)
		} else {
			startList[hash] = config
		}
	}

	r.logger.Debugf("Start list: %d, Stop list: %d", len(startList), len(stopList))

	// Stop removed runners
	for hash, runner := range stopList {
		r.logger.Debugf("Stopping runner: %s", runner)
		delete(r.runners, hash)
		go runner.Stop()
		moduleStops.Add(1)
	}

	// Start new runners
	for hash, config := range startList {
		runner, err := createRunner(r.factory, r.pipeline, config)
		if err != nil {
			if _, ok := err.(*common.ErrInputNotFinished); ok {
				// error is related to state, we should not log at error level
				r.logger.Debugf("Error creating runner from config: %s", err)
			} else {
				r.logger.Errorf("Error creating runner from config: %s", err)
			}
			errs = append(errs, errors.Wrap(err, "Error creating runner from config"))
			continue
		}

		r.logger.Debugf("Starting runner: %s", runner)
		r.runners[hash] = runner
		runner.Start()
		moduleStarts.Add(1)
	}

	// NOTE: This metric tracks the number of modules in the list. The true
	// number of modules in the running state may differ because modules can
	// stop on their own (i.e. on errors) and also when this stops a module
	// above it is done asynchronously.
	moduleRunning.Set(int64(len(r.runners)))

	return errs.Err()
}

// Stop all runners
func (r *RunnerList) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.runners) == 0 {
		return
	}

	r.logger.Infof("Stopping %v runners ...", len(r.runners))

	wg := sync.WaitGroup{}
	for hash, runner := range r.copyRunnerList() {
		wg.Add(1)

		delete(r.runners, hash)

		// Stop modules in parallel
		go func(h uint64, run Runner) {
			defer wg.Done()
			r.logger.Debugf("Stopping runner: %s", run)
			run.Stop()
			r.logger.Debugf("Stopped runner: %s", run)
		}(hash, runner)
	}

	wg.Wait()
}

// Has returns true if a runner with the given hash is running
func (r *RunnerList) Has(hash uint64) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	_, ok := r.runners[hash]
	return ok
}

// HashConfig hashes a given config.C
func HashConfig(c *config.C) (uint64, error) {
	var config map[string]interface{}
	if err := c.Unpack(&config); err != nil {
		return 0, err
	}
	return hashstructure.Hash(config, nil)
}

func (r *RunnerList) copyRunnerList() map[uint64]Runner {
	list := make(map[uint64]Runner, len(r.runners))
	for k, v := range r.runners {
		list[k] = v
	}
	return list
}

func createRunner(factory RunnerFactory, pipeline beat.PipelineConnector, cfg *reload.ConfigWithMeta) (Runner, error) {
	// Pass a copy of the config to the factory, this way if the factory modifies it,
	// that doesn't affect the hash of the original one.
	c, _ := config.NewConfigFrom(cfg.Config)
	return factory.Create(pipetool.WithDynamicFields(pipeline, cfg.Meta), c)
}
