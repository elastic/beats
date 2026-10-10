// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

// Secret store implementations register themselves on import.
import (
	_ "github.com/elastic/beats/v7/x-pack/heartbeat/secretstores/vault"
)
