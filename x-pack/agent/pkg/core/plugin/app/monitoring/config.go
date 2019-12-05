// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitoring

// Config describes a configuration of a monitoring
type Config struct {
	MonitorLogs    bool `yaml:"logs" config:"logs"`
	MonitorMetrics bool `yaml:"metrics" config:"metrics"`
}
