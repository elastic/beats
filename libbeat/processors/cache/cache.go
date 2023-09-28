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

package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

const name = "cache"

func init() {
	// We cannot use this as a JS plugin as it is stateful and includes a Close method.
	processors.RegisterPlugin(name, New)
}

var (
	// ErrNoMatch is returned when the event doesn't contain the field
	// specified in key_field.
	ErrNoMatch = errors.New("field in key_field not found in the event")

	// ErrNoData is returned when metadata for an event can't be collected.
	ErrNoData = errors.New("metadata not found")

	instanceID atomic.Uint32
)

// cache is a caching enrichment processor.
type cache struct {
	config config
	store  Store
	cancel context.CancelFunc
	log    *logp.Logger
}

// Resulting processor implements `Close()` to release the cache resources.
func New(cfg *conf.C) (beat.Processor, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the %s configuration: %w", name, err)
	}
	// Logging (each processor instance has a unique ID).
	id := int(instanceID.Inc())
	log := logp.NewLogger(name).With("instance_id", id)

	src, cancel, err := getStoreFor(config, log)
	if err != nil {
		return nil, fmt.Errorf("failed to get the store for %s: %w", name, err)
	}

	p := &cache{
		config: config,
		store:  src,
		cancel: cancel,
		log:    log,
	}
	p.log.Infow("initialized cache processor", "details", p)
	return p, nil
}

// getStoreFor returns a backing store for the provided configuration,
// and a context cancellation that releases the cache resource when it
// is no longer required. The cancellation should be called when the
// processor is closed.
func getStoreFor(cfg config, log *logp.Logger) (Store, context.CancelFunc, error) {
	switch {
	case cfg.Store.Memory != nil:
		s, cancel := memStores.get(cfg.Store.Memory.ID, cfg)
		return s, cancel, nil

	case cfg.Store.File != nil:
		err := os.MkdirAll(paths.Resolve(paths.Data, "cache_processor"), 0o700)
		if err != nil {
			return nil, noop, fmt.Errorf("cache processor could not create store directory: %w", err)
		}
		s, cancel := fileStores.get(cfg.Store.File.ID, cfg, log)
		return s, cancel, nil

	default:
		// This should have been caught by config validation.
		return nil, noop, errors.New("no configured store")
	}
}

// noop is a no-op context.CancelFunc.
func noop() {}

// Store is the interface implemented by metadata providers.
type Store interface {
	Put(key string, val any) error
	Get(key string) (any, error)
	Delete(key string) error

	// The string returned from the String method should
	// be the backing store ID. Either "file:<id>" or
	// "memory:<id>".
	fmt.Stringer
}

type CacheEntry struct {
	Key     string    `json:"key"`
	Value   any       `json:"val"`
	Expires time.Time `json:"expires"`
	index   int
}

// Run enriches the given event with the host metadata.
func (p *cache) Run(event *beat.Event) (*beat.Event, error) {
	switch {
	case p.config.Put != nil:
		p.log.Debugw("put", "backend_id", p.store, "config", p.config.Put)
		err := p.putFrom(event)
		if err != nil {
			switch {
			case errors.Is(err, mapstr.ErrKeyNotFound):
				if p.config.IgnoreMissing {
					return event, nil
				}
				return event, err
			}
			return event, fmt.Errorf("error applying %s put processor: %w", name, err)
		}
		return event, nil

	case p.config.Get != nil:
		p.log.Debugw("get", "backend_id", p.store, "config", p.config.Get)
		result, err := p.getFor(event)
		if err != nil {
			switch {
			case errors.Is(err, mapstr.ErrKeyNotFound):
				if p.config.IgnoreMissing {
					return event, nil
				}
			case errors.Is(err, ErrNoData):
				return event, err
			}
			return event, fmt.Errorf("error applying %s get processor: %w", name, err)
		}
		if result != nil {
			return result, nil
		}
		return event, ErrNoMatch

	case p.config.Delete != nil:
		p.log.Debugw("delete", "backend_id", p.store, "config", p.config.Delete)
		err := p.deleteFor(event)
		if err != nil {
			return event, fmt.Errorf("error applying %s delete processor: %w", name, err)
		}
		return event, nil

	default:
		// This should never happen, but we don't need to flag it.
		return event, nil
	}
}

