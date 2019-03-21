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
	"github.com/elastic/beats/libbeat/common/reload"
	"github.com/elastic/beats/libbeat/logp"
)

// PublisherList implements a reloadable.List of Runners
type PublisherList struct {
	publishers map[uint64]Publisher
	mutex      sync.RWMutex
	factory    PublisherFactory
	pipeline   beat.Pipeline
	logger     *logp.Logger
}

// NewPublisherList builds and returns a PublisherList
func NewPublisherList(name string, factory PublisherFactory, pipeline beat.Pipeline) *PublisherList {
	return &PublisherList{
		publishers: map[uint64]Publisher{},
		factory:    factory,
		pipeline:   pipeline,
		logger:     logp.NewLogger(name),
	}
}

// Reload the list of publishers to match the given state
func (r *PublisherList) Reload(configs []*reload.ConfigWithMeta) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var errs multierror.Errors

	startList := map[uint64]*reload.ConfigWithMeta{}
	stopList := r.copyPublisherList()

	r.logger.Debugf("Starting reload procedure, current publishers: %d", len(stopList))

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

	// Stop removed publishers
	for hash, runner := range stopList {
		r.logger.Debugf("Stopping runner: %s", runner)
		delete(r.publishers, hash)
		go runner.Stop()
	}

	// Start new publishers
	for hash, config := range startList {
		// Pass a copy of the config to the factory, this way if the factory modifies it,
		// that doesn't affect the hash of the original one.
		c, _ := common.NewConfigFrom(config.Config)
		runner, err := r.factory.Create(c)
		if err != nil {
			r.logger.Errorf("Error creating runner from config: %s", err)
			errs = append(errs, errors.Wrap(err, "Error creating runner from config"))
			continue
		}

		r.logger.Debugf("Starting runner: %s", runner)
		r.publishers[hash] = runner
		runner.Start(r.pipeline, config.Meta)
	}

	return errs.Err()
}

// Stop all publishers
func (r *PublisherList) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.publishers) == 0 {
		return
	}

	r.logger.Infof("Stopping %v publishers ...", len(r.publishers))

	wg := sync.WaitGroup{}
	for hash, publisher := range r.copyPublisherList() {
		wg.Add(1)

		delete(r.publishers, hash)

		// Stop modules in parallel
		go func(h uint64, run Publisher) {
			defer wg.Done()
			r.logger.Debugf("Stopping publisher: %s", run)
			run.Stop()
			r.logger.Debugf("Stopped publisher: %s", run)
		}(hash, publisher)
	}

	wg.Wait()
}

// Has returns true if a runner with the given hash is running
func (r *PublisherList) Has(hash uint64) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	_, ok := r.publishers[hash]
	return ok
}

// HashConfig hashes a given common.Config
func HashConfig(c *common.Config) (uint64, error) {
	var config map[string]interface{}
	c.Unpack(&config)
	return hashstructure.Hash(config, nil)
}

func (r *PublisherList) copyPublisherList() map[uint64]Publisher {
	list := make(map[uint64]Publisher, len(r.publishers))
	for k, v := range r.publishers {
		list[k] = v
	}
	return list
}
