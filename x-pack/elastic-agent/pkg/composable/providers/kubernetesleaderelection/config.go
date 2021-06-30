// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// TODO review the need for this
// +build linux darwin windows

package kubernetesleaderelection

// Config for kubernetes_leaderelection provider
type Config struct {
	KubeConfig string `config:"kube_config"`
	// Name of the leaderelection lease
	LeaderLease string `config:"leader_lease"`
}

// InitDefaults initializes the default values for the config.
func (c *Config) InitDefaults() {
	c.LeaderLease = "elastic-agent-cluster-leader"
}
