// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v8/x-pack/osquerybeat/internal/ecs"
)

const (
	configName                  = "osq_config"
	defaultScheduleSplayPercent = 10
	maxECSMappingDepth          = 25 // Max ECS dot delimited key path, that is sufficient for the current ECS mapping

	keyField = "field"
	keyValue = "value"
)

var (
	ErrECSMappingIsInvalid = errors.New("ECS mapping is invalid")
	ErrECSMappingIsTooDeep = errors.New("ECS mapping is too deep")
)

type QueryInfo struct {
	Query      string
	ECSMapping ecs.Mapping
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

	// Datastream namesapces map that allows to lookup the namespace per query.
	// The datastream namespaces map is handled separatelly from query info
	// because if we delay updating it until the osqueryd config refresh (up to 1 minute, the way we do with queryinfo)
	// we could be sending data into the datastream with namespace that we don't have permissions meanwhile
	namespaces map[string]string

	// Osquery configuration
	osqueryConfig *config.OsqueryConfig

	// Raw config bytes cached
	configString string

	// One common namespace from the first input as of 7.16
	// This is used to ad-hoc queries results over GetNamespace API
	namespace string
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
	p.mx.RLock()
	defer p.mx.RUnlock()

	return p.queriesCount
}

func (p *ConfigPlugin) LookupQueryInfo(name string) (qi QueryInfo, ok bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	qi, ok = p.queryInfoMap[name]
	return qi, ok
}

func (p *ConfigPlugin) LookupNamespace(name string) (ns string, ok bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	ns, ok = p.namespaces[name]
	return ns, ok
}

func (p *ConfigPlugin) GetNamespace() string {
	p.mx.RLock()
	defer p.mx.RUnlock()
	return p.namespace
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

	p.log.Debug("Osqueryd configuration:", c)

	return map[string]string{
		configName: c,
	}, nil
}

func newOsqueryConfig(osqueryConfig *config.OsqueryConfig) *config.OsqueryConfig {
	if osqueryConfig == nil {
		osqueryConfig = &config.OsqueryConfig{}
	}
	if osqueryConfig.Options == nil {
		osqueryConfig.Options = make(map[string]interface{})
	}
	const scheduleSplayPercentKey = "schedule_splay_percent"
	if _, ok := osqueryConfig.Options[scheduleSplayPercentKey]; !ok {
		osqueryConfig.Options[scheduleSplayPercentKey] = defaultScheduleSplayPercent
	}
	return osqueryConfig
}

func (p *ConfigPlugin) render() (string, error) {
	if p.configString == "" {
		raw, err := newOsqueryConfig(p.osqueryConfig).Render()
		if err != nil {
			return "", err
		}
		p.configString = string(raw)
	}

	return p.configString, nil
}

func (p *ConfigPlugin) set(inputs []config.InputConfig) (err error) {

	p.configString = ""
	p.namespace = ""

	queriesCount := 0
	osqueryConfig := &config.OsqueryConfig{}
	newQueryInfoMap := make(map[string]QueryInfo)
	namespaces := make(map[string]string)

	// Set the members if no errors
	defer func() {
		if err != nil {
			return
		}
		p.osqueryConfig = osqueryConfig
		p.newQueryInfoMap = newQueryInfoMap
		p.namespaces = namespaces
		p.queriesCount = queriesCount
	}()

	// Return if no inputs, all the members will be reset by deferred call above
	if len(inputs) == 0 {
		return nil
	}

	// Read namespace from the first input as of 7.16
	p.namespace = inputs[0].Datastream.Namespace
	if p.namespace == "" {
		p.namespace = config.DefaultNamespace
	}

	// Since 7.16 version only one integration/input is expected
	// The inputs[0].Osquery can be nil if this is pre 7.16 integration configuration
	if inputs[0].Osquery != nil {
		osqueryConfig = inputs[0].Osquery
	}

	// Common code to register query with lookup maps, enforce snapshot and increment queries count
	registerQuery := func(name, ns string, qi config.Query) (config.Query, error) {
		var ecsm ecs.Mapping
		ecsm, err = flattenECSMapping(qi.ECSMapping)
		if err != nil {
			return qi, err
		}

		newQueryInfoMap[name] = QueryInfo{
			Query:      qi.Query,
			ECSMapping: ecsm,
		}
		namespaces[name] = p.namespace
		queriesCount++

		qi.Snapshot = true
		return qi, nil
	}

	// Iterate osquery configuration's scheduled queries, add flattened ECS mappings to lookup map
	for name, qi := range osqueryConfig.Schedule {
		qi, err = registerQuery(name, p.namespace, qi)
		if err != nil {
			return err
		}
		osqueryConfig.Schedule[name] = qi
	}

	// Iterate osquery configuration's packs queries, add flattened ECS mappings to lookup map
	for packName, pack := range osqueryConfig.Packs {
		for name, qi := range pack.Queries {
			qi, err = registerQuery(getPackQueryName(packName, name), p.namespace, qi)
			if err != nil {
				return err
			}
			pack.Queries[name] = qi
		}
	}

	// Iterate inputs for Osquery configuration for backwards compatibility
	for _, input := range inputs {
		pack := config.Pack{
			Queries:   make(map[string]config.Query),
			Platform:  input.Platform,
			Version:   input.Version,
			Discovery: input.Discovery,
		}
		for _, stream := range input.Streams {
			qi := config.Query{
				Query:      stream.Query,
				Interval:   stream.Interval,
				Platform:   stream.Platform,
				Version:    stream.Version,
				ECSMapping: stream.ECSMapping,
			}

			qi, err = registerQuery(getPackQueryName(input.Name, stream.ID), p.namespace, qi)
			if err != nil {
				return err
			}
			pack.Queries[stream.ID] = qi
		}

		if len(pack.Queries) != 0 {
			if osqueryConfig.Packs == nil {
				osqueryConfig.Packs = make(map[string]config.Pack)
			}
			osqueryConfig.Packs[input.Name] = pack
		}
	}

	return nil
}

func getPackQueryName(packName, queryName string) string {
	return "pack_" + packName + "_" + queryName
}

// Due to current configuration passing between the agent and beats the keys that contain dots (".")
// are split into the nested tree-like structure.
// This converts this dynamic map[string]interface{} tree into strongly typed flat map.
func flattenECSMapping(m map[string]interface{}) (ecs.Mapping, error) {
	if m == nil {
		return nil, nil
	}
	ecsm := make(ecs.Mapping)
	for k, v := range m {
		if strings.TrimSpace(k) == "" {
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
			if strings.TrimSpace(s) == "" {
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
				if strings.TrimSpace(k) == "" {
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
