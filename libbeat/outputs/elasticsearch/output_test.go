// +build integration

package elasticsearch

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
)

var testOptions = outputs.Options{}

func createElasticsearchConnection(flushInterval int, bulkSize int) *elasticsearchOutput {
	index := fmt.Sprintf("packetbeat-int-test-%d", os.Getpid())

	esPort, err := strconv.Atoi(GetEsPort())

	if err != nil {
		logp.Err("Invalid port. Cannot be converted to in: %s", GetEsPort())
	}

	config, _ := common.NewConfigFrom(map[string]interface{}{
		"hosts":          []string{GetEsHost()},
		"port":           esPort,
		"username":       os.Getenv("ES_USER"),
		"password":       os.Getenv("ES_PASS"),
		"path":           "",
		"index":          fmt.Sprintf("%v-%%{+yyyy.MM.dd}", index),
		"protocol":       "http",
		"flush_interval": flushInterval,
		"bulk_max_size":  bulkSize,
	})

	output := &elasticsearchOutput{beat: common.BeatInfo{Beat: "test"}}
	output.init(config)
	return output
}

func TestOneEvent(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch", "output_elasticsearch"})
	}

	ts := time.Now()

	output := createElasticsearchConnection(0, 0)

	event := common.MapStr{}
	event["@timestamp"] = common.Time(ts)
	event["type"] = "redis"
	event["status"] = "OK"
	event["responsetime"] = 34
	event["dst_ip"] = "192.168.21.1"
	event["dst_port"] = 6379
	event["src_ip"] = "192.168.22.2"
	event["src_port"] = 6378
	event["name"] = "appserver1"
	r := common.MapStr{}
	r["request"] = "MGET key1"
	r["response"] = "value1"

	index, _ := output.index.Select(event)
	debugf("index = %s", index)

	client := output.randomClient()
	client.CreateIndex(index, common.MapStr{
		"settings": common.MapStr{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	})

	err := output.PublishEvent(nil, testOptions, outputs.Data{Event: event})
	if err != nil {
		t.Errorf("Failed to publish the event: %s", err)
	}

	// give control to the other goroutine, otherwise the refresh happens
	// before the refresh. We should find a better solution for this.
	time.Sleep(200 * time.Millisecond)

	_, _, err = client.Refresh(index)
	if err != nil {
		t.Errorf("Failed to refresh: %s", err)
	}

	defer func() {
		_, _, err = client.Delete(index, "", "", nil)
		if err != nil {
			t.Errorf("Failed to delete index: %s", err)
		}
	}()

	params := map[string]string{
		"q": "name:appserver1",
	}
	_, resp, err := client.SearchURI(index, "", params)

	if err != nil {
		t.Errorf("Failed to query elasticsearch for index(%s): %s", index, err)
		return
	}
	debugf("resp = %s", resp)
	if resp.Hits.Total != 1 {
		t.Errorf("Wrong number of results: %d", resp.Hits.Total)
	}

}

func TestEvents(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"output_elasticsearch"})
	}

	ts := time.Now()

	output := createElasticsearchConnection(0, 0)

	event := common.MapStr{}
	event["@timestamp"] = common.Time(ts)
	event["type"] = "redis"
	event["status"] = "OK"
	event["responsetime"] = 34
	event["dst_ip"] = "192.168.21.1"
	event["dst_port"] = 6379
	event["src_ip"] = "192.168.22.2"
	event["src_port"] = 6378
	event["name"] = "appserver1"
	r := common.MapStr{}
	r["request"] = "MGET key1"
	r["response"] = "value1"
	event["redis"] = r

	index, _ := output.index.Select(event)
	output.randomClient().CreateIndex(index, common.MapStr{
		"settings": common.MapStr{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	})

	err := output.PublishEvent(nil, testOptions, outputs.Data{Event: event})
	if err != nil {
		t.Errorf("Failed to publish the event: %s", err)
	}

	r = common.MapStr{}
	r["request"] = "MSET key1 value1"
	r["response"] = 0
	event["redis"] = r

	err = output.PublishEvent(nil, testOptions, outputs.Data{Event: event})
	if err != nil {
		t.Errorf("Failed to publish the event: %s", err)
	}

	// give control to the other goroutine, otherwise the refresh happens
	// before the refresh. We should find a better solution for this.
	time.Sleep(200 * time.Millisecond)

	output.randomClient().Refresh(index)

	params := map[string]string{
		"q": "name:appserver1",
	}

	defer func() {
		_, _, err = output.randomClient().Delete(index, "", "", nil)
		if err != nil {
			t.Errorf("Failed to delete index: %s", err)
		}
	}()

	_, resp, err := output.randomClient().SearchURI(index, "", params)

	if err != nil {
		t.Errorf("Failed to query elasticsearch: %s", err)
	}
	if resp.Hits.Total != 2 {
		t.Errorf("Wrong number of results: %d", resp.Hits.Total)
	}
}

func testBulkWithParams(t *testing.T, output *elasticsearchOutput) {
	ts := time.Now()
	index, _ := output.index.Select(common.MapStr{
		"@timestamp": common.Time(ts),
	})

	output.randomClient().CreateIndex(index, common.MapStr{
		"settings": common.MapStr{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	})

	for i := 0; i < 10; i++ {

		event := common.MapStr{}
		event["@timestamp"] = common.Time(time.Now())
		event["type"] = "redis"
		event["status"] = "OK"
		event["responsetime"] = 34
		event["dst_ip"] = "192.168.21.1"
		event["dst_port"] = 6379
		event["src_ip"] = "192.168.22.2"
		event["src_port"] = 6378
		event["shipper"] = "appserver" + strconv.Itoa(i)
		r := common.MapStr{}
		r["request"] = "MGET key" + strconv.Itoa(i)
		r["response"] = "value" + strconv.Itoa(i)
		event["redis"] = r

		err := output.PublishEvent(nil, testOptions, outputs.Data{Event: event})
		if err != nil {
			t.Errorf("Failed to publish the event: %s", err)
		}

	}

	// give control to the other goroutine, otherwise the refresh happens
	// before the index. We should find a better solution for this.
	time.Sleep(200 * time.Millisecond)

	output.randomClient().Refresh(index)

	params := map[string]string{
		"q": "type:redis",
	}

	defer func() {
		_, _, err := output.randomClient().Delete(index, "", "", nil)
		if err != nil {
			t.Errorf("Failed to delete index: %s", err)
		}
	}()

	_, resp, err := output.randomClient().SearchURI(index, "", params)

	if err != nil {
		t.Errorf("Failed to query elasticsearch: %s", err)
		return
	}
	if resp.Hits.Total != 10 {
		t.Errorf("Wrong number of results: %d", resp.Hits.Total)
	}
}

func TestBulkEvents(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"output_elasticsearch", "elasticsearch"})
	}

	testBulkWithParams(t, createElasticsearchConnection(50, 2))
	testBulkWithParams(t, createElasticsearchConnection(50, 1000))
	testBulkWithParams(t, createElasticsearchConnection(50, 5))
}
