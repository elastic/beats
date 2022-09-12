// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
)

var TestLocationDefault = TestLocationMpls

var TestLocationMpls = &config.LocationWithID{
	ID: "na-mpls",
	Geo: util.GeoConfig{
		Name:     "Minneapolis",
		Location: "44.9778, 93.2650",
	},
}

var TwistAddRunFrom = framework.MakeTwist("add run_from", func(s framework.Scenario) framework.Scenario {
	s.RunFrom = TestLocationDefault
	return s
})

func TwistMultiRun(times int) framework.Twist {
	return framework.MakeTwist(fmt.Sprintf("run %d times", times), func(s framework.Scenario) framework.Scenario {
		s.NumberOfRuns = times
		return s
	})
}
