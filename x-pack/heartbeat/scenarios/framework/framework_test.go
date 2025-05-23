// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package framework

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var testScenario Scenario = Scenario{
	Name: "My Scenario",
	Tags: []string{"testTag"},
	Runner: func(t *testing.T) (config mapstr.M, meta ScenarioRunMeta, close func(), err error) {
		return mapstr.M{
			"type":     "http",
			"id":       "testID",
			"name":     "testName",
			"schedule": "@every 10s",
		}, meta, nil, nil
	},
	RunFrom: &config.LocationWithID{
		ID: "TestID",
		Geo: util.GeoConfig{
			Name: "TestName",
		},
	},
}

func TestClone(t *testing.T) {
	clone := testScenario.clone()
	clone.Name = "CloneName"
	require.NotEqual(t, testScenario.Name, clone.Name)
	clone.Tags = []string{"CloneTag"}
	require.NotEqual(t, testScenario.Tags, clone.Tags)
	clone.RunFrom.ID = "CloneID"
	require.NotEqual(t, testScenario.RunFrom.ID, clone.RunFrom.ID)
	clone.RunFrom.Geo.Name = "CloneGeoName"
	require.NotEqual(t, testScenario.RunFrom.Geo.Name, clone.RunFrom.Geo.Name)

}
