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

package jolokia

import (
	"time"

	"github.com/menderesk/beats/v7/libbeat/autodiscover/template"
	"github.com/menderesk/beats/v7/libbeat/common"
)

var (
	defaultInterval     = 10 * time.Second
	defaultProbeTimeout = 1 * time.Second
	defaultGracePeriod  = 30 * time.Second
)

// Config for Jolokia Discovery autodiscover provider
type Config struct {
	// List of network interfaces to use for discovery probes
	Interfaces []InterfaceConfig `config:"interfaces,replace" validate:"nonzero"`

	Builders  []*common.Config        `config:"builders"`
	Appenders []*common.Config        `config:"appenders"`
	Templates template.MapperSettings `config:"templates"`
}

// InterfaceConfig is the configuration for a network interface used for probes
type InterfaceConfig struct {
	// Name of the interface
	Name string `config:"name" validate:"required"`

	// Time between discovery probes
	Interval time.Duration `config:"interval" validate:"positive,nonzero"`

	// Time to wait till a response to a probe arrives
	ProbeTimeout time.Duration `config:"probe_timeout" validate:"positive,nonzero"`

	// Time since an instance is last seen and is considered removed
	GracePeriod time.Duration `config:"grace_period" validate:"positive,nonzero"`
}

// Unpack implements the config unpacker for interface configs
func (c *InterfaceConfig) Unpack(from *common.Config) error {
	// Overriding Unpack just to set defaults
	// See https://github.com/menderesk/go-ucfg/issues/104
	type tmpConfig InterfaceConfig
	defaults := defaultInterfaceConfig()
	tmp := tmpConfig(defaults)

	err := from.Unpack(&tmp)
	if err != nil {
		return err
	}

	*c = InterfaceConfig(tmp)
	return nil
}

func defaultInterfaceConfig() InterfaceConfig {
	return InterfaceConfig{
		Interval:     defaultInterval,
		ProbeTimeout: defaultProbeTimeout,
		GracePeriod:  defaultGracePeriod,
	}
}

func defaultConfig() Config {
	anyInterface := defaultInterfaceConfig()
	anyInterface.Name = "any"
	return Config{
		Interfaces: []InterfaceConfig{
			anyInterface,
		},
	}
}
