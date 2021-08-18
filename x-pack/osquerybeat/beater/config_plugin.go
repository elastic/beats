// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
)

const (
	configName           = "osq_config"
	scheduleSplayPercent = 10
	maxECSMappingDepth   = 25 // Max ECS dot delimited key path, that is sufficient for the current ECS mapping

	keyField = "field"
	keyValue = "value"
)

var (
	ErrECSMappingIsInvalid = errors.New("ECS mapping is invalid")
	ErrECSMappingIsTooDeep = errors.New("ECS mapping is too deep")
)

type QueryInfo struct {
	QueryConfig query
	ECSMapping  ecs.Mapping
	Namespace   string
}

type queryInfoMap map[string]QueryInfo

type ConfigPlugin struct {
	log *logp.Logger

	mx sync.RWMutex

	queriesCount int

	// A map that allows to look up the queryInfo by query name
	queryInfoMap queryInfoMap

	// This map holds the new queries info before the configuration requested from the plugin.
	// This replaces the queryInfoMap upon receiving GenerateConfig call from osqueryd.
	// Until we receive this call from osqueryd we should use the previously set mapping,
	// otherwise we potentially could receive the query result for the old queries before osqueryd requested the new configuration
	// and we would not be able to resolve types or ECS mapping or the namespace.
	newQueryInfoMap queryInfoMap

	// Packs
	packs map[string]pack

	// Raw config bytes cached
	configString string
}

func NewConfigPlugin(log *logp.Logger) *ConfigPlugin {
	p := &ConfigPlugin{
		log:          log.With("ctx", "config"),
		queryInfoMap: make(queryInfoMap),
	}

	return p
}

func (p *ConfigPlugin) Set(inputs []config.InputConfig) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	return p.set(inputs)
}

func (p *ConfigPlugin) Count() int {
	return p.queriesCount
}

func (p *ConfigPlugin) LookupQueryInfo(name string) (qi QueryInfo, ok bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	qi, ok = p.queryInfoMap[name]
	return qi, ok
}

func (p *ConfigPlugin) GenerateConfig(ctx context.Context) (map[string]string, error) {
	p.log.Debug("configPlugin GenerateConfig is called")

	p.mx.Lock()
	defer p.mx.Unlock()

	c, err := p.render()
	if err != nil {
		return nil, err
	}

	// replace the query info map
	if p.newQueryInfoMap != nil {
		p.queryInfoMap = p.newQueryInfoMap
		p.newQueryInfoMap = nil
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
	Discovery []string         `json:"discovery,omitempty"`
	Platform  string           `json:"platform,omitempty"`
	Version   string           `json:"version,omitempty"`
	Queries   map[string]query `json:"queries,omitempty"`
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

func (p *ConfigPlugin) set(inputs []config.InputConfig) error {
	var err error

	p.configString = ""
	queriesCount := 0
	newQueryInfoMap := make(map[string]QueryInfo)
	// queryMap := make(map[string]query)
	// ecsMap := make(map[string]ecs.Mapping)
	p.packs = make(map[string]pack)
	for _, input := range inputs {
		pack := pack{
			Queries:   make(map[string]query),
			Platform:  input.Platform,
			Version:   input.Version,
			Discovery: input.Discovery,
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
			var ecsm ecs.Mapping
			if len(stream.ECSMapping) > 0 {
				ecsm, err = flattenECSMapping(stream.ECSMapping)
				if err != nil {
					return err
				}
			}
			newQueryInfoMap[id] = QueryInfo{
				QueryConfig: query,
				ECSMapping:  ecsm,
				Namespace:   input.Datastream.Namespace,
			}
			pack.Queries[stream.ID] = query
			queriesCount++
		}
		p.packs[input.Name] = pack
	}
	p.newQueryInfoMap = newQueryInfoMap
	p.queriesCount = queriesCount
	return nil
}

// Due to current configuration passing between the agent and beats the keys that contain dots (".")
// are split into the nested tree-like structure.
// This converts this dynamic map[string]interface{} tree into strongly typed flat map.
func flattenECSMapping(m map[string]interface{}) (ecs.Mapping, error) {
	ecsm := make(ecs.Mapping)
	for k, v := range m {
		if "" == strings.TrimSpace(k) {
			return nil, fmt.Errorf("empty key at depth 0: %w", ErrECSMappingIsInvalid)
		}
		err := traverseTree(0, ecsm, []string{k}, v)
		if err != nil {
			return nil, err
		}
	}
	return ecsm, nil
}

func traverseTree(depth int, ecsm ecs.Mapping, path []string, v interface{}) error {

	if path[len(path)-1] == keyField {
		if s, ok := v.(string); ok {
			if len(path) == 1 {
				return fmt.Errorf("unexpected top level key '%s': %w", keyField, ErrECSMappingIsInvalid)
			}
			if "" == strings.TrimSpace(s) {
				return fmt.Errorf("empty field value: %w", ErrECSMappingIsInvalid)
			}
			ecsm[strings.Join(path[:len(path)-1], ".")] = ecs.MappingInfo{
				Field: s,
			}
		} else {
			if v == nil {
				return fmt.Errorf("mapping to nil field: %w", ErrECSMappingIsInvalid)
			} else {
				return fmt.Errorf("unexpected field type %T: %w", v, ErrECSMappingIsInvalid)
			}
		}
		return nil
	} else if path[len(path)-1] == keyValue {
		if len(path) == 1 {
			return fmt.Errorf("unexpected top level key '%s': %w", keyValue, ErrECSMappingIsInvalid)
		}
		ecsm[strings.Join(path[:len(path)-1], ".")] = ecs.MappingInfo{
			Value: v,
		}
		return nil
	} else if m, ok := v.(map[string]interface{}); ok {
		if depth < maxECSMappingDepth {
			for k, v := range m {
				if "" == strings.TrimSpace(k) {
					return fmt.Errorf("empty key at depth %d: %w", depth+1, ErrECSMappingIsInvalid)
				}
				err := traverseTree(depth+1, ecsm, append(path, k), v)
				if err != nil {
					return err
				}
			}
		} else {
			return ErrECSMappingIsTooDeep
		}
	}
	return nil
}
