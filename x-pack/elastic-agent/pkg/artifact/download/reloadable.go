// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package download

import "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"

// Reloader is an interface allowing to reload artifact config
type Reloader interface {
	Reload(*artifact.Config) error
}
