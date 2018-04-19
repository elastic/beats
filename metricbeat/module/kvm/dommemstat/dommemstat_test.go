package dommemstat

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/digitalocean/go-libvirt/libvirttest"
)

func TestFetchEventContents(t *testing.T) {
	conn := libvirttest.New()

	f := mbtest.NewReportingMetricSetV2(t, getConfig(conn))

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(events) == 0 {
		t.Fatal("no events received")
	}

	for _, e := range events {
		if e.Error != nil {
			t.Fatalf("received error: %+v", e.Error)
		}
	}
	if len(events) == 0 {
		t.Fatal("received no events")
	}

	e := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), e)

	statName, err := e.MetricSetFields.GetValue("stat.name")
	if err == nil {
		assert.EqualValues(t, statName.(string), "actualballon")
	} else {
		t.Errorf("error while getting value from event: %v", err)
	}

	statValue, err := e.MetricSetFields.GetValue("stat.value")
	if err == nil {
		assert.EqualValues(t, statValue, uint64(1048576))
	} else {
		t.Errorf("error while getting value from event: %v", err)
	}
}

func getConfig(conn *libvirttest.MockLibvirt) map[string]interface{} {
	return map[string]interface{}{
		"module":     "kvm",
		"metricsets": []string{"dommemstat"},
		"hosts":      []string{"test://" + conn.RemoteAddr().String() + ":123"},
	}
}
