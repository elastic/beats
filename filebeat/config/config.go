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

package config

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Defaults for config variables which are not set
const (
	DefaultType = "log"
)

type Config struct {
	Inputs             []*conf.C            `config:"inputs"`
	Registry           Registry             `config:"registry"`
	ConfigDir          string               `config:"config_dir"`
	ShutdownTimeout    time.Duration        `config:"shutdown_timeout"`
	Modules            []*conf.C            `config:"modules"`
	ConfigInput        *conf.C              `config:"config.inputs"`
	ConfigModules      *conf.C              `config:"config.modules"`
	Autodiscover       *autodiscover.Config `config:"autodiscover"`
	OverwritePipelines bool                 `config:"overwrite_pipelines"`
}

type Registry struct {
	Path          string        `config:"path"`
	Permissions   os.FileMode   `config:"file_permissions"`
	FlushTimeout  time.Duration `config:"flush"`
	CleanInterval time.Duration `config:"cleanup_interval"`
	MigrateFile   string        `config:"migrate_file"`

	// Type selects the registry backend implementation.
	// Supported values: "bbolt" (default), "memlog".
	Type string `config:"type"`

	// BBolt holds bbolt-specific configuration.
	BBolt BBoltConfig `config:"bbolt"`
}

type BBoltConfig struct {
	// DiskTTL is the inactivity duration after which entries are considered stale.
	// If 0, disk GC is disabled.
	DiskTTL time.Duration `config:"disk_ttl"`

	// CacheTTL is reserved for Phase 2 (in-memory cache). Not used yet.
	CacheTTL time.Duration `config:"cache_ttl"`

	// GCBatchSize is reserved for Phase 3 (incremental GC). Not used yet.
	GCBatchSize int `config:"gc_batch_size"`

	// FileMode is used as the file mode for newly created bbolt DB files.
	FileMode os.FileMode `config:"file_permissions"`

	// Timeout sets the bbolt open timeout.
	Timeout time.Duration `config:"timeout"`

	// NoGrowSync disables the grow sync behavior in bbolt.
	NoGrowSync bool `config:"no_grow_sync"`

	// NoFreelistSync disables freelist syncing in bbolt.
	NoFreelistSync bool `config:"no_freelist_sync"`
}

var DefaultConfig = Config{
	Registry: Registry{
		Path:          "registry",
		Permissions:   0o600,
		MigrateFile:   "",
		CleanInterval: 5 * time.Minute,
		FlushTimeout:  time.Second,
		Type:          "bbolt",
		BBolt: BBoltConfig{
			DiskTTL:        30 * 24 * time.Hour,
			CacheTTL:       1 * time.Hour,
			GCBatchSize:    50_000,
			FileMode:       0o600,
			Timeout:        1 * time.Second,
			NoGrowSync:     false,
			NoFreelistSync: true,
		},
	},
	ShutdownTimeout:    0,
	OverwritePipelines: false,
}

func (r Registry) NormalizedType() string {
	if r.Type == "" {
		return "bbolt"
	}
	return r.Type
}

func (r Registry) ValidateConfig() error {
	switch r.NormalizedType() {
	case "bbolt", "memlog":
	default:
		return fmt.Errorf("unknown filebeat.registry.type: %q", r.Type)
	}

	if r.Path == "" {
		return fmt.Errorf("filebeat.registry.path is empty")
	}

	if r.BBolt.Timeout < 0 {
		return fmt.Errorf("filebeat.registry.bbolt.timeout must be >= 0 (got %v)", r.BBolt.Timeout)
	}
	if r.BBolt.DiskTTL < 0 {
		return fmt.Errorf("filebeat.registry.bbolt.disk_ttl must be >= 0 (got %v)", r.BBolt.DiskTTL)
	}
	if r.BBolt.CacheTTL < 0 {
		return fmt.Errorf("filebeat.registry.bbolt.cache_ttl must be >= 0 (got %v)", r.BBolt.CacheTTL)
	}
	if r.BBolt.GCBatchSize < 0 {
		return fmt.Errorf("filebeat.registry.bbolt.gc_batch_size must be >= 0 (got %d)", r.BBolt.GCBatchSize)
	}

	return nil
}

// ListEnabledInputs returns a list of enabled inputs sorted by alphabetical order.
func (config *Config) ListEnabledInputs() []string {
	t := struct {
		Type string `config:"type"`
	}{}
	var inputs []string
	for _, input := range config.Inputs {
		if input.Enabled() {
			_ = input.Unpack(&t)
			inputs = append(inputs, t.Type)
		}
	}
	sort.Strings(inputs)
	return inputs
}

// IsInputEnabled returns true if the plugin name is enabled.
func (config *Config) IsInputEnabled(name string) bool {
	enabledInputs := config.ListEnabledInputs()
	for _, input := range enabledInputs {
		if name == input {
			return true
		}
	}
	return false
}
