package hwmon

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
	"github.com/stretchr/testify/assert"
)

func TestDiscover(t *testing.T) {
	results, err := DetectHwmon(resolve.NewTestResolver(""))
	assert.NoError(t, err)
	t.Logf("results: %#v", results)

}

func TestMetricsSensor(t *testing.T) {
	results, err := DetectHwmon(resolve.NewTestResolver(""))
	assert.NoError(t, err)
	coreTemp := results[0]
	sensors := coreTemp.Sensors
	for _, sensor := range sensors {
		res, err := sensor.Fetch(coreTemp.AbsPath)
		if err != nil {
			t.Fatalf("error in fetch: %s", err)
		}
		to := mapstr.M{}
		err = typeconv.Convert(&to, res)
		if err != nil {
			t.Fatalf("error converting event: %s", err)
		}
		t.Logf("Result: %s", to.StringToPrint())
	}
}

func TestFetch(t *testing.T) {
	// This is meant to test how this library would be used by a metricset.

	// This would be called in New() and not Fetch(), as the results are not expected to change.
	results, err := DetectHwmon(resolve.NewTestResolver(""))
	assert.NoError(t, err)

	for _, device := range results {
		//Each device should be sent as it's own event, as they represent metrics from different places.
		sensors, err := ReportSensors(device)
		if err != nil {
			t.Fatalf("error reading sensors: %s", err)
		}
		to := mapstr.M{}
		typeconv.Convert(&to, sensors)
		// Report()
		t.Logf("Sensor: %s", to.StringToPrint())
	}
}
