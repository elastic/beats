// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstash

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
	"github.com/elastic/elastic-agent-libs/config"
)

// ToMap converts Logstash output config to a map[string]any
// raises error if config is invalid
func ToMap(outputConfig *config.C) (map[string]any, error) {
	lsHostWorker := struct {
		outputs.HostWorkerCfg `config:",inline"`
		logstash.Config       `config:",inline"`
	}{
		Config: logstash.DefaultConfig(),
	}

	// unpack and validate LS config
	if err := outputConfig.Unpack(&lsHostWorker); err != nil {
		return nil, fmt.Errorf("failed unpacking logstash config: %w", err)
	}

	lsConfig := config.MustNewConfigFrom(lsHostWorker)
	var lsConfigMap map[string]any
	if err := lsConfig.Unpack(&lsConfigMap); err != nil {
		return nil, err
	}

	return lsConfigMap, nil
}
