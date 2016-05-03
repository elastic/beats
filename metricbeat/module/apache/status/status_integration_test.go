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
	event, err := f.Fetch(f.Module().Config().Hosts[0])
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check number of fields.
	assert.Equal(t, 13, len(event))
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "apache",
		"metricsets": []string{"status"},
		"hosts":      []string{apache.GetApacheEnvHost()},
	}
}
