// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"github.com/elastic/fleet/x-pack/pkg/artifact"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/retry"
)

// Config is an operator configuration
type Config struct {
	ReattachCollectionPath string `yaml:"reattach_collection_path" config:"reattach_collection_path"`

	ProcessConfig *process.Config `yaml:"process" config:"process"`
	RetryConfig   *retry.Config   `yaml:"retry" config:"retry"`

	DownloadConfig *artifact.Config `yaml:"download" config:"download"`
}
