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

package hints

import (
	"fmt"

	"strings"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	autodiscover.Registry.AddBuilder("hints", NewMetricHints)
}

const (
	module     = "module"
	namespace  = "namespace"
	hosts      = "hosts"
	metricsets = "metricsets"
	period     = "period"
	timeout    = "timeout"
	ssl        = "ssl"

	defaultTimeout = "3s"
	defaultPeriod  = "1m"
)

type metricHints struct {
	Key      string
	Registry *mb.Register
}

// NewMetricHints builds a new metrics builder based on hints
func NewMetricHints(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %v", err)
	}

	return &metricHints{config.Key, config.Registry}, nil
}

// Create configs based on hints passed from providers
func (m *metricHints) CreateConfig(event bus.Event) []*common.Config {
	var config []*common.Config
	host, _ := event["host"].(string)
	if host == "" {
		return config
	}

	port, _ := common.TryToInt(event["port"])

	hints, ok := event["hints"].(common.MapStr)
	if !ok {
		return config
	}

	modulesConfig := m.getModules(hints)
	if modulesConfig != nil {
		configs := []*common.Config{}
		for _, cfg := range modulesConfig {
			if config, err := common.NewConfigFrom(cfg); err == nil {
				configs = append(configs, config)
			}
		}
		logp.Debug("hints.builder", "generated config %+v", configs)
		// Apply information in event to the template to generate the final config
		return template.ApplyConfigTemplate(event, configs)

	}

	mod := m.getModule(hints)
	if mod == "" {
		return config
	}

	hosts, ok := m.getHostsWithPort(hints, port)
	if !ok {
		return config
	}

	ns := m.getNamespace(hints)
	msets := m.getMetricSets(hints, mod)
	tout := m.getTimeout(hints)
	ival := m.getPeriod(hints)
	sslConf := m.getSSLConfig(hints)
	procs := m.getProcessors(hints)

	moduleConfig := common.MapStr{
		"module":     mod,
		"metricsets": msets,
		"hosts":      hosts,
		"timeout":    tout,
		"period":     ival,
		"enabled":    true,
		"ssl":        sslConf,
		"processors": procs,
	}

	if ns != "" {
		moduleConfig["namespace"] = ns
	}

	logp.Debug("hints.builder", "generated config: %v", moduleConfig.String())

	// Create config object
	cfg, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		logp.Debug("hints.builder", "config merge failed with error: %v", err)
	}
	logp.Debug("hints.builder", "generated config: +%v", *cfg)
	config = append(config, cfg)

	// Apply information in event to the template to generate the final config
	// This especially helps in a scenario where endpoints are configured as:
	// co.elastic.metrics/hosts= "${data.host}:9090"
	return template.ApplyConfigTemplate(event, config)
}

func (m *metricHints) getModule(hints common.MapStr) string {
	return builder.GetHintString(hints, m.Key, module)
}

func (m *metricHints) getMetricSets(hints common.MapStr, module string) []string {
	var msets []string
	var err error
	msets = builder.GetHintAsList(hints, m.Key, metricsets)

	if len(msets) == 0 {
		// If no metricset list is given, take module defaults
		// fallback to all metricsets if module has no defaults
		msets, err = m.Registry.DefaultMetricSets(module)
		if err != nil || len(msets) == 0 {
			msets = m.Registry.MetricSets(module)
		}
	}

	return msets
}

func (m *metricHints) getHostsWithPort(hints common.MapStr, port int) ([]string, bool) {
	var result []string
	thosts := builder.GetHintAsList(hints, m.Key, hosts)

	// Only pick hosts that have ${data.port} or the port on current event. This will make
	// sure that incorrect meta mapping doesn't happen
	for _, h := range thosts {
		if strings.Contains(h, "data.port") || strings.Contains(h, fmt.Sprintf(":%d", port)) ||
			// Use the event that has no port config if there is a ${data.host}:9090 like input
			(port == 0 && strings.Contains(h, "data.host")) {
			result = append(result, h)
		}
	}

	if len(thosts) > 0 && len(result) == 0 {
		logp.Debug("hints.builder", "no hosts selected for port %d with hints: %+v", port, thosts)
		return nil, false
	}

	return result, true
}

func (m *metricHints) getNamespace(hints common.MapStr) string {
	return builder.GetHintString(hints, m.Key, namespace)
}

func (m *metricHints) getPeriod(hints common.MapStr) string {
	if ival := builder.GetHintString(hints, m.Key, period); ival != "" {
		return ival
	}

	return defaultPeriod
}

func (m *metricHints) getTimeout(hints common.MapStr) string {
	if tout := builder.GetHintString(hints, m.Key, timeout); tout != "" {
		return tout
	}
	return defaultTimeout
}

func (m *metricHints) getSSLConfig(hints common.MapStr) common.MapStr {
	return builder.GetHintMapStr(hints, m.Key, ssl)
}

func (m *metricHints) getModules(hints common.MapStr) []common.MapStr {
	return builder.GetHintAsConfigs(hints, m.Key)
}

func (m *metricHints) getProcessors(hints common.MapStr) []common.MapStr {
	return builder.GetProcessors(hints, m.Key)

}
