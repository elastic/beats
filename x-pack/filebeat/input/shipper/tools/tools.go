// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build tools
// +build tools

package tools

import (
	// mage notice will fail without this, since it'll try and fetch this with `go install`
	_ "go.elastic.co/go-licence-detector"

	_ "github.com/elastic/elastic-agent-libs/dev-tools/mage"

	_ "gotest.tools/gotestsum/cmd"
)
