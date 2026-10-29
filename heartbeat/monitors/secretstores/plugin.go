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

package secretstores

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gohugoio/hashstructure"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
)

var errPluginClosed = errors.New("monitor is stopped")

// WrapFactory returns a copy of pf whose plugins resolve the secret
// references of their config on every run. Making a plugin never fails
// because of the secret stores: their errors are returned by the jobs, so
// the monitor reports them and retries on its next run.
func (r *Resolver) WrapFactory(pf plugin.PluginFactory) plugin.PluginFactory {
	inner := pf
	resolve := func(cfg *conf.C) (*conf.C, error) { return r.Resolve(cfg, inner.Destinations) }
	pf.Make = func(name string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
		return newSecretPlugin(resolve, inner, name, cfg, info, r.logger).plugin(), nil
	}
	return pf
}

// secretPlugin runs the jobs of an inner plugin created from the resolved
// config. Each run resolves the config again, a cache hit most of the time,
// and creates a new inner plugin if the resolved config changed. This picks
// up rotated secrets, and recovers a monitor started while its secret store
// was unavailable.
//
// Each job is scheduled on its own, so the number of jobs is fixed when the
// plugin is made. Without an inner plugin at that time, there is one job.
type secretPlugin struct {
	resolve  func(*conf.C) (*conf.C, error)
	factory  plugin.PluginFactory
	name     string
	info     beat.Info
	logger   *logp.Logger
	jobCount int

	mu sync.Mutex
	// config is the unresolved monitor config, replaced by update.
	config *conf.C
	inner  plugin.Plugin
	built  bool
	// hash is the hash of the resolved config of inner, or of createErr.
	hash      uint64
	createErr error
	closed    bool
}

func newSecretPlugin(resolve func(*conf.C) (*conf.C, error), factory plugin.PluginFactory, name string, cfg *conf.C, info beat.Info, logger *logp.Logger) *secretPlugin {
	p := &secretPlugin{
		resolve:  resolve,
		factory:  factory,
		name:     name,
		info:     info,
		logger:   logger,
		config:   cfg,
		jobCount: 1,
	}
	inner, err := p.current()
	if err != nil {
		p.logger.Warnf("Could not create the monitor plugin, retrying on every run: %v", err)
		return p
	}
	p.jobCount = len(inner.Jobs)
	return p
}

// plugin returns the plugin given to the monitor.
func (p *secretPlugin) plugin() plugin.Plugin {
	js := make([]jobs.Job, p.jobCount)
	for i := range js {
		js[i] = p.job(i)
	}
	endpoints := p.jobCount
	if p.built {
		endpoints = p.inner.Endpoints
	}
	return plugin.Plugin{
		Jobs:      js,
		DoClose:   p.close,
		DoUpdate:  p.update,
		Endpoints: endpoints,
		Logger:    p.info.Logger,
	}
}

// job returns the job running the job i of the inner plugin.
func (p *secretPlugin) job(i int) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		inner, err := p.current()
		if err != nil {
			return nil, err
		}
		if len(inner.Jobs) != p.jobCount {
			return nil, fmt.Errorf("the number of endpoints changed from %d to %d after resolving the secret references, reconfigure the monitor to apply it", p.jobCount, len(inner.Jobs))
		}
		return inner.Jobs[i](event)
	}
}

// current returns the inner plugin for the current resolved config,
// creating it if the resolved config changed.
func (p *secretPlugin) current() (plugin.Plugin, error) {
	p.mu.Lock()
	config := p.config
	p.mu.Unlock()

	// Resolve without the lock, so a slow secret store doesn't block close.
	resolved, err := p.resolve(config)
	if err != nil {
		return plugin.Plugin{}, err
	}
	hash, err := hashConfig(resolved)
	if err != nil {
		return plugin.Plugin{}, fmt.Errorf("could not hash the resolved config: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	switch {
	case p.closed:
		return plugin.Plugin{}, errPluginClosed
	case p.built && p.hash == hash:
		return p.inner, nil
	case p.createErr != nil && p.hash == hash:
		// The same config fails the same way, only a change is worth a retry.
		return plugin.Plugin{}, p.createErr
	}

	// Closing the previous plugin here is safe: only browser plugins have
	// a Close, and they have a single job, whose previous run is over since
	// runs of a job never overlap.
	_ = p.closeInner()
	inner, err := p.factory.Make(p.name, resolved, p.info)
	p.hash = hash
	if err != nil {
		// A plugin can return resources to close along with the error.
		if closeErr := inner.Close(); closeErr != nil {
			p.logger.Errorf("Could not close the monitor plugin: %v", closeErr)
		}
		p.createErr = err
		return plugin.Plugin{}, err
	}
	p.inner, p.built, p.createErr = inner, true, nil
	return inner, nil
}

// update replaces the config. The next run resolves it, so updating never
// contacts the secret stores.
func (p *secretPlugin) update(cfg *conf.C) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = cfg
	return nil
}

func (p *secretPlugin) close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return p.closeInner()
}

// closeInner closes the inner plugin. p.mu must be held.
func (p *secretPlugin) closeInner() error {
	if !p.built {
		return nil
	}
	inner := p.inner
	p.inner, p.built = plugin.Plugin{}, false
	err := inner.Close()
	if err != nil {
		p.logger.Errorf("Could not close the monitor plugin: %v", err)
	}
	return err
}

// hashConfig hashes a resolved config.
func hashConfig(c *conf.C) (uint64, error) {
	unpacked := map[string]any{}
	if err := c.Unpack(unpacked); err != nil {
		return 0, err
	}
	return hashstructure.Hash(unpacked, nil)
}
