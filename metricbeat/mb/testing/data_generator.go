package testing

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
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

	fullEvent := common.MapStr{
		"@timestamp": "2016-05-23T08:05:34.853Z",
		"beat": common.MapStr{
			"hostname": "host.example.com",
			"name":     "host.example.com",
		},
		"metricset": common.MapStr{
			"host":   "localhost",
			"module": m.Module().Name(),
			"name":   m.Name(),
			"rtt":    115,
		},
		m.Module().Name(): common.MapStr{
			m.Name(): event,
		},
		"type": "metricsets",
	}

	output, _ := json.MarshalIndent(fullEvent, "", "    ")

	err = ioutil.WriteFile(path+"/_meta/data.json", output, 0644)
	if err != nil {
		return err
	}
	return nil
}
