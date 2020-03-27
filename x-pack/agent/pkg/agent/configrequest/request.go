// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configrequest

import (
	"time"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/program"
)

// Request is the minimal interface a config request must have.
type Request interface {
	ID() string
	CreatedAt() time.Time
	Programs() []program.Program
}
