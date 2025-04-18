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
	"fmt"
	"sync"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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
func NewRunnerList(name string, factory RunnerFactory, pipeline beat.PipelineConnector, logger *logp.Logger) *RunnerList {
	return &RunnerList{
		runners:  map[uint64]Runner{},
		factory:  factory,
		pipeline: pipeline,
		logger:   logger.Named(name),
	}
}

// Runners returns a slice containing all
// currently running runners
func (r *RunnerList) Runners() []Runner {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	runners := make([]Runner, 0, len(r.runners))
	for _, r := range r.runners {
		runners = append(runners, r)
	}
	return runners
}

// Reload the list of runners to match the given state
//
// Runners might fail to start, it's the callers responsibility to
// handle any error. During execution, any encountered errors are
// accumulated in a []errors and returned as errors.Join(errs) upon completion.
//
// While the stopping of runners occurs on separate goroutines,
// Reload will wait for all runners to finish before starting any new runners.
//
// The starting of runners occurs synchronously, one after the other.
//
// It is recommended not to call this method more than once per second to avoid
// unnecessary starting and stopping of runners.
func (r *RunnerList) Reload(configs []*reload.ConfigWithMeta) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errs []error

	startList := map[uint64]*reload.ConfigWithMeta{}
	stopList := r.copyRunnerList()

	r.logger.Debugf("Starting reload procedure, current runners: %d", len(stopList))

	// diff current & desired state, create action lists
	for _, config := range configs {
		hash, err := HashConfig(config.Config)
		if err != nil {
			r.logger.Errorf("Unable to hash given config: %s", err)
			errs = append(errs, fmt.Errorf("Unable to hash given config: %w", err)) //nolint:staticcheck //Keep old behavior
			continue
		}

		if _, ok := r.runners[hash]; ok {
			delete(stopList, hash)
		} else {
			startList[hash] = config
		}
	}

	r.logger.Debugf("Start list: %d, Stop list: %d", len(startList), len(stopList))

	wg := sync.WaitGroup{}
	// Stop removed runners
	for hash, runner := range stopList {
		wg.Add(1)
		r.logger.Debugf("Stopping runner: %s", runner)
		delete(r.runners, hash)
		go func(runner Runner) {
			defer wg.Done()
			runner.Stop()
			r.logger.Debugf("Runner: '%s' has stopped", runner)
		}(runner)
		moduleStops.Add(1)
	}

	// Wait for all runners to stop before starting new ones
	wg.Wait()

	// Start new runners
	for hash, config := range startList {
		runner, err := createRunner(r.factory, r.pipeline, config)
		if err != nil {
			if errors.As(err, new(*common.ErrInputNotFinished)) {
				// error is related to state, we should not log at error level
				r.logger.Debugf("Error creating runner from config: %s", err)
			} else {
				r.logger.Errorf("Error creating runner from config: %s", err)
			}

			// If InputUnitID is not empty, then we're running under Elastic-Agent
			// and we need to report the errors per unit.
			if config.InputUnitID != "" {
				err = UnitError{
					Err:    err,
					UnitID: config.InputUnitID,
				}
			}

			errs = append(errs, fmt.Errorf("Error creating runner from config: %w", err))
			continue
		}

		r.logger.Debugf("Starting runner: %s", runner)
		r.runners[hash] = runner
		if config.StatusReporter != nil {
			if runnerWithStatus, ok := runner.(status.WithStatusReporter); ok {
				runnerWithStatus.SetStatusReporter(config.StatusReporter)
			}
		}

		runner.Start()
		moduleStarts.Add(1)
		if config.DiagCallback != nil {
			if diag, ok := runner.(diagnostics.DiagnosticReporter); ok {
				r.logger.Debugf("Runner '%s' has diagnostics, attempting to register", runner)
				for _, dc := range diag.Diagnostics() {
					config.DiagCallback.Register(dc.Name, dc.Description, dc.Filename, dc.ContentType, dc.Callback)
				}
			} else {
				r.logger.Debugf("Runner %s does not implement DiagnosticRunner, skipping", runner)
			}
		}
	}

	// NOTE: This metric tracks the number of modules in the list. The true
	// number of modules in the running state may differ because modules can
	// stop on their own (i.e. on errors) and also when this stops a module
	// above it is done asynchronously.
	moduleRunning.Set(int64(len(r.runners)))

	return errors.Join(errs...)
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

type UnitError struct {
	UnitID string
	Err    error
}

func (u UnitError) Error() string {
	return u.Err.Error()
}

func (u UnitError) Unwrap() error {
	return u.Err
}
