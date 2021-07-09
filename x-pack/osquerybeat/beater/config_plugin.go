// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
)

const (
	configName           = "osq_config"
	scheduleSplayPercent = 10
)

type ConfigPlugin struct {
	log *logp.Logger

	mx sync.RWMutex

	queriesCount int

	// A map that allows to look up the original query by name for the column types resolution
	queryMap map[string]query

	// Packs
	packs map[string]pack

	// Raw config bytes cached
	configString string
}

func NewConfigPlugin(log *logp.Logger) *ConfigPlugin {
	p := &ConfigPlugin{
		log: log.With("ctx", "config"),
	}

	return p
}

func (p *ConfigPlugin) Set(inputs []config.InputConfig) {
	p.mx.Lock()
	defer p.mx.Unlock()

	p.set(inputs)
}

func (p *ConfigPlugin) Count() int {
	return p.queriesCount
}

func (p *ConfigPlugin) ResolveName(name string) (sql string, ok bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	sc, ok := p.queryMap[name]

	return sc.Query, ok
}

func (p *ConfigPlugin) GenerateConfig(ctx context.Context) (map[string]string, error) {
	p.log.Debug("configPlugin GenerateConfig is called")

	p.mx.Lock()
	defer p.mx.Unlock()

	c, err := p.render()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		configName: c,
	}, nil
}

type query struct {
	Query    string `json:"query"`
	Interval int    `json:"interval,omitempty"`
	Platform string `json:"platform,omitempty"`
	Version  string `json:"version,omitempty"`
	Snapshot bool   `json:"snapshot,omitempty"`
}

type pack struct {
	Queries map[string]query `json:"queries,omitempty"`
}

type osqueryConfig struct {
	Options map[string]interface{} `json:"options"`
	Packs   map[string]pack        `json:"packs,omitempty"`
}

func newOsqueryConfig(packs map[string]pack) osqueryConfig {
	return osqueryConfig{
		Options: map[string]interface{}{
			"schedule_splay_percent": scheduleSplayPercent,
		},
		Packs: packs,
	}
}

func (c osqueryConfig) render() ([]byte, error) {
	return json.MarshalIndent(c, "", "    ")
}

func (p *ConfigPlugin) render() (string, error) {
	if p.configString == "" {
		raw, err := newOsqueryConfig(p.packs).render()
		if err != nil {
			return "", err
		}
		p.configString = string(raw)
	}

	return p.configString, nil
}

func (p *ConfigPlugin) set(inputs []config.InputConfig) {
	p.configString = ""
	queriesCount := 0
	p.queryMap = make(map[string]query)
	p.packs = make(map[string]pack)
	for _, input := range inputs {
		pack := pack{
			Queries: make(map[string]query),
		}
		for _, stream := range input.Streams {
			id := "pack_" + input.Name + "_" + stream.ID
			query := query{
				Query:    stream.Query,
				Interval: stream.Interval,
				Platform: stream.Platform,
				Version:  stream.Version,
				Snapshot: true, // enforce snapshot for all queries
			}
			p.queryMap[id] = query
			pack.Queries[stream.ID] = query
			queriesCount++
		}
		p.packs[input.Name] = pack
	}
	p.queriesCount = queriesCount
}
