// +build integration

package namespace

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/aerospike"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "aerospike",
		"metricsets": []string{"namespace"},
		"hosts":      []string{aerospike.GetAerospikeEnvHost() + ":" + aerospike.GetAerospikeEnvPort()},
	}
}
