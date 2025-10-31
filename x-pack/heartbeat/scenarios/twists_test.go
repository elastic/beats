// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scenarios

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/beats/v7/x-pack/heartbeat/scenarios/framework"
	"github.com/elastic/elastic-agent-libs/mapstr"
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

func TwistMultiRun(times int) *framework.Twist {
	return framework.MakeTwist(fmt.Sprintf("run %d times", times), func(s framework.Scenario) framework.Scenario {
		s.NumberOfRuns = times
		return s
	})
}

// StdAttemptTwists is a list of real world attempt numbers, that is to say both one and two twists.
var StdAttemptTwists = []*framework.Twist{TwistMaxAttempts(1), TwistMaxAttempts(2)}

func TwistMaxAttempts(maxAttempts int) *framework.Twist {
	return framework.MakeTwist(fmt.Sprintf("run with %d max_attempts", maxAttempts), func(s framework.Scenario) framework.Scenario {
		s.Tags = append(s.Tags, "retry")
		origRunner := s.Runner
		s.Runner = func(t *testing.T) (config mapstr.M, meta framework.ScenarioRunMeta, close func(), err error) {
			config, meta, close, err = origRunner(t)
			config["max_attempts"] = maxAttempts
			return config, meta, close, err
		}
		return s
	})
}