// putFrom takes the configured value from the event and stores it in the cache
// if it exists.
func (p *cache) putFrom(event *beat.Event) error {
	k, err := event.GetValue(p.config.Put.Key)
	if err != nil {
		return err
	}
	key, ok := k.(string)
	if !ok {
		return fmt.Errorf("key field '%s' not a string: %T", p.config.Put.Key, k)
	}
	p.log.Debugw("put", "backend_id", p.store, "key", key)

	val, err := event.GetValue(p.config.Put.Value)
	if err != nil {
		return err
	}

	err = p.store.Put(key, val)
	if err != nil {
		return fmt.Errorf("failed to put '%s' into '%s': %w", key, p.config.Put.Value, err)
	}
	return nil
}

// getFor gets the configured value from the cache for the event and inserts
// it into the configured field if it exists.
func (p *cache) getFor(event *beat.Event) (result *beat.Event, err error) {
	// Check for clobbering.
	dst := p.config.Get.Target
	if !p.config.OverwriteKeys {
		if _, err := event.GetValue(dst); err == nil {
			return nil, fmt.Errorf("target field '%s' already exists and overwrite_keys is false", dst)
		}
	}

	// Get key into store for metadata.
	key := p.config.Get.Key
	v, err := event.GetValue(key)
	if err != nil {
		return nil, err
	}
	k, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("key field '%s' not a string: %T", key, v)
	}
	p.log.Debugw("get", "backend_id", p.store, "key", k)

	// Get metadata...
	meta, err := p.store.Get(k)
	if err != nil {
		return nil, fmt.Errorf("%w for '%s': %w", ErrNoData, k, err)
	}
	if meta == nil {
		return nil, fmt.Errorf("%w for '%s'", ErrNoData, k)
	}
	if m, ok := meta.(map[string]interface{}); ok {
		meta = mapstr.M(m)
	}
	// ... and write it into the event.
	// The implementation of PutValue currently leaves event
	// essentially unchanged in the case of an error (in the
	// case of an @metadata field there may be a mutation,
	// but at most this will be the addition of a Meta field
	// value to event). None of this is documented.
	if _, err = event.PutValue(dst, meta); err != nil {
		return nil, err
	}
	return event, nil
}

// deleteFor deletes the configured value from the cache based on the value of
// the configured key.
func (p *cache) deleteFor(event *beat.Event) error {
	v, err := event.GetValue(p.config.Delete.Key)
	if err != nil {
		return err
	}
	k, ok := v.(string)
	if !ok {
		return fmt.Errorf("key field '%s' not a string: %T", p.config.Delete.Key, v)
	}
	return p.store.Delete(k)
}

func (p *cache) Close() error {
	p.cancel()
	return nil
}

// String returns the processor representation formatted as a string
func (p *cache) String() string {
	switch {
	case p.config.Put != nil:
		return fmt.Sprintf("%s=[operation=put, store_id=%s, key_field=%s, value_field=%s, ttl=%v, ignore_missing=%t, overwrite_fields=%t]",
			name, p.store, p.config.Put.Key, p.config.Put.Value, p.config.Put.TTL, p.config.IgnoreMissing, p.config.OverwriteKeys)
	case p.config.Get != nil:
		return fmt.Sprintf("%s=[operation=get, store_id=%s, key_field=%s, target_field=%s, ignore_missing=%t, overwrite_fields=%t]",
			name, p.store, p.config.Get.Key, p.config.Get.Target, p.config.IgnoreMissing, p.config.OverwriteKeys)
	case p.config.Delete != nil:
		return fmt.Sprintf("%s=[operation=delete, store_id=%s, key_field=%s]", name, p.store, p.config.Delete.Key)
	default:
		return fmt.Sprintf("%s=[operation=invalid]", name)
	}
}
