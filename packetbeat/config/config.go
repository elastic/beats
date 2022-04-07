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

package config

import (
	"errors"
	"time"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/processors"
	"github.com/elastic/beats/v8/packetbeat/procs"
)

type Config struct {
	Interfaces      InterfacesConfig          `config:"interfaces"`
	Flows           *Flows                    `config:"flows"`
	Protocols       map[string]*common.Config `config:"protocols"`
	ProtocolsList   []*common.Config          `config:"protocols"`
	Procs           procs.ProcsConfig         `config:"procs"`
	IgnoreOutgoing  bool                      `config:"ignore_outgoing"`
	ShutdownTimeout time.Duration             `config:"shutdown_timeout"`
}

// FromStatic initializes a configuration given a common.Config
func (c Config) FromStatic(cfg *common.Config) (Config, error) {
	err := cfg.Unpack(&c)
	if err != nil {
		return c, err
	}
	return c, nil
}

// ICMP returns the ICMP configuration
func (c Config) ICMP() (*common.Config, error) {
	var icmp *common.Config
	if c.Protocols["icmp"].Enabled() {
		icmp = c.Protocols["icmp"]
	}

	for _, cfg := range c.ProtocolsList {
		info := struct {
			Type string `config:"type" validate:"required"`
		}{}

		if err := cfg.Unpack(&info); err != nil {
			return nil, err
		}

		if info.Type != "icmp" {
			continue
		}

		if icmp != nil {
			return nil, errors.New("more than one icmp configuration found")
		}

		icmp = cfg
	}
	return icmp, nil
}

type InterfacesConfig struct {
	Device                string   `config:"device"`
	Type                  string   `config:"type"`
	File                  string   `config:"file"`
	WithVlans             bool     `config:"with_vlans"`
	BpfFilter             string   `config:"bpf_filter"`
	Snaplen               int      `config:"snaplen"`
	BufferSizeMb          int      `config:"buffer_size_mb"`
	EnableAutoPromiscMode bool     `config:"auto_promisc_mode"`
	InternalNetworks      []string `config:"internal_networks"`
	TopSpeed              bool
	Dumpfile              string
	OneAtATime            bool
	Loop                  int
}

type Flows struct {
	Enabled       *bool                   `config:"enabled"`
	Timeout       string                  `config:"timeout"`
	Period        string                  `config:"period"`
	EventMetadata common.EventMetadata    `config:",inline"`
	Processors    processors.PluginConfig `config:"processors"`
	KeepNull      bool                    `config:"keep_null"`
	// Index is used to overwrite the index where flows are published
	Index string `config:"index"`
}

type ProtocolCommon struct {
	Ports              []int         `config:"ports"`
	SendRequest        bool          `config:"send_request"`
	SendResponse       bool          `config:"send_response"`
	TransactionTimeout time.Duration `config:"transaction_timeout"`
}

func (f *Flows) IsEnabled() bool {
	return f != nil && (f.Enabled == nil || *f.Enabled)
}
