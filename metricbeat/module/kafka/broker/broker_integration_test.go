// +build integration
package broker

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/kafka"
)

func TestData(t *testing.T) {

	kafka.GenerateKafkaData(t)

	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}

}

func TestTopic(t *testing.T) {

	// Create initial topic
	kafka.GenerateKafkaData(t)

	f := mbtest.NewEventsFetcher(t, getConfig())
	data, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}

	t.Errorf("DATA: %+v", data)

	/*

		// Create 10 messages
		for i := 0; i < 10; i++ {
			generateKafkaData(t)
		}

		dataAfter, err := f.Fetch()
		if err != nil {
			t.Fatal("write", err)
		}*/

}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"broker"},
		"hosts":      []string{kafka.GetTestKafkaHost()},
	}
}
