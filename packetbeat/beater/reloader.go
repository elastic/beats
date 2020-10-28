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
	"sort"
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
	mutex        sync.Mutex
	factory      cfgfile.RunnerFactory
	runner       cfgfile.Runner
	configHashes []uint64
	pipeline     beat.PipelineConnector
	logger       *logp.Logger
}

func equalHashes(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
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

func (r *reloader) Reload(configs []*reload.ConfigWithMeta) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.logger.Debug("Starting reload procedure")

	hashes := make([]uint64, len(configs))
	combined := make([]*common.Config, len(configs))
	for i, c := range configs {
		combined[i] = c.Config
		hash, err := cfgfile.HashConfig(c.Config)
		if err != nil {
			r.logger.Errorf("Unable to hash given config: %s", err)
			return errors.Wrap(err, "unable to hash given config")
		}
		hashes[i] = hash
	}
	sort.Slice(hashes, func(i, j int) bool { return hashes[i] < hashes[j] })

	config, err := common.NewConfigFrom(combined)
	if err != nil {
		r.logger.Errorf("Unable to combine configurations: %s", err)
		return errors.Wrap(err, "unable to combine configurations")
	}

	if equalHashes(hashes, r.configHashes) {
		// we have the same config reloaded
		return nil
	}
	// reinitialize config hash
	r.configHashes = nil

	if r.runner != nil {
		go r.runner.Stop()
	}
	// reinitialize runner
	r.runner = nil

	runner, err := r.factory.Create(pipetool.WithDynamicFields(r.pipeline, nil), config)
	if err != nil {
		r.logger.Errorf("Unable to create new runner: %s", err)
		return errors.Wrap(err, "unable to create new runner")
	}

	r.logger.Debugf("Starting runner: %s", runner)
	r.configHashes = hashes
	r.runner = runner
	runner.Start()

	return nil
}
