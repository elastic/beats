// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && aws
// +build integration,aws

package dynamodb

import (
	"testing"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/aws/mtest"
)

func TestData(t *testing.T) {
	config := mtest.GetConfigForTest(t, "dynamodb", "300s")

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}
