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

package module

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Factory creates new Runner instances from configuration objects.
// It is used to register and reload modules.
type Factory struct {
	beatInfo    beat.Info
	monitoring  beatmonitoring.Monitoring
	options     []Option
	registry    *mb.Register
	batchedMode bool
}

// metricSetWithProcessors is an interface to check if a MetricSet has directly attached Processors
// NOTE: Processors that implement the Closer interface are going to be closed from the pipeline when required,
// namely during dynamic configuration reloading. Thus, it is critical for the Metricset to always instantiate
// properly the processor and not consider it as always running.
type metricSetWithProcessors interface {
	Processors() []beat.Processor
}

// NewFactory creates new Reloader instance for the given config
func NewFactory(beatInfo beat.Info, monitoring beatmonitoring.Monitoring, registry *mb.Register, options ...Option) cfgfile.RunnerFactory {
	return &Factory{
		beatInfo:   beatInfo,
		monitoring: monitoring,
		options:    options,
		registry:   registry,
	}
}

// NewBatchedFactory creates a Factory that uses batched runners for periodic
// metricsets. All periodic metricsets in a module are synchronized on a single
// ticker and their events are sent via client.PublishAll per metricset per
// cycle instead of one event at a time.
func NewBatchedFactory(beatInfo beat.Info, monitoring beatmonitoring.Monitoring, registry *mb.Register, options ...Option) cfgfile.RunnerFactory {
	return &Factory{
		beatInfo:    beatInfo,
		monitoring:  monitoring,
		options:     options,
		registry:    registry,
		batchedMode: true,
	}
}

// Create creates a new metricbeat module runner reporting events to the passed pipeline.
func (r *Factory) Create(p beat.PipelineConnector, c *conf.C) (cfgfile.Runner, error) {
	module, metricSets, err := mb.NewModule(c, r.registry, r.beatInfo.Paths, r.beatInfo.Logger)
	if err != nil {
		return nil, err
	}

	if r.batchedMode {
		return r.createBatched(p, c, module, metricSets)
	}
	return r.createPerMetricSet(p, c, module, metricSets)
}

// createPerMetricSet creates one runner per metricset (the default mode).
func (r *Factory) createPerMetricSet(p beat.PipelineConnector, c *conf.C, module mb.Module, metricSets []mb.MetricSet) (cfgfile.Runner, error) {
	runners := make([]cfgfile.Runner, 0, len(metricSets))
	for _, metricSet := range metricSets {
		wrapper, err := NewWrapperForMetricSet(module, metricSet, r.monitoring, r.beatInfo.Logger, r.options...)
		if err != nil {
			return nil, err
		}

		client, err := r.connectMetricSet(p, c, module, metricSet)
		if err != nil {
			return nil, err
		}
		runners = append(runners, NewRunner(client, wrapper))
	}

	return newRunnerGroup(runners), nil
}

// createBatched creates a single batched runner for all periodic metricsets
// in the module. Push metricsets get individual runners as usual.
func (r *Factory) createBatched(p beat.PipelineConnector, c *conf.C, module mb.Module, metricSets []mb.MetricSet) (cfgfile.Runner, error) {
	var (
		pushRunners    []cfgfile.Runner
		batchedMSWs    []*metricSetWrapper
		batchedClients []beat.Client
	)

	for _, metricSet := range metricSets {
		wrapper, err := NewWrapperForMetricSet(module, metricSet, r.monitoring, r.beatInfo.Logger, r.options...)
		if err != nil {
			return nil, err
		}

		client, err := r.connectMetricSet(p, c, module, metricSet)
		if err != nil {
			return nil, err
		}

		msw := wrapper.metricSets[0]
		if msw.isPush() {
			pushRunners = append(pushRunners, NewRunner(client, wrapper))
		} else {
			batchedMSWs = append(batchedMSWs, msw)
			batchedClients = append(batchedClients, client)
		}
	}

	var runners []cfgfile.Runner
	runners = append(runners, pushRunners...)
	if len(batchedMSWs) > 0 {
		runners = append(runners, newBatchedRunner(batchedClients, batchedMSWs, module, r.monitoring, r.beatInfo.Logger))
	}

	return newRunnerGroup(runners), nil
}

// connectMetricSet creates a pipeline client for a metricset with the
// appropriate processors configured.
func (r *Factory) connectMetricSet(p beat.PipelineConnector, c *conf.C, module mb.Module, metricSet mb.MetricSet) (beat.Client, error) {
	connector, err := NewConnector(r.beatInfo, p, c)
	if err != nil {
		return nil, err
	}

	err = connector.UseMetricSetProcessors(r.registry, module.Name(), metricSet.Name())
	if err != nil {
		return nil, err
	}

	if msWithProcs, ok := metricSet.(metricSetWithProcessors); ok {
		connector.addProcessors(msWithProcs.Processors())
	}

	return connector.Connect()
}

// CheckConfig checks if a config is valid or not
func (r *Factory) CheckConfig(config *conf.C) error {
	_, err := NewWrapper(config, r.registry, r.beatInfo.Logger, r.monitoring, r.beatInfo.Paths, r.options...)
	if err != nil {
		return err
	}

	return nil
}
