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
	defaultTimeout                  = time.Second
	defaultMaxTransactionSize int64 = 65536
)

// Config defines configuration for the bbolt storage backend.
type Config struct {
	// Timeout is the amount of time to wait to obtain a file lock on the
	// bbolt database file.
	Timeout time.Duration `config:"timeout"`

	// FSync specifies that fsync should be called after each database write.
	FSync bool `config:"fsync"`

	// Compaction holds compaction related configuration.
	Compaction CompactionConfig `config:"compaction"`

	// Retention holds TTL and periodic retention cleanup configuration.
	Retention RetentionConfig `config:"retention"`
}

// CompactionConfig defines configuration for bbolt database compaction.
type CompactionConfig struct {
	// OnStart specifies that compaction is attempted each time on start.
	OnStart bool `config:"on_start"`

	// MaxTransactionSize specifies the maximum number of items in a single
	// compaction or cleanup iteration.
	MaxTransactionSize int64 `config:"max_transaction_size"`

	// CleanupOnStart specifies that leftover temporary compaction files are
	// removed on start.
	CleanupOnStart bool `config:"cleanup_on_start"`
}

// RetentionConfig defines configuration for TTL-based entry retention.
type RetentionConfig struct {
	// TTL defines how long entries are kept in the store before being
	// removed. A zero value disables TTL-based removal.
	TTL time.Duration `config:"ttl"`

	// Interval specifies how often expired entries are removed.
	// Only effective when TTL is also set. A zero value disables
	// periodic removal.
	Interval time.Duration `config:"interval"`
}

// Validate checks the configuration for invalid or contradictory values.
func (c *Config) Validate() error {
	if c.Timeout < 0 {
		return errors.New("bbolt timeout must not be negative")
	}
	if c.Compaction.MaxTransactionSize < 0 {
		return errors.New("bbolt compaction max_transaction_size must not be negative")
	}
	if c.Retention.TTL < 0 {
		return errors.New("bbolt retention TTL must not be negative")
	}
	if c.Retention.TTL > 0 && c.Retention.Interval < 0 {
		return errors.New("bbolt retention interval must not be negative when TTL is set")
	}
	return nil
}

// DefaultConfig returns the default bbolt configuration.
func DefaultConfig() Config {
	return Config{
		Timeout: defaultTimeout,
		Compaction: CompactionConfig{
			MaxTransactionSize: defaultMaxTransactionSize,
		},
	}
}
