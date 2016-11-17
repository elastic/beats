// +build integration

package consumer

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
	dataBefore, err := f.Fetch()
	if err != nil {
		t.Fatal("write", err)
	}

	t.Errorf("%v", dataBefore)
	/*

		var n int64 = 10
		var i int64 = 0
		// Create n messages
		for ; i < n; i++ {
			kafka.GenerateKafkaData(t)
		}

		dataAfter, err := f.Fetch()
		if err != nil {
			t.Fatal("write", err)
		}*/

	// Checks that no new topics / partitions were added
	//assert.True(t, len(dataBefore) == len(dataAfter))

	// Compares offset before and after
	/*offsetBefore := dataBefore[0]["offset"].(common.MapStr)["newest"].(int64)
	offsetAfter := dataAfter[0]["offset"].(common.MapStr)["newest"].(int64)

	if offsetBefore+n != offsetAfter {
		t.Errorf("Offset before: %v", offsetBefore)
		t.Errorf("Offset after: %v", offsetAfter)
	}
	assert.True(t, offsetBefore+n == offsetAfter)*/

}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"consumer"},
		"hosts":      []string{kafka.GetTestKafkaHost()},
	}
}
