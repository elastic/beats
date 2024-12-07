// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type inputMetrics struct {
	errs *monitoring.Uint // total number of errors
}

func newInputMetrics(reg *monitoring.Registry) *inputMetrics {
	if reg == nil {
		return nil
	}

	out := &inputMetrics{
		errs: monitoring.NewUint(reg, "errors_total"),
	}

	return out
}
