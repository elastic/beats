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

package report

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/monitoring"
)

// Enumerations of various Formats. A reporter can choose whether to
// interpret this setting or not, and if so, how to interpret it.
const (
	FormatUnknown Format = iota // to protect against zero-value errors
	FormatXPackMonitoringBulk
	FormatBulk
)

var (
	defaultConfig = config{}

	reportFactories = map[string]ReporterFactory{}

	errMonitoringBothConfigEnabled = errors.New("both xpack.monitoring.* and monitoring.* cannot be set. Prefer to set monitoring.* and set monitoring.elasticsearch.hosts to monitoring cluster hosts")
	warnMonitoringDeprecatedConfig = "xpack.monitoring.* settings are deprecated. Use monitoring.* instead, but set monitoring.elasticsearch.hosts to monitoring cluster hosts"
)

// Format encodes the type of format to report monitoring data in. This
// is currently only being used by the elasticsearch reporter.
// This is a hack that is necessary so we can map certain monitoring
// configuration options to certain behaviors in reporters. Depending on
// the configuration option used, the correct format is set, and reporters
// that know how to interpret the format use it to choose the appropriate
// reporting behavior.
type Format int

type config struct {
	// allow for maximum one reporter being configured
	Reporter common.ConfigNamespace `config:",inline"`
}

// Settings is a collection of options defining reporter
type Settings struct {
	DefaultUsername string
	Format          Format
}

// Reporter gives ability to start,stop and identify a reporter
type Reporter interface {
	fmt.Stringer
	Start()
	Stop()
}

// ReporterFactory is a factory function returning specific Reporter
type ReporterFactory func(beat.Info, Settings, *common.Config) (Reporter, error)

// SelectConfig selects the appropriate monitoring configuration based on the user's settings in $BEAT.yml. Users may either
// use xpack.monitoring.* settings OR monitoring.* settings but not both.
func SelectConfig(beatCfg monitoring.BeatConfig) (*common.Config, *Settings, error) {
	switch {
	case beatCfg.Monitoring.Enabled() && beatCfg.XPackMonitoring.Enabled():
		return nil, nil, errMonitoringBothConfigEnabled
	case beatCfg.XPackMonitoring.Enabled():
		cfgwarn.Deprecate("7.0", warnMonitoringDeprecatedConfig)
		monitoringCfg := beatCfg.XPackMonitoring
		return monitoringCfg, &Settings{Format: FormatXPackMonitoringBulk}, nil
	case beatCfg.Monitoring.Enabled():
		monitoringCfg := beatCfg.Monitoring
		return monitoringCfg, &Settings{Format: FormatBulk}, nil
	default:
		return nil, nil, nil
	}
}

// RegisterReporterFactory registers a factory for a specific reporter type
func RegisterReporterFactory(name string, f ReporterFactory) {
	if reportFactories[name] != nil {
		panic(fmt.Sprintf("Reporter '%v' already registered", name))
	}
	reportFactories[name] = f
}

// New returns reporter for the given config
func New(
	beat beat.Info,
	settings Settings,
	cfg *common.Config,
	outputs common.ConfigNamespace,
) (Reporter, error) {
	name, cfg, err := getReporterConfig(cfg, settings, outputs)
	if err != nil {
		return nil, err
	}

	f := reportFactories[name]
	if f == nil {
		return nil, fmt.Errorf("unknown reporter type '%v'", name)
	}

	return f(beat, settings, cfg)
}

func getReporterConfig(
	monitoringConfig *common.Config,
	settings Settings,
	outputs common.ConfigNamespace,
) (string, *common.Config, error) {
	cfg := collectSubObject(monitoringConfig)
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return "", nil, err
	}

	// load reporter from `monitoring` section and optionally
	// merge with output settings
	if config.Reporter.IsSet() {
		name := config.Reporter.Name()
		rc := config.Reporter.Config()

		// merge reporter config with output config if both are present
		if outCfg := outputs.Config(); outputs.Name() == name && outCfg != nil {
			// require monitoring to not configure any hosts if output is configured:
			hosts := struct {
				Hosts []string `config:"hosts"`
			}{}
			rc.Unpack(&hosts)

			if settings.Format == FormatXPackMonitoringBulk && len(hosts.Hosts) > 0 {
				pathMonHosts := rc.PathOf("hosts")
				pathOutHost := outCfg.PathOf("hosts")
				err := fmt.Errorf("'%v' and '%v' are configured", pathMonHosts, pathOutHost)
				return "", nil, err
			}

			merged, err := common.MergeConfigs(outCfg, rc)
			if err != nil {
				return "", nil, err
			}
			rc = merged
		}

		return name, rc, nil
	}

	// find output also available for reporting telemetry.
	if outputs.IsSet() {
		name := outputs.Name()
		if reportFactories[name] != nil {
			return name, outputs.Config(), nil
		}
	}

	return "", nil, errors.New("No monitoring reporter configured")
}

func collectSubObject(cfg *common.Config) *common.Config {
	out := common.NewConfig()
	for _, field := range cfg.GetFields() {
		if obj, err := cfg.Child(field, -1); err == nil {
			// on error field is no object, but primitive value -> ignore
			out.SetChild(field, -1, obj)
			continue
		}
	}
	return out
}
