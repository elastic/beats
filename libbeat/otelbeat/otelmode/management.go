// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmode

import (
	"sync/atomic"
)

var otelManagementEnabled atomic.Bool

func SetOtelMode(enabled bool) {
	otelManagementEnabled.Store(enabled)
}

// Enabled() returns true if beatreceiver is running under Elastic Agent
func Enabled() bool {
	return otelManagementEnabled.Load()
}
