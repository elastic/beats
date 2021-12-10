package broker

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "pulsar")

	metricSet := mbtest.NewFetcher(t, getConfig())
	events, errs := metricSet.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "pulsar")
	metricSet := mbtest.NewFetcher(t, getConfig())
	metricSet.WriteEvents(t, "")
}

// getConfig returns config for pulsar module
func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "pulsar",
		"metricsets": []string{"broker"},
		"hosts":      []string{"http://localhost:8080"},
	}
}
