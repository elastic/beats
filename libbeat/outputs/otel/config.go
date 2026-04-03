// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelconsumer

import (
	"github.com/elastic/elastic-agent-libs/config"
)

type otelConsumerConfig struct {
	Queue config.Namespace `config:"queue"`
}

func defaultConfig() otelConsumerConfig {
	return otelConsumerConfig{}
}
