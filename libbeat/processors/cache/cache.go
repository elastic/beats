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
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	src, cancel, err := getStoreFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get the store for %s: %w", name, err)
	}

	// Logging (each processor instance has a unique ID).
	id := int(instanceID.Inc())
	log := logp.NewLogger(name).With("instance_id", id)

	p := cache{
		config: config,
		store:  src,
		cancel: cancel,
		log:    log,
	}
	return &p, nil
}

// Store is the interface implemented by metadata providers.
type Store interface {
	Put(key string, val any) error
	Get(key string) (any, error)
	Delete(key string) error
}

type CacheEntry struct {
	key     string
	value   any
	expires time.Time
	index   int
}

var (
	storeMu    sync.Mutex
	memStores  = map[string]*memStore{}
	fileStores = map[string]*memStore{}
)

// getStoreFor returns a backing store for the provided configuration,
// and a context cancellation that releases the cache resource when it
// is no longer required. The cancellation should be called when the
// processor is closed.
func getStoreFor(cfg config) (Store, context.CancelFunc, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	switch {
	case cfg.Store.Memory != nil:
		s, cancel := getMemStore(memStores, cfg.Store.Memory.ID, cfg)
		return s, cancel, nil

	case cfg.Store.File != nil:
		logp.L().Warn("using memory store when file is configured")
		// TODO: Replace place-holder code with a file-store.
		s, cancel := getMemStore(fileStores, cfg.Store.File.ID, cfg)
		return s, cancel, nil

	default:
		// This should have been caught by config validation.
		return nil, noop, errors.New("no configured store")
	}
}

func getMemStore(stores map[string]*memStore, id string, cfg config) (*memStore, context.CancelFunc) {
	s, ok := stores[id]
	if ok {
		// We may have already constructed the store with
		// a get or a delete config, so set the TTL, cap
		// and effort if we have a put config. If another
		// put config has already been included, we ignore
		// the put options now.
		s.setPutOptions(cfg)
		return s, noop
	}
	s = newMemStore(cfg)
	stores[id] = s
	return s, func() {
		// TODO: Consider making this reference counted.
		// Currently, what we have is essentially an
		// ownership model, where the put operation is
		// owner. This could conceivably be problematic
		// if a processor were shared between different
		// inputs and the put is closed due to a config
		// change.
		storeMu.Lock()
		delete(stores, id)
		storeMu.Unlock()
	}
}

// noop is a no-op context.CancelFunc.
func noop() {}

// Run enriches the given event with the host metadata.
func (p *cache) Run(event *beat.Event) (*beat.Event, error) {
	switch {
	case p.config.Put != nil:
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
	// ... and write it into the cloned event.
	result = event.Clone()
	if _, err = result.PutValue(dst, meta); err != nil {
		return nil, err
	}
	return result, nil
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
		return fmt.Sprintf("%s=[operation=put, key_field=%s, value_field=%s, ttl=%v, ignore_missing=%t, overwrite_fields=%t]",
			name, p.config.Put.Key, p.config.Put.Value, p.config.Put.TTL, p.config.IgnoreMissing, p.config.OverwriteKeys)
	case p.config.Get != nil:
		return fmt.Sprintf("%s=[operation=get, key_field=%s, target_field=%s, ignore_missing=%t, overwrite_fields=%t]",
			name, p.config.Get.Key, p.config.Get.Target, p.config.IgnoreMissing, p.config.OverwriteKeys)
	case p.config.Delete != nil:
		return fmt.Sprintf("%s=[operation=delete, key_field=%s]", name, p.config.Delete.Key)
	default:
		return fmt.Sprintf("%s=[operation=invalid]", name)
	}
}
