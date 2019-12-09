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

// +build linux darwin windows

package kubernetes

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

// Config for kubernetes autodiscover provider
type Config struct {
	KubeConfig     string        `config:"kube_config"`
	Namespace      string        `config:"namespace"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout" validate:"positive"`

	// Needed when resource is a pod
	HostDeprecated string `config:"host"`
	Node           string `config:"node"`
	// Scope can be either node or cluster.
	Scope    string `config:"scope"`
	Resource string `config:"resource"`

	Prefix    string                  `config:"prefix"`
	Hints     *common.Config          `config:"hints"`
	Builders  []*common.Config        `config:"builders"`
	Appenders []*common.Config        `config:"appenders"`
	Templates template.MapperSettings `config:"templates"`
}

func defaultConfig() *Config {
	return &Config{
		SyncPeriod:     10 * time.Minute,
		Resource:       "pod",
		CleanupTimeout: 60 * time.Second,
		Prefix:         "co.elastic",
	}
}

// Validate ensures correctness of config
func (c *Config) Validate() error {
	// Make sure that prefix doesn't ends with a '.'
	if c.Prefix[len(c.Prefix)-1] == '.' && c.Prefix != "." {
		c.Prefix = c.Prefix[:len(c.Prefix)-2]
	}

	if len(c.Templates) == 0 && !c.Hints.Enabled() && len(c.Builders) == 0 {
		return fmt.Errorf("no configs or hints defined for autodiscover provider")
	}

	// Check if host is being defined and change it to node instead.
	if c.Node == "" && c.HostDeprecated != "" {
		c.Node = c.HostDeprecated
		cfgwarn.Deprecate("8.0", "`host` will be deprecated, use `node` instead")
	}

	// Check if resource is either node or pod. If yes then default the scope to "node" if not provided.
	// Default the scope to "cluster" for everything else.
	switch c.Resource {
	case "node", "pod":
		if c.Scope == "" {
			c.Scope = "node"
		}

	default:
		if c.Scope == "node" {
			logp.L().Warnf("can not set scope to `node` when using resource %s. resetting scope to `cluster`", c.Resource)
		}
		c.Scope = "cluster"
	}

	if c.Scope != "node" && c.Scope != "cluster" {
		return fmt.Errorf("invalid `scope` configured. supported values are `node` and `cluster`")
	}

	return nil
}
