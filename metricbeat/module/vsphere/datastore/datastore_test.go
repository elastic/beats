package datastore

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/simulator"
)

func TestFetchEventContents(t *testing.T) {
	model := simulator.ESX()
	if err := model.Create(); err != nil {
		t.Fatal(err)
	}

	ts := model.Service.NewServer()
	defer ts.Close()

	f := mbtest.NewEventsFetcher(t, getConfig(ts))
	events, err := f.Fetch()
	if err != nil {
		t.Fatal("fetch error", err)
	}

	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "LocalDS_0", event["name"])
	assert.EqualValues(t, "local", event["fstype"])

	// Values are based on the result 'df -k'.
	fields := []string{"capacity.total.bytes", "capacity.free.bytes",
		"capacity.used.bytes", "capacity.used.pct"}
	for _, field := range fields {
		value, err := event.GetValue(field)
		if err != nil {
			t.Error(err)
		} else {
			isNonNegativeInt64(t, field, value)
		}
	}
}

func isNonNegativeInt64(t testing.TB, field string, v interface{}) {
	i, ok := v.(int64)
	if !ok {
		t.Errorf("%v: got %T, but expected int64", field, v)
		return
	}

	if i < 0 {
		t.Errorf("%v: value is negative (%v)", field, i)
		return
	}
}

func TestData(t *testing.T) {
	model := simulator.ESX()
	if err := model.Create(); err != nil {
		t.Fatal(err)
	}

	ts := model.Service.NewServer()
	defer ts.Close()

	f := mbtest.NewEventsFetcher(t, getConfig(ts))

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(ts *simulator.Server) map[string]interface{} {
	urlSimulator := ts.URL.Scheme + "://" + ts.URL.Host + ts.URL.Path

	return map[string]interface{}{
		"module":     "vsphere",
		"metricsets": []string{"datastore"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
