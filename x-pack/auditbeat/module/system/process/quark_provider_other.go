// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !linux || !(amd64 || arm64) || !cgo

package process

import (
	"errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// NewFromQuark instantiates the module with quark's backend.
func NewFromQuark(ms MetricSet) (mb.MetricSet, error) {
	return nil, errors.New("quark is only available on linux on amd64/arm64 and needs cgo")
}
