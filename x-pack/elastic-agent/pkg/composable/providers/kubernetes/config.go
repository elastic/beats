// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// TODO review the need for this
// +build linux darwin windows

package kubernetes

import (
	"time"
)

// Config for kubernetes provider
type Config struct {
	KubeConfig     string        `config:"kube_config"`
	SyncPeriod     time.Duration `config:"sync_period"`
	CleanupTimeout time.Duration `config:"cleanup_timeout" validate:"positive"`

	// Needed when resource is a pod
	Node string `config:"node"`

	// Scope of the provider (cluster or node)
	Scope string `config:"scope"`
}

// InitDefaults initializes the default values for the config.
func (c *Config) InitDefaults() {
	c.SyncPeriod = 10 * time.Minute
	c.CleanupTimeout = 60 * time.Second
	c.Scope = "node"
}
