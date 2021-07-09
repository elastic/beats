// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// TODO review the need for this
// +build linux darwin windows

package kubernetes

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// Config for kubernetes provider
type Config struct {
	KubeConfig     string        `config:"kube_config"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout" validate:"positive"`

	// Needed when resource is a pod
	Node string `config:"node"`

	// Scope of the provider (cluster or node)
	Scope    string `config:"scope"`
	Resource string `config:"resource"`
}

// InitDefaults initializes the default values for the config.
func (c *Config) InitDefaults() {
	c.SyncPeriod = 10 * time.Minute
	c.CleanupTimeout = 60 * time.Second
	c.Scope = "node"
}

// Validate ensures correctness of config
func (c *Config) Validate() error {
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

	return nil
}
