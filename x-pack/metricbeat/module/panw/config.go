// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package panw

import (
	"errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

const (
	ModuleName = "panw"
)

type Config struct {
	HostIp    string `config:"host_ip"`
	ApiKey    string `config:"apiKey"`
	DebugMode string `config:"apiDebugMode"`
}

func NewConfig(base mb.BaseMetricSet) (*Config, error) {
	config := Config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if (config.HostIp == "") || (config.ApiKey == "") {
		return nil, errors.New("host_ip and apiKey must be set	")
	}

	return &config, nil

}
