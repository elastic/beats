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

	errw "github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

type config struct {
	// allow for maximum one reporter being configured
	Reporter common.ConfigNamespace `config:",inline"`
}

type Settings struct {
	DefaultUsername string
	ClusterUUID     string
}

type Reporter interface {
	Stop()
}

type ReporterFactory func(beat.Info, Settings, *common.Config) (Reporter, error)

type hostsCfg struct {
	Hosts []string `config:"hosts"`
}

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
			hosts := hostsCfg{}
			rc.Unpack(&hosts)

			merged, err := common.MergeConfigs(outCfg, rc)
			if err != nil {
				return "", nil, err
			}

			// Make sure hosts from reporter configuration get precedence over hosts
			// from output configuration
			if err := mergeHosts(merged, outCfg, rc); err != nil {
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

func mergeHosts(merged, outCfg, reporterCfg *common.Config) error {
	if merged == nil {
		merged = common.NewConfig()
	}

	outputHosts := hostsCfg{}
	if outCfg != nil {
		if err := outCfg.Unpack(&outputHosts); err != nil {
			return errw.Wrap(err, "unable to parse hosts from output config")
		}
	}

	reporterHosts := hostsCfg{}
	if reporterCfg != nil {
		if err := reporterCfg.Unpack(&reporterHosts); err != nil {
			return errw.Wrap(err, "unable to parse hosts from reporter config")
		}
	}

	if len(outputHosts.Hosts) == 0 && len(reporterHosts.Hosts) == 0 {
		return nil
	}

	// Give precedence to reporter hosts over output hosts
	var newHostsCfg *common.Config
	var err error
	if len(reporterHosts.Hosts) > 0 {
		newHostsCfg, err = common.NewConfigFrom(reporterHosts.Hosts)
	} else {
		newHostsCfg, err = common.NewConfigFrom(outputHosts.Hosts)
	}
	if err != nil {
		return errw.Wrap(err, "unable to make config from new hosts")
	}

	if err := merged.SetChild("hosts", -1, newHostsCfg); err != nil {
		return errw.Wrap(err, "unable to set new hosts into merged config")
	}
	return nil
}
