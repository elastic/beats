// +build windows

package perfmon

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
)

func TestExistingCounter(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig("process", "processor_performance", "\\Processor Information(_Total)\\% Processor Performance"))
	data, err := f.Fetch()

	assert.Nil(t, err)
	assert.Regexp(t, `{"process":{"processor_performance":[0-9]+\.[0-9]+}}`, data)

}

func TestNonExistingCounter(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig("process", "processor_performance", "\\Non Existing\\Counter"))
	data, err := f.Fetch()

	assert.Nil(t, err)
	assert.Regexp(t, `{"process":{"processor_performance":0`, data)

}

func getConfig(group string, alias string, query string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"perfmon"},
		"perfmon.counters": [1]map[string]interface{}{
			map[string]interface{}{
				"group": group,
				"collectors": [1]map[string]interface{}{
					map[string]interface{}{
						"alias": alias,
						"query": query,
					},
				},
			},
		},
	}
}
