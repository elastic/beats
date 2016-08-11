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

func createElasticsearchConnection(flushInterval int, bulkSize int) elasticsearchOutput {
	index := fmt.Sprintf("packetbeat-int-test-%d", os.Getpid())

	esPort, err := strconv.Atoi(GetEsPort())

	if err != nil {
		logp.Err("Invalid port. Cannot be converted to in: %s", GetEsPort())
	}

	config, _ := common.NewConfigFrom(map[string]interface{}{
		"save_topology":    true,
		"hosts":            []string{GetEsHost()},
		"port":             esPort,
		"username":         os.Getenv("ES_USER"),
		"password":         os.Getenv("ES_PASS"),
		"path":             "",
		"index":            fmt.Sprintf("%v-%%{+yyyy.MM.dd}", index),
		"protocol":         "http",
		"flush_interval":   flushInterval,
		"bulk_max_size":    bulkSize,
		"template.enabled": false,
	})

	var output elasticsearchOutput
	output.init(config, 10)
	return output
}

func TestTopologyInES(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	elasticsearchOutput1 := createElasticsearchConnection(0, 0)
	elasticsearchOutput2 := createElasticsearchConnection(0, 0)
	elasticsearchOutput3 := createElasticsearchConnection(0, 0)

	elasticsearchOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	elasticsearchOutput2.PublishIPs("proxy2", []string{"10.1.0.9",
		"fe80::4e8d:79ff:fef2:de6a"})
	elasticsearchOutput3.PublishIPs("proxy3", []string{"10.1.0.10"})

	name2 := elasticsearchOutput3.GetNameByIP("10.1.0.9")
	if name2 != "proxy2" {
		t.Errorf("Failed to update proxy2 in topology: name=%s", name2)
	}

	elasticsearchOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	elasticsearchOutput2.PublishIPs("proxy2", []string{"10.1.0.9"})
	elasticsearchOutput3.PublishIPs("proxy3", []string{"192.168.1.2"})

	name3 := elasticsearchOutput3.GetNameByIP("192.168.1.2")
	if name3 != "proxy3" {
		t.Errorf("Failed to add a new IP")
	}

	name3 = elasticsearchOutput3.GetNameByIP("10.1.0.10")
	if name3 != "" {
		t.Errorf("Failed to delete old IP of proxy3: %s", name3)
	}

	name2 = elasticsearchOutput3.GetNameByIP("fe80::4e8d:79ff:fef2:de6a")
	if name2 != "" {
		t.Errorf("Failed to delete old IP of proxy2: %s", name2)
	}
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
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"topology", "output_elasticsearch"})
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

func testBulkWithParams(t *testing.T, output elasticsearchOutput) {
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
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"topology", "output_elasticsearch", "elasticsearch"})
	}

	output := createElasticsearchConnection(50, 2)
	testBulkWithParams(t, output)

	output = createElasticsearchConnection(50, 1000)
	testBulkWithParams(t, output)

	output = createElasticsearchConnection(50, 5)
	testBulkWithParams(t, output)
}
