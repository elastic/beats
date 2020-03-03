// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
)

// Config for nomad autodiscover provider
type Config struct {
	Address        string        `config:"address"`
	Region         string        `config:"region"`
	Namespace      string        `config:"namespace"`
	SecretID       string        `config:"secret_id"`
	Host           string        `config:"host"`
	WaitTime       time.Duration `config:"wait_time"`
	SyncPeriod     time.Duration `config:"sync_period"`
	AllowStale     bool          `config:"allow_stale"`
	CleanupTimeout time.Duration `config:"cleanup_timeout" validate:"positive"`

	Prefix    string                  `config:"prefix"`
	Hints     *common.Config          `config:"hints"`
	Builders  []*common.Config        `config:"builders"`
	Appenders []*common.Config        `config:"appenders"`
	Templates template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		Address:        "http://127.0.0.1:4646",
		Region:         "",
		Namespace:      "",
		SecretID:       "",
		AllowStale:     true,
		WaitTime:       15 * time.Second,
		SyncPeriod:     30 * time.Second,
		CleanupTimeout: 60 * time.Second,
		Prefix:         "co.elastic",
	}
}

// Validate ensures correctness of config
func (c *Config) Validate() {
	// Make sure that prefix doesn't ends with a '.'
	if c.Prefix[len(c.Prefix)-1] == '.' && c.Prefix != "." {
		c.Prefix = c.Prefix[:len(c.Prefix)-2]
	}
}
