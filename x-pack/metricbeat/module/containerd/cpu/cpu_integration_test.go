// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && linux
// +build integration,linux

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cpu

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetchMetricset(t *testing.T) {
	config := GetContainerdConfig(t, "cpu")
	metricSet := mbtest.NewFetcher(t, config)
	events, errs := metricSet.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
}

// GetContainerdConfig function returns configuration for talking to containerd.
func GetContainerdConfig(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"module":     "containerd",
		"metricsets": []string{metricSetName},
		"hosts":      []string{"localhost:1338"},
		"calcpct":    true,
	}
}
