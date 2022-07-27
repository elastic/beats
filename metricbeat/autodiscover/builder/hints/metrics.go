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
	"strconv"
	"strings"

	"github.com/elastic/go-ucfg"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-autodiscover/utils"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	err := autodiscover.Registry.AddBuilder("hints", NewMetricHints)
	if err != nil {
		logp.Error(fmt.Errorf("could not add `hints` builder"))
	}
}

const (
	module         = "module"
	namespace      = "namespace"
	hosts          = "hosts"
	metricsets     = "metricsets"
	period         = "period"
	timeout        = "timeout"
	ssl            = "ssl"
	metricsfilters = "metrics_filters"
	metricspath    = "metrics_path"
	username       = "username"
	password       = "password"

	defaultTimeout = "3s"
	defaultPeriod  = "1m"
)

type metricHints struct {
	Key      string
	Registry *mb.Register

	logger *logp.Logger
}

// NewMetricHints builds a new metrics builder based on hints
func NewMetricHints(cfg *conf.C) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %w", err)
	}

	return &metricHints{config.Key, config.Registry, logp.NewLogger("hints.builder")}, nil
}

// Create configs based on hints passed from providers
func (m *metricHints) CreateConfig(event bus.Event, options ...ucfg.Option) []*conf.C {
	var (
		configs []*conf.C
		noPort  bool
	)
	host, _ := event["host"].(string)
	if host == "" {
		return configs
	}

	port, ok := common.TryToInt(event["port"])
	if !ok {
		noPort = true
	}

	hints, ok := event["hints"].(mapstr.M)
	if !ok {
		return configs
	}

	modulesConfig := m.getModuleConfigs(hints)
	// here we handle raw configs if provided
	if modulesConfig != nil {
		configs := []*conf.C{}
		for _, cfg := range modulesConfig {
			if config, err := conf.NewConfigFrom(cfg); err == nil {
				configs = append(configs, config)
			}
		}
		logp.Debug("hints.builder", "generated config %+v", configs)
		// Apply information in event to the template to generate the final config
		return template.ApplyConfigTemplate(event, configs, options...)

	}

	modules := m.getModules(hints)
	for _, hint := range modules {
		mod := m.getModule(hint)
		if mod == "" {
			continue
		}

		hosts, ok := m.getHostsWithPort(hint, port, noPort)
		if !ok {
			continue
		}

		ns := m.getNamespace(hint)
		msets := m.getMetricSets(hint, mod)
		tout := m.getTimeout(hint)
		ival := m.getPeriod(hint)
		sslConf := m.getSSLConfig(hint)
		procs := m.getProcessors(hint)
		metricspath := m.getMetricPath(hint)
		username := m.getUsername(hint)
		password := m.getPassword(hint)

		moduleConfig := mapstr.M{
			"module":     mod,
			"metricsets": msets,
			"timeout":    tout,
			"period":     ival,
			"enabled":    true,
			"ssl":        sslConf,
			"processors": procs,
		}

		if mod == "prometheus" {
			moduleConfig[metricsfilters] = m.getMetricsFilters(hint)
		}

		if ns != "" {
			moduleConfig["namespace"] = ns
		}
		if metricspath != "" {
			moduleConfig["metrics_path"] = metricspath
		}
		if username != "" {
			moduleConfig["username"] = username
		}
		if password != "" {
			moduleConfig["password"] = password
		}

		// If there are hosts that match, ensure that there is a module config for each valid host.
		// We do this because every config that is from a Pod that has an exposed port will generate a valid
		// module config. However, the pod level hint will generate a config with all hosts that are defined in the
		// config. To make sure that these pod level configs get deduped, it is essential that we generate exactly one
		// module config per host.
		if len(hosts) != 0 {
			for _, h := range hosts {
				mod := moduleConfig.Clone()
				mod["hosts"] = []string{h}

				logp.Debug("hints.builder", "generated config: %v", mod)

				// Create config object
				cfg := m.generateConfig(mod)
				configs = append(configs, cfg)
			}
		} else {
			logp.Debug("hints.builder", "generated config: %v", moduleConfig)

			// Create config object
			cfg := m.generateConfig(moduleConfig)
			configs = append(configs, cfg)
		}

	}

	// Apply information in event to the template to generate the final config
	// This especially helps in a scenario where endpoints are configured as:
	// co.elastic.metrics/hosts= "${data.host}:9090"
	return template.ApplyConfigTemplate(event, configs, options...)
}

