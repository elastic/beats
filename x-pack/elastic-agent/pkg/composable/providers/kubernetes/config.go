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
	Scope     string    `config:"scope"`
	Resources Resources `config:"resources"`
}

type Resources struct {
	Pod     *ResourceConfig `config:"pod"`
	Node    *ResourceConfig `config:"node"`
	Service *ResourceConfig `config:"service"`
}

// Config for kubernetes provider
type ResourceConfig struct {
	KubeConfig     string        `config:"kube_config"`
	Namespace      string        `config:"namespace"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout" validate:"positive"`

	// Needed when resource is a pod
	Node string `config:"node"`
}

// InitDefaults initializes the default values for the config.
func (c *Config) InitDefaults() {
	if c.Resources.Pod == nil {
		c.Resources.Pod = &ResourceConfig{}
	}
	c.Resources.Pod.SyncPeriod = 10 * time.Minute
	c.Resources.Pod.CleanupTimeout = 60 * time.Second
	c.Scope = "node"
}

// Validate ensures correctness of config
func (c *Config) Validate() error {
	// Check if resource is service. If yes then default the scope to "cluster".
	if c.Resources.Service != nil {
		if c.Scope == "node" {
			logp.L().Warnf("can not set scope to `node` when using resource `Service`. resetting scope to `cluster`")
		}
		c.Scope = "cluster"
	}

	return nil
}
