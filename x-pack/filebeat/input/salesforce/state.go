// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import "time"

type state struct {
	StartTime   time.Time `struct:"start_timestamp"`
	LogDateTime string    `struct:"timestamp"`
}
