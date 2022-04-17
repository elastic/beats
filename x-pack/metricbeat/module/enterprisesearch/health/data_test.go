// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package health

import (
	"testing"

	"github.com/menderesk/beats/v7/libbeat/logp"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"

	_ "github.com/menderesk/beats/v7/x-pack/metricbeat/module/enterprisesearch"
)

func TestEventMapping(t *testing.T) {
	logp.TestingSetup()
	mbtest.TestDataFiles(t, "enterprisesearch", "health")
}
