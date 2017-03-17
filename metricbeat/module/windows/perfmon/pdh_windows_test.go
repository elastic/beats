// +build windows

package perfmon

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestCollectData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig("process", "processor_performance", "\\Processor Information(_Total)\\% Processor Performance"))
	event, err := f.Fetch()

}

func getConfig(group string, alias string, query string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"info"},
		"perfmon.counters": map[string]interface{}{
			"group": group,
			"collectors": map[string]interface{}{
				"alias": alias,
				"query": query,
			},
		},
	}
}
