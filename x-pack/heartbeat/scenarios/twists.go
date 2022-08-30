package scenarios

import (
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

var TwistAddLocation = MakeTwist("add location", func(s Scenario) Scenario {
	s.Location = TestLocationDefault
	return s
})
