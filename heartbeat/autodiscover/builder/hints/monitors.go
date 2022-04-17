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

	"github.com/menderesk/go-ucfg"

	"github.com/menderesk/beats/v7/libbeat/autodiscover"
	"github.com/menderesk/beats/v7/libbeat/autodiscover/builder"
	"github.com/menderesk/beats/v7/libbeat/autodiscover/template"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/bus"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddBuilder("hints", NewHeartbeatHints)
}

const (
	montype    = "type"
	schedule   = "schedule"
	hosts      = "hosts"
	processors = "processors"
)

type heartbeatHints struct {
	config *config
	logger *logp.Logger
}

// NewHeartbeatHints builds a heartbeat hints builder
func NewHeartbeatHints(cfg *common.Config) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %v", err)
	}

	return &heartbeatHints{config, logp.NewLogger("hints.builder")}, nil
}

// Create config based on input hints in the bus event
func (hb *heartbeatHints) CreateConfig(event bus.Event, options ...ucfg.Option) []*common.Config {
	var hints common.MapStr
	hIface, ok := event["hints"]
	if ok {
		hints, _ = hIface.(common.MapStr)
	}

	monitorConfig := hb.getRawConfigs(hints)

	// If explicty disabled, return nothing
	if builder.IsDisabled(hints, hb.config.Key) {
		hb.logger.Warnf("heartbeat config disabled by hint: %+v", event)
		return []*common.Config{}
	}

	port, _ := common.TryToInt(event["port"])

	host, _ := event["host"].(string)
	if host == "" {
		return []*common.Config{}
	}

	if monitorConfig != nil {
		configs := []*common.Config{}
		for _, cfg := range monitorConfig {
			if config, err := common.NewConfigFrom(cfg); err == nil {
				configs = append(configs, config)
			}
		}
		hb.logger.Debugf("generated config %+v", configs)
		// Apply information in event to the template to generate the final config
		return template.ApplyConfigTemplate(event, configs)
	}

	tempCfg := common.MapStr{}
	monitors := builder.GetHintsAsList(hints, hb.config.Key)

	var configs []*common.Config
	for _, monitor := range monitors {
		// If a monitor doesn't have a schedule associated with it then default it.
		if _, ok := monitor[schedule]; !ok {
			monitor[schedule] = hb.config.DefaultSchedule
		}

		if procs := hb.getProcessors(monitor); len(procs) != 0 {
			monitor[processors] = procs
		}

		h := hb.getHostsWithPort(monitor, port)
		monitor[hosts] = h

		config, err := common.NewConfigFrom(monitor)
		if err != nil {
			hb.logger.Debugf("unable to create config from MapStr %+v", tempCfg)
			return []*common.Config{}
		}
		hb.logger.Debugf("hints.builder", "generated config %+v", config)
		configs = append(configs, config)
	}

	// Apply information in event to the template to generate the final config
	return template.ApplyConfigTemplate(event, configs)
}

func (hb *heartbeatHints) getType(hints common.MapStr) common.MapStr {
	return builder.GetHintMapStr(hints, hb.config.Key, montype)
}

func (hb *heartbeatHints) getSchedule(hints common.MapStr) []string {
	return builder.GetHintAsList(hints, hb.config.Key, schedule)
}

func (hb *heartbeatHints) getRawConfigs(hints common.MapStr) []common.MapStr {
	return builder.GetHintAsConfigs(hints, hb.config.Key)
}

func (hb *heartbeatHints) getProcessors(hints common.MapStr) []common.MapStr {
	return builder.GetConfigs(hints, "", "processors")
}

func (hb *heartbeatHints) getHostsWithPort(hints common.MapStr, port int) []string {
	var result []string
	thosts := builder.GetHintAsList(hints, "", hosts)
	// Only pick hosts that have ${data.port} or the port on current event. This will make
	// sure that incorrect meta mapping doesn't happen
	for _, h := range thosts {
		if strings.Contains(h, "data.port") || strings.Contains(h, fmt.Sprintf(":%d", port)) ||
			// Use the event that has no port config if there is a ${data.host}:9090 like input
			(port == 0 && strings.Contains(h, "data.host")) {
			result = append(result, h)
		} else if port == 0 && !strings.Contains(h, ":") {
			// For ICMP like use cases allow only host to be passed if there is no port
			result = append(result, h)
		} else {
			hb.logger.Warn("unable to frame a host from input host: %s", h)
		}
	}

	if len(thosts) > 0 && len(result) == 0 {
		hb.logger.Debugf("no hosts selected for port %d with hints: %+v", port, thosts)
		return nil
	}

	return result
}
