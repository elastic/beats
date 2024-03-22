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
	"errors"
	"time"
)

type config struct {
	Get    *getConfig `config:"get"`
	Put    *putConfig `config:"put"`
	Delete *delConfig `config:"delete"`

	Store *storeConfig `config:"backend" validate:"required"`

	// IgnoreMissing: Ignore errors if event has no matching field.
	IgnoreMissing bool `config:"ignore_missing"`

	// OverwriteKeys allow target_fields to overwrite existing fields.
	OverwriteKeys bool `config:"overwrite_keys"`
}

func (cfg *config) Validate() error {
	var ops int
	if cfg.Put != nil {
		ops++
	}
	if cfg.Get != nil {
		ops++
	}
	if cfg.Delete != nil {
		ops++
	}
	switch ops {
	case 0:
		return errors.New("no operation specified for cache processor")
	case 1:
		return nil
	default:
		return errors.New("cannot specify multiple operations together in a cache processor")
	}
}

type getConfig struct {
	// Key is the field containing the key to lookup for matching.
	Key string `config:"key_field" validate:"required"`

	// Target is the destination field where fields will be added.
	Target string `config:"target_field" validate:"required"`
}

type putConfig struct {
	// Key is the field containing the key to lookup for matching.
	Key string `config:"key_field" validate:"required"`

	// Target is the destination field where fields will be added.
	Value string `config:"value_field" validate:"required"`

	TTL *time.Duration `config:"ttl" validate:"required"`
}

type delConfig struct {
	// Key is the field containing the key to lookup for deletion.
	Key string `config:"key_field" validate:"required"`
}

func defaultConfig() config {
	return config{
		IgnoreMissing: true,
		OverwriteKeys: false,
	}
}

type storeConfig struct {
	Memory *memConfig  `config:"memory"`
	File   *fileConfig `config:"file"`

	// Capacity is the number of elements that may be stored.
	Capacity int `config:"capacity"`

	// Effort is currently experimental and
	// not in public-facing documentation.
	Effort int `config:"eviction_effort"`
}

type memConfig struct {
	ID string `config:"id" validate:"required"`
}

type fileConfig struct {
	ID            string        `config:"id" validate:"required"`
	WriteOutEvery time.Duration `config:"write_period"`
}

func (cfg *storeConfig) Validate() error {
	switch {
	case cfg.Memory != nil && cfg.File != nil:
		return errors.New("must specify only one of backend.memory.id or backend.file.id")
	case cfg.Memory != nil, cfg.File != nil:
	default:
		return errors.New("must specify one of backend.memory.id or backend.file.id")
	}
	return nil
}
