package scenarios

import (
	"testing"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/require"
)

var testScenario Scenario = Scenario{
	Name: "My Scenario",
	Tags: []string{"testTag"},
	Runner: func(t *testing.T) (config mapstr.M, close func(), err error) {
		return mapstr.M{
			"type":     "http",
			"id":       "testID",
			"name":     "testName",
			"schedule": "@every 10s",
		}, nil, nil
	},
	Location: &config.LocationWithID{
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
	clone.Location.ID = "CloneID"
	require.NotEqual(t, testScenario.Location.ID, clone.Location.ID)
	clone.Location.Geo.Name = "CloneGeoName"
	require.NotEqual(t, testScenario.Location.Geo.Name, clone.Location.Geo.Name)

}
