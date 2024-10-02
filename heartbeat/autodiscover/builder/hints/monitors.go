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

	"github.com/elastic/go-ucfg"

	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-autodiscover/utils"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	schedule   = "schedule"
	hosts      = "hosts"
	processors = "processors"
	scheme     = "type"
)

type heartbeatHints struct {
	config *config
	logger *logp.Logger
}

// InitializeModule initializes this module.
func InitializeModule() {
	err := autodiscover.Registry.AddBuilder("hints", NewHeartbeatHints)
	if err != nil {
		logp.Error(fmt.Errorf("could not add `hints` builder"))
	}
}

// NewHeartbeatHints builds a heartbeat hints builder
func NewHeartbeatHints(cfg *conf.C) (autodiscover.Builder, error) {
	config := defaultConfig()
	err := cfg.Unpack(config)

	if err != nil {
		return nil, fmt.Errorf("unable to unpack hints config due to error: %w", err)
	}

	return &heartbeatHints{config, logp.L()}, nil
}

// Create config based on input hints in the bus event
func (hb *heartbeatHints) CreateConfig(event bus.Event, options ...ucfg.Option) []*conf.C {
	var (
		hints    mapstr.M
		podEvent bool
	)

	hIface, ok := event["hints"]
	if ok {
		hints, _ = hIface.(mapstr.M)
	}

	monitorConfig := hb.getRawConfigs(hints)

	// If explicty disabled, return nothing
	if utils.IsDisabled(hints, hb.config.Key) {
		hb.logger.Warnf("heartbeat config disabled by hint: %+v", event)
		return []*conf.C{}
	}

	port, ok := common.TryToInt(event["port"])
	if !ok {
		podEvent = true
	}

	host, _ := event["host"].(string)
	if host == "" {
		return []*conf.C{}
	}

	if monitorConfig != nil {
		configs := []*conf.C{}
		for _, cfg := range monitorConfig {
			if config, err := conf.NewConfigFrom(cfg); err == nil {
				configs = append(configs, config)
			}
		}
		hb.logger.Debugf("generated config %+v", configs)
		// Apply information in event to the template to generate the final config
		return template.ApplyConfigTemplate(event, configs)
	}

	tempCfg := mapstr.M{}
	monitors := utils.GetHintsAsList(hints, hb.config.Key)

	configs := make([]*conf.C, 0, len(monitors))
	for _, monitor := range monitors {
		// If a monitor doesn't have a schedule associated with it then default it.
		if _, ok := monitor[schedule]; !ok {
			monitor[schedule] = hb.config.DefaultSchedule
		}

		if procs := hb.getProcessors(monitor); len(procs) != 0 {
			monitor[processors] = procs
		}

		h, err := hb.getHostsWithPort(monitor, port, podEvent)
		if err != nil {
			hb.logger.Warnf("unable to find valid hosts for %+v: %w", monitor, err)
			continue
		}

		monitor[hosts] = h

		config, err := conf.NewConfigFrom(monitor)
		if err != nil {
			hb.logger.Debugf("unable to create config from MapStr %+v", tempCfg)
			return []*conf.C{}
		}
		hb.logger.Debugf("hints.builder", "generated config %+v", config)
		configs = append(configs, config)
	}

	// Apply information in event to the template to generate the final config
	return template.ApplyConfigTemplate(event, configs)
}

func (hb *heartbeatHints) getRawConfigs(hints mapstr.M) []mapstr.M {
	return utils.GetHintAsConfigs(hints, hb.config.Key)
}

func (hb *heartbeatHints) getProcessors(hints mapstr.M) []mapstr.M {
	return utils.GetConfigs(hints, "", "processors")
}

func (hb *heartbeatHints) getHostsWithPort(hints mapstr.M, port int, podEvent bool) ([]string, error) {
	thosts := utils.GetHintAsList(hints, "", hosts)
	mType := utils.GetHintString(hints, "", scheme)

	// We can't reliable detect duplicated monitors since we don't have all ports/hosts,
	// relying on runner deduping monitors, see https://github.com/elastic/beats/pull/29041
	hostSet := map[string]struct{}{}
	for _, h := range thosts {
		if mType == "icmp" && strings.Contains(h, ":") {
			hb.logger.Warnf("ICMP scheme does not support port specification: %s", h)
			continue
		} else if strings.Contains(h, "${data.port}") && podEvent {
			// Pod events don't contain port metadata, skip
			continue
		}

		hostSet[h] = struct{}{}
	}

	if len(hostSet) == 0 {
		return nil, fmt.Errorf("no hosts selected for port %d with hints: %+v", port, thosts)
	}

	result := make([]string, 0, len(hostSet))
	for host := range hostSet {
		result = append(result, host)
	}

	return result, nil
}