func (m *metricHints) generateConfig(mod mapstr.M) *conf.C {
	cfg, err := conf.NewConfigFrom(mod)
	if err != nil {
		logp.Debug("hints.builder", "config merge failed with error: %v", err)
	}
	logp.Debug("hints.builder", "generated config: %+v", conf.DebugString(cfg, true))
	return cfg
}

func (m *metricHints) getModule(hints mapstr.M) string {
	return utils.GetHintString(hints, m.Key, module)
}

func (m *metricHints) getMetricSets(hints mapstr.M, module string) []string {
	var msets []string
	var err error
	msets = utils.GetHintAsList(hints, m.Key, metricsets)

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

func (m *metricHints) getHostsWithPort(hints mapstr.M, port int, noPort bool) ([]string, bool) {
	var result []string
	thosts := utils.GetHintAsList(hints, m.Key, hosts)

	// Only pick hosts that:
	// 1. have noPort (pod level event) and data.ports.<port_name> defined
	// 2. have ${data.port} or the port on current event.
	// This will make sure that incorrect meta mapping doesn't happen
	for _, h := range thosts {
		if strings.Contains(h, "data.ports.") && noPort {
			result = append(result, fmt.Sprintf("'%v'", h))
			// move on to the next host
			continue
		}
		if strings.Contains(h, "data.port") && port != 0 && !noPort || m.checkHostPort(h, port) ||
			// Use the event that has no port config if there is a ${data.host}:9090 like input
			(noPort && strings.Contains(h, "data.host")) {
			result = append(result, h)
		}
	}

	if len(thosts) > 0 && len(result) == 0 {
		m.logger.Debug("no hosts selected for port %d with hints: %+v", port, thosts)
		return nil, false
	}

	return result, true
}

func (m *metricHints) checkHostPort(h string, p int) bool {
	port := strconv.Itoa(p)

	index := strings.LastIndex(h, ":"+port)
	// Check if host contains :port. If not then return false
	if index == -1 {
		return false
	}

	// Check if the host ends with :port. Return true if yes
	end := index + len(port) + 1
	if end == len(h) {
		return true
	}

	// Check if the character immediately after :port. If its not a number then return true.
	// This is to avoid adding :80 as a valid host for an event that has port=8080
	// Also ensure that port=3306 and hint="tcp(${data.host}:3306)/" is valid
	return h[end] < '0' || h[end] > '9'
}

func (m *metricHints) getNamespace(hints mapstr.M) string {
	return utils.GetHintString(hints, m.Key, namespace)
}

func (m *metricHints) getMetricPath(hints mapstr.M) string {
	return utils.GetHintString(hints, m.Key, metricspath)
}

func (m *metricHints) getUsername(hints mapstr.M) string {
	return utils.GetHintString(hints, m.Key, username)
}

func (m *metricHints) getPassword(hints mapstr.M) string {
	return utils.GetHintString(hints, m.Key, password)
}

func (m *metricHints) getPeriod(hints mapstr.M) string {
	if ival := utils.GetHintString(hints, m.Key, period); ival != "" {
		return ival
	}

	return defaultPeriod
}

func (m *metricHints) getTimeout(hints mapstr.M) string {
	if tout := utils.GetHintString(hints, m.Key, timeout); tout != "" {
		return tout
	}
	return defaultTimeout
}

func (m *metricHints) getSSLConfig(hints mapstr.M) mapstr.M {
	return utils.GetHintMapStr(hints, m.Key, ssl)
}

func (m *metricHints) getMetricsFilters(hints mapstr.M) mapstr.M {
	mf := mapstr.M{}
	for k := range utils.GetHintMapStr(hints, m.Key, metricsfilters) {
		mf[k] = utils.GetHintAsList(hints, m.Key, metricsfilters+"."+k)
	}
	return mf
}

func (m *metricHints) getModuleConfigs(hints mapstr.M) []mapstr.M {
	return utils.GetHintAsConfigs(hints, m.Key)
}

func (m *metricHints) getProcessors(hints mapstr.M) []mapstr.M {
	return utils.GetProcessors(hints, m.Key)

}

func (m *metricHints) getModules(hints mapstr.M) []mapstr.M {
	modules := utils.GetHintsAsList(hints, m.Key)
	var output []mapstr.M

	for _, mod := range modules {
		output = append(output, mapstr.M{
			m.Key: mod,
		})
	}

	return output
}
