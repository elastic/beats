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
)

// Format encodes the type of format to report monitoring data in. This
// is currently only being used by the elaticsearch reporter.
// This is a hack that is necessary so we can map certain monitoring
// configuration options to certain behaviors in reporters. Depending on
// the configuration option used, the correct format is set, and reporters
// that know how to interpret the format use it to choose the appropriate
// reporting behavior.
type Format int

// Enumerations of various Formats. A reporter can choose whether to
// interpret this setting or not, and if so, how to interpret it.
const (
	FormatUnknown Format = iota // to protect against zero-value errors
	FormatXPackMonitoringBulk
	FormatBulk
)

type config struct {
	// allow for maximum one reporter being configured
	Reporter common.ConfigNamespace `config:",inline"`
}

type Settings struct {
	DefaultUsername string
	Format          Format
	ClusterUUID     string
}

type Reporter interface {
	Stop()
}

type ReporterFactory func(beat.Info, Settings, *common.Config) (Reporter, error)

var (
	defaultConfig = config{}

	reportFactories = map[string]ReporterFactory{}
)

func RegisterReporterFactory(name string, f ReporterFactory) {
	if reportFactories[name] != nil {
		panic(fmt.Sprintf("Reporter '%v' already registered", name))
	}
	reportFactories[name] = f
}

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
