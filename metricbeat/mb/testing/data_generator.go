package testing

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	// To enable the data building, run go test  `github.com/elastic/beats/metricbeat/module/system/memory/... -data=true`
	dataFlag = flag.Bool("data", false, "Enabled creating of data")
)

func WriteEvent(f mb.EventFetcher, t *testing.T) error {

	if !*dataFlag {
		t.Skip("Skip data generation tests")
	}

	event, err := f.Fetch()
	if err != nil {
		return err
	}

	return createEvent(event, f)
}

func WriteEvents(f mb.EventsFetcher, t *testing.T) error {

	if !*dataFlag {
		t.Skip("Skip data generation tests")
	}
	events, err := f.Fetch()
	if err != nil {
		return err
	}

	return createEvent(events[0], f)
}

func createEvent(event common.MapStr, m mb.MetricSet) error {

	path, err := os.Getwd()
	if err != nil {
		return err
	}

	startTime, _ := time.Parse(time.RFC3339Nano, "2016-05-23T08:05:34.853Z")

	build := beater.EventBuilder{
		ModuleName:    m.Module().Name(),
		MetricSetName: m.Name(),
		Host:          "localhost",
		StartTime:     startTime,
		FetchDuration: 115 * time.Microsecond,
		Event:         event,
	}

	fullEvent, _ := build.Build()

	// Delete meta data as not needed for the event output here
	delete(fullEvent, "_event_metadata")

	output, _ := json.MarshalIndent(fullEvent, "", "    ")

	err = ioutil.WriteFile(path+"/_meta/data.json", output, 0644)
	if err != nil {
		return err
	}
	return nil
}
