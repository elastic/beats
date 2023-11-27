// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"time"
)

type state struct {
	StartTime time.Time `struct:"start_timestamp"`
	LogTime   string    `struct:"timestamp"`
}

// setCheckpoint sets checkpoint from source to current state instance
func (s *state) setCheckpoint(chkpt string) {
	s.LogTime = chkpt
}
