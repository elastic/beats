// Fetches statistics from the output of running beats
package expvar

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/golang"
)

func init() {
	MetricSet.Register()
}

// MetricSet Setup
var MetricSet = helper.NewMetricSet("expvar", ExpvarMetric{}, golang.Module)

// MetricSetter object
type ExpvarMetric struct {
	helper.MetricSetConfig
}

func (b ExpvarMetric) Setup() error {
	return nil
}

// Fetch expvars from a running beat
func (b ExpvarMetric) Fetch() (events []common.MapStr, err error) {

	//path := "http://localhost:6060/debug/vars"

	event := common.MapStr{
		"type":  "helloworld",
		"index": "indexnameyes",
		"bac":   "rrre",
	}

	events = append(events, event)

	return events, nil
}

func (b ExpvarMetric) Cleanup() error {
	return nil
}
