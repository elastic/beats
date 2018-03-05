package raid

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	data, err := f.Fetch()
	assert.NoError(t, err)
	assert.Equal(t, 8, len(data))
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":           "system",
		"metricsets":       []string{"raid"},
		"raid.mount_point": "./_meta/testdata",
	}
}
