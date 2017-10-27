// +build integration windows

package services

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	config := map[string]interface{}{
		"module":     "windows",
		"metricsets": []string{"services"},
	}

	f := mbtest.NewEventsFetcher(t, config)
	f.Fetch()
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestReadService(t *testing.T) {
	reader, err := NewServiceReader()
	if err != nil {
		t.Fatal(err)
	}

	services, err := reader.Read()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(services)
}
