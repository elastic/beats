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

package add_remote_metadata

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const name = "add_remote_metadata"

func init() {
	// We cannot use this as a JS plugin as it is stateful and includes a Close method.
	processors.RegisterPlugin(name, New)
}

var (
	// ErrNoMatch is returned when the event doesn't contain any of the fields
	// specified in match_pids.
	ErrNoMatch = errors.New("none of the fields in match_keys found in the event")

	// ErrNoData is returned when metadata for a metadata can't be collected.
	ErrNoData = errors.New("metadata not found")

	instanceID atomic.Uint32
)

type addRemoteMetadata struct {
	config   config
	mappings mapstr.M
	src      MetadataGetter
	cancel   context.CancelFunc
	log      *logp.Logger
}

// Resulting processor implements `Close()` to release the cache resources.
func New(cfg *conf.C) (beat.Processor, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the %s configuration: %w", name, err)
	}
	new, ok := providers[config.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", config.Provider)
	}
	src, cancel, err := new(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the %s configuration for provider %s: %w", name, config.Provider, err)
	}

	// Logging (each processor instance has a unique ID).
	id := int(instanceID.Inc())
	log := logp.NewLogger(name).With("instance_id", id)

	mappings, err := config.getMappings()
	if err != nil {
		return nil, fmt.Errorf("error unpacking %v.target_fields: %w", name, err)
	}

	p := addRemoteMetadata{
		config:   config,
		src:      src,
		cancel:   cancel,
		mappings: mappings,
		log:      log,
	}
	return &p, nil
}

// MetadataGetter is the interface implemented by metadata providers.
type MetadataGetter interface {
	GetMetadata(key string) (interface{}, error)
}

var providers = map[string]func(*conf.C) (MetadataGetter, context.CancelFunc, error){
	"map": newMapProvider,
}

// mapProvider is a simple provider based on a static map look-up.
type mapProvider map[string]interface{}

func newMapProvider(cfg *conf.C) (MetadataGetter, context.CancelFunc, error) {
	var c struct {
		Values mapProvider `config:"metadata"`
	}
	err := cfg.Unpack(&c)
	if err != nil {
		return nil, noop, err
	}
	return c.Values, noop, nil
}

// GetMetadata implements the metadataGetter interface.
func (p mapProvider) GetMetadata(k string) (interface{}, error) {
	m, ok := p[k]
	if !ok {
		return nil, nil
	}
	return m, nil
}

// noop is a no-op context.CancelFunc.
func noop() {}

// Run enriches the given event with the host metadata.
func (p *addRemoteMetadata) Run(event *beat.Event) (*beat.Event, error) {
	for _, k := range p.config.MatchKeys {
		result, err := p.enrich(event, k)
		if err != nil {
			switch {
			case errors.Is(err, mapstr.ErrKeyNotFound):
				continue
			case errors.Is(err, ErrNoData):
				return event, err
			default:
				return event, fmt.Errorf("error applying %s processor: %w", name, err)
			}
		}
		if result != nil {
			event = result
		}
		return event, nil
	}
	if p.config.IgnoreMissing {
		return event, nil
	}
	return event, ErrNoMatch
}

func (p *addRemoteMetadata) enrich(event *beat.Event, key string) (result *beat.Event, err error) {
	v, err := event.GetValue(key)
	if err != nil {
		return nil, err
	}
	k, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("key field '%s' not a string: %T", key, v)
	}

	meta, err := p.src.GetMetadata(k)
	if err != nil {
		return nil, fmt.Errorf("%w for '%s': %w", ErrNoData, k, err)
	}
	if meta == nil {
		return nil, fmt.Errorf("%w for '%s'", ErrNoData, k)
	}
	if m, ok := meta.(map[string]interface{}); ok {
		meta = mapstr.M(m)
	}
	result = event.Clone()
	switch meta := meta.(type) {
	case mapstr.M:
		if len(meta) == 0 {
			return nil, fmt.Errorf("%w for '%s'", ErrNoData, k)
		}
		for dst, v := range p.mappings {
			src, ok := v.(string)
			if !ok {
				// Should never happen, as source is generated by Config.prepareMappings()
				return nil, errors.New("source is not a string")
			}
			if !p.config.OverwriteKeys {
				if _, err := result.GetValue(dst); err == nil {
					return nil, fmt.Errorf("target field '%s' already exists and overwrite_keys is false", dst)
				}
			}

			val, err := meta.GetValue(src)
			if err != nil {
				// skip missing values
				continue
			}

			if _, err = result.PutValue(dst, val); err != nil {
				return nil, err
			}
		}
	default:
		if p.config.Target == "" {
			return nil, fmt.Errorf("no target field specified for non-object metadata: %v", meta)
		}
		if _, err = result.PutValue(p.config.Target, meta); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (p *addRemoteMetadata) Close() error {
	p.cancel()
	return nil
}

// String returns the processor representation formatted as a string
func (p *addRemoteMetadata) String() string {
	return fmt.Sprintf("%v=[match_pids=%v, mappings=%v, ignore_missing=%v, overwrite_fields=%v]",
		name, p.config.MatchKeys, p.mappings, p.config.IgnoreMissing, p.config.OverwriteKeys)
}
