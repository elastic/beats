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

package kubernetes

import (
	"time"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

// Config for kubernetes autodiscover provider
type Config struct {
	InCluster      bool          `config:"in_cluster"`
	KubeConfig     string        `config:"kube_config"`
	Host           string        `config:"host"`
	Namespace      string        `config:"namespace"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout"`

	Prefix         string                  `config:"prefix"`
	HintsEnabled   bool                    `config:"hints.enabled"`
	DefaultDisable bool                    `config:"default.disable"`
	Builders       []*common.Config        `config:"builders"`
	Appenders      []*common.Config        `config:"appenders"`
	Templates      template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		InCluster:      true,
		SyncPeriod:     1 * time.Second,
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
