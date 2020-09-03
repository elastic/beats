// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	// include the composable providers
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/agent"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/env"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/host"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/local"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/localdynamic"
	_ "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable/providers/path"
)
