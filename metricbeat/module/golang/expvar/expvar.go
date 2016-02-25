// Fetches statistics from the output of running beats
package expvar

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	_ "github.com/elastic/beats/metricbeat/module/golang"
)

func init() {
	helper.Registry.AddMetricSeter("golang", "expvar", MetricSeter{})
}

type MetricSeter struct{}

func (m MetricSeter) Setup() error {
	return nil
}

// Fetch expvars from a running beat
func (m MetricSeter) Fetch(ms *helper.MetricSet) (events []common.MapStr, err error) {

	//path := "http://localhost:6060/debug/vars"

	event := common.MapStr{
		"type":  "helloworld",
		"index": "indexnameyes",
		"bac":   "rrre",
	}

	events = append(events, event)

	return events, nil
}

func (m MetricSeter) Cleanup() error {
	return nil
}
