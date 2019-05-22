// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearch

import (
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

// Config for docker autodiscover provider
type Config struct {
	Query          map[string]interface{}  `config:"query"`
	Fields         []string                `config:"fields"`
	Index          string                  `config:"index"`
	Builders       []*common.Config        `config:"builders"`
	Appenders      []*common.Config        `config:"appenders"`
	HintsEnabled   bool                    `config:"hints.enabled"`
	DefaultDisable bool                    `config:"default.disable"`
	Templates      template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		Query:  common.MapStr{},
		Fields: []string{"host.ip"},
	}
}

// Validate ensures correctness of config
func (c *Config) Validate() {
}
