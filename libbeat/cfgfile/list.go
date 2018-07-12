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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// RunnerList implements a reloadable.List of Runners
type RunnerList struct {
	runners  map[uint64]Runner
	mutex    sync.RWMutex
	factory  RunnerFactory
	pipeline beat.Pipeline
	logger   *logp.Logger
}

// ConfigWithMeta holds a pair of common.Config and optional metadata for it
type ConfigWithMeta struct {
	// Config to store
	Config *common.Config

	// Meta data related to this config
	Meta *common.MapStrPointer
}

// NewRunnerList builds and returns a RunnerList
func NewRunnerList(name string, factory RunnerFactory, pipeline beat.Pipeline) *RunnerList {
	return &RunnerList{
		runners:  map[uint64]Runner{},
		factory:  factory,
		pipeline: pipeline,
		logger:   logp.NewLogger(name),
	}
}

// Reload the list of runners to match the given state
func (r *RunnerList) Reload(configs []*ConfigWithMeta) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errs multierror.Errors

	startList := map[uint64]*ConfigWithMeta{}
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

		if _, ok := stopList[hash]; ok {
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
	}

	// Start new runners
	for hash, config := range startList {
		// Pass a copy of the config to the factory, this way if the factory modifies it,
		// that doesn't affect the hash of the original one.
		c, _ := common.NewConfigFrom(config.Config)
		runner, err := r.factory.Create(r.pipeline, c, config.Meta)
		if err != nil {
			r.logger.Errorf("Error creating runner from config: %s", err)
			errs = append(errs, errors.Wrap(err, "Error creating runner from config"))
			continue
		}

		r.logger.Debugf("Starting runner: %s", runner)
		r.runners[hash] = runner
		runner.Start()
	}

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

// HashConfig hashes a given common.Config
func HashConfig(c *common.Config) (uint64, error) {
	var config map[string]interface{}
	c.Unpack(&config)
	return hashstructure.Hash(config, nil)
}

func (r *RunnerList) copyRunnerList() map[uint64]Runner {
	list := make(map[uint64]Runner, len(r.runners))
	for k, v := range r.runners {
		list[k] = v
	}
	return list
}
