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
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/packetbeat/procs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var errFanoutGroupAFPacketOnly = errors.New("fanout_group is only valid with af_packet type")

type Config struct {
	Interface          *InterfaceConfig   `config:"interfaces"`
	Interfaces         []InterfaceConfig  `config:"interfaces"`
	Flows              *Flows             `config:"flows"`
	Protocols          map[string]*conf.C `config:"protocols"`
	ProtocolsList      []*conf.C          `config:"protocols"`
	Procs              procs.ProcsConfig  `config:"procs"`
	IgnoreOutgoing     bool               `config:"ignore_outgoing"`
	ShutdownTimeout    time.Duration      `config:"shutdown_timeout"`
	OverwritePipelines bool               `config:"overwrite_pipelines"` // Only used by standalone Packetbeat.
}

// FromStatic initializes a configuration given a config.C
func (c Config) FromStatic(cfg *conf.C) (Config, error) {
	err := cfg.Unpack(&c)
	if err != nil {
		return c, err
	}
	iface, err := cfg.Child("interfaces", -1)
	if err == nil {
		if !iface.IsArray() {
			c.Interfaces = []InterfaceConfig{*c.Interface}
		}
	}
	c.Interface = nil
	counts := make(map[string]int)
	for i, iface := range c.Interfaces {
		name := iface.Device
		if name == "" {
			if runtime.GOOS == "linux" {
				name = "any"
			} else {
				name = "default_route"
			}
		}
		counts[name]++
		if 0 < c.Interfaces[i].PollDefaultRoute && c.Interfaces[i].PollDefaultRoute < time.Second {
			c.Interfaces[i].PollDefaultRoute = time.Second
		}
	}
	for n, c := range counts {
		if c == 1 {
			delete(counts, n)
		}
	}
	if len(counts) != 0 {
		dups := make([]string, 0, len(counts))
		for n := range counts {
			dups = append(dups, n)
		}
		return c, fmt.Errorf("duplicated device configurations: %s", strings.Join(dups, ", "))
	}
	return c, nil
}

// ICMP returns the ICMP configuration
func (c Config) ICMP() (*conf.C, error) {
	var icmp *conf.C
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

type InterfaceConfig struct {
	Device                string        `config:"device"`
	PollDefaultRoute      time.Duration `config:"poll_default_route"`
	MetricsInterval       time.Duration `config:"metrics_interval"`
	Type                  string        `config:"type"`
	File                  string        `config:"file"`
	WithVlans             bool          `config:"with_vlans"`
	BpfFilter             string        `config:"bpf_filter"`
	Snaplen               int           `config:"snaplen"`
	BufferSizeMb          int           `config:"buffer_size_mb"`
	EnableAutoPromiscMode bool          `config:"auto_promisc_mode"`
	InternalNetworks      []string      `config:"internal_networks"`
	FanoutGroup           *uint16       `config:"fanout_group"` // Fanout group ID for AF_PACKET.
	TopSpeed              bool
	Dumpfile              string // Dumpfile is the basename of pcap dumpfiles. The file names will have a creation time stamp and .pcap extension appended.
	OneAtATime            bool
	Loop                  int
}

type Flows struct {
	Enabled       *bool                   `config:"enabled"`
	Timeout       string                  `config:"timeout"`
	Period        string                  `config:"period"`
	EventMetadata mapstr.EventMetadata    `config:",inline"`
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

func (i InterfaceConfig) Validate() error {
	if i.Type != "af_packet" && i.FanoutGroup != nil {
		return errFanoutGroupAFPacketOnly
	}
	return nil
}
