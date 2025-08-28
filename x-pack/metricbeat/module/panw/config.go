// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package panw

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
)

const (
	ModuleName = "panw"
)

type Config struct {
	HostIp    string `config:"host_ip" validate:"required"`
	ApiKey    string `config:"api_key" validate:"required"`
	Port      uint   `config:"port"`
	DebugMode string `config:"api_debug_mode"`
}

func NewConfig(base mb.BaseMetricSet) (*Config, error) {
	config := Config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil

}
