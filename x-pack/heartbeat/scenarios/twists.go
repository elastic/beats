// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/processors/util"
)

var TestLocationDefault = TestLocationMpls

var TestLocationMpls = &config.LocationWithID{
	ID: "na-mpls",
	Geo: util.GeoConfig{
		Name:     "Minneapolis",
		Location: "44.9778, 93.2650",
	},
}

var TwistEnableES = MakeTwist("with ES", func(s Scenario) Scenario {
	s.ESEnabled = true
	return s
})

var TwistAddLocation = MakeTwist("add location", func(s Scenario) Scenario {
	s.Location = TestLocationDefault
	return s
})

func TwistMultiRun(times int) Twist {
	return MakeTwist(fmt.Sprintf("run %d times", times), func(s Scenario) Scenario {
		s.NumberOfRuns = times
		return s
	})
}
