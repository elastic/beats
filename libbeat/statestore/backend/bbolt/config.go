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

package bbolt

import (
	"errors"
	"time"
)

const (
	defaultTimeout                       = time.Second
	defaultMaxTransactionSize      int64 = 65536
	defaultReboundTriggerMiB       int64 = 10
	defaultReboundNeededMiB        int64 = 100
	defaultCompactionCheckInterval       = 5 * time.Second
)

// Config defines configuration for the bbolt storage backend.
// Parameter names are aligned with the OTel filestorage extension
// for future compatibility.
type Config struct {
	// Timeout is the amount of time to wait to obtain a file lock on the
	// bbolt database file.
	Timeout time.Duration `config:"timeout"`

	// FSync specifies that fsync should be called after each database write.
	FSync bool `config:"fsync"`

	// TTL defines how long entries are kept in the store before being
	// removed by the cleanup process. A zero value disables TTL-based cleanup.
	TTL time.Duration `config:"ttl"`

	// Compaction holds compaction and cleanup related configuration.
	Compaction CompactionConfig `config:"compaction"`
}

// CompactionConfig defines configuration for bbolt database compaction
// and TTL-based entry cleanup. Parameter names are aligned with the
// OTel filestorage extension for future compatibility.
type CompactionConfig struct {
	// OnStart specifies that compaction is attempted each time on start.
	OnStart bool `config:"on_start"`

	// OnRebound specifies that compaction is attempted online when rebound
	// conditions are met.
	OnRebound bool `config:"on_rebound"`

	// ReboundNeededThresholdMiB specifies the minimum total allocated size
	// to mark the need for online compaction.
	ReboundNeededThresholdMiB int64 `config:"rebound_needed_threshold_mib"`

	// ReboundTriggerThresholdMiB is used when compaction is marked as
	// needed. When allocated data size drops below the specified value,
	// compaction starts.
	ReboundTriggerThresholdMiB int64 `config:"rebound_trigger_threshold_mib"`

	// MaxTransactionSize specifies the maximum number of items in a single
	// compaction iteration.
	MaxTransactionSize int64 `config:"max_transaction_size"`

	// CheckInterval specifies the frequency of the rebound compaction check.
	CheckInterval time.Duration `config:"check_interval"`

	// CleanupOnStart specifies that leftover temporary compaction files are
	// removed on start.
	CleanupOnStart bool `config:"cleanup_on_start"`

	// CleanupInterval specifies how often TTL-based entry cleanup runs.
	// Only effective when TTL is also configured. A zero value disables
	// the periodic cleanup.
	CleanupInterval time.Duration `config:"cleanup_interval"`
}

// Validate checks the configuration for invalid or contradictory values.
func (c *Config) Validate() error {
	if c.Timeout < 0 {
		return errors.New("bbolt timeout must not be negative")
	}
	if c.TTL < 0 {
		return errors.New("bbolt TTL must not be negative")
	}
	if c.Compaction.MaxTransactionSize < 0 {
		return errors.New("bbolt compaction max_transaction_size must not be negative")
	}
	if c.Compaction.OnRebound && c.Compaction.CheckInterval <= 0 {
		return errors.New("bbolt compaction check_interval must be positive when on_rebound is enabled")
	}
	if c.TTL > 0 && c.Compaction.CleanupInterval < 0 {
		return errors.New("bbolt compaction cleanup_interval must not be negative when TTL is set")
	}
	return nil
}

// DefaultConfig returns the default bbolt configuration.
func DefaultConfig() Config {
	return Config{
		Timeout: defaultTimeout,
		FSync:   false,
		TTL:     0,
		Compaction: CompactionConfig{
			OnStart:                    false,
			OnRebound:                  false,
			ReboundNeededThresholdMiB:  defaultReboundNeededMiB,
			ReboundTriggerThresholdMiB: defaultReboundTriggerMiB,
			MaxTransactionSize:         defaultMaxTransactionSize,
			CheckInterval:              defaultCompactionCheckInterval,
			CleanupOnStart:             false,
			CleanupInterval:            0,
		},
	}
}
