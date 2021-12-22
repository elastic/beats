// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package containerd

// Config contains the config needed for containerd
type Config struct {
	CalculateCpuPct bool `config:"calcpct.cpu"`
	CalculateMemPct bool `config:"calcpct.memory"`
}

// DefaultConfig returns default module config
func DefaultConfig() Config {
	return Config{
		CalculateCpuPct: true,
		CalculateMemPct: true,
	}
}
