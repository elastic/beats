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

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"

	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// Config for kubernetes autodiscover provider
type Config struct {
	KubeConfig     string        `config:"kube_config"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout" validate:"positive"`

	// Needed when resource is a pod
	HostDeprecated string            `config:"host"`
	Node           string            `config:"node"`
	Namespace      string            `config:"namespace"`
	Selector       common.MapStr     `config:"selector"`
	// Scope can be either node or cluster.
	Scope    string `config:"scope"`
	Resource string `config:"resource"`
	// Unique identifies if this provider enables its templates only when it is elected as leader in a k8s cluster
	Unique      bool   `config:"unique"`
	LeaderLease string `config:"leader_lease"`

	// Sharding config provides information for autodiscover to run in sharded mode
	Sharding ShardingConfig `config:"sharding"`

	Prefix    string                  `config:"prefix"`
	Hints     *common.Config          `config:"hints"`
	Builders  []*common.Config        `config:"builders"`
	Appenders []*common.Config        `config:"appenders"`
	Templates template.MapperSettings `config:"templates"`

	AddResourceMetadata *metadata.AddResourceMetadataConfig `config:"add_resource_metadata"`
}

type ShardingConfig struct {
	// Count defines the number of instances of Beats running in sharded mode
	Count int `config:"count"`
	// Instance identifies the nth instance of the Beat running in sharded mode
	Instance int `config:"instance"`
}

func defaultConfig() *Config {
	return &Config{
		SyncPeriod:     10 * time.Minute,
		Resource:       "pod",
		CleanupTimeout: 60 * time.Second,
		Prefix:         "co.elastic",
		Unique:         false,
		Sharding: ShardingConfig{
			Instance: -1, // We use -1 as the default so that we can deduce it from a statefulset name by default if not provided
			Count:    0,
		},
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

	if c.Sharding.Count != 0 {
		if c.Scope == "node" {
			logp.L().Warnf("can not set sharding.count to `%d` when scope to `cluster`", c.Sharding.Count)
			c.Sharding.Count = 0
			c.Sharding.Instance = -1
		} else {
			// Validate if instance is within the total number of shards being reported by `count`.
			if c.Sharding.Instance > c.Sharding.Count-1 {
				return fmt.Errorf("instance can't be greater that the total number of shards - 1 but has value %d", c.Sharding.Instance)
			}

			// Ensure that if a user decides to run cluster mode sharded Beats, then each instance and its replica do leader election.
			if c.Unique && c.LeaderLease != "" {
				c.LeaderLease = fmt.Sprintf("%s-%d", c.LeaderLease, c.Sharding.Instance)
			}

			// On a best effort try to get the sharded instance from the pod name if deployed as a statefulset
			if c.Sharding.Instance == -1 {
				var ok bool
				c.Sharding.Instance, ok = kubernetes.DiscoverPodInstanceNumber()
				if !ok {
					return fmt.Errorf("unable to determine `instance` number. `instance` is either derived from a statefulset pod or defined explicitly on the config")
				}
			}
		}
	}

	if c.Unique && c.Scope != "cluster" {
		logp.L().Warnf("can only set `unique` when scope is `cluster`")
	}

	return nil
}
