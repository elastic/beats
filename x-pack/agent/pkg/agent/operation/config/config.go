// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app/monitoring"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/retry"
)

// Config is an operator configuration
type Config struct {
	ProcessConfig *process.Config `yaml:"process" config:"process"`
	RetryConfig   *retry.Config   `yaml:"retry" config:"retry"`

	DownloadConfig *artifact.Config `yaml:"download" config:"download"`

	MonitoringConfig *monitoring.Config `yaml:"settings.monitoring" config:"settings.monitoring"`
}
