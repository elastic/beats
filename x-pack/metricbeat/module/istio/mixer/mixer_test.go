// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package mixer

import (
	"testing"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"

	_ "github.com/menderesk/beats/v7/x-pack/metricbeat/module/istio"
)

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "istio", "mixer")
}
