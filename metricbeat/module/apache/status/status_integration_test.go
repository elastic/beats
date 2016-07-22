// +build integration

package status

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/apache"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check number of fields.
	if len(event) < 11 {
		t.Fatal("Too few top-level elements in the event")
	}
}

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "apache",
		"metricsets": []string{"status"},
		"hosts":      []string{apache.GetApacheEnvHost()},
	}
}
