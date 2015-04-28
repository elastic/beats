package elasticsearch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
)

const elasticsearchAddr = "localhost"
const elasticsearchPort = 9200

func createElasticsearchConnection() ElasticsearchOutput {

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	var elasticsearchOutput ElasticsearchOutput
	elasticsearchOutput.Init(outputs.MothershipConfig{
		Enabled:       true,
		Save_topology: true,
		Host:          elasticsearchAddr,
		Port:          elasticsearchPort,
		Username:      "",
		Password:      "",
		Path:          "",
		Index:         index,
		Protocol:      "",
	}, 10)

	return elasticsearchOutput
}

func TestTopologyInES(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping topology tests in short mode, because they require Elasticsearch")
	}

	elasticsearchOutput1 := createElasticsearchConnection()
	elasticsearchOutput2 := createElasticsearchConnection()
	elasticsearchOutput3 := createElasticsearchConnection()

	elasticsearchOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	elasticsearchOutput2.PublishIPs("proxy2", []string{"10.1.0.9",
		"fe80::4e8d:79ff:fef2:de6a"})
	elasticsearchOutput3.PublishIPs("proxy3", []string{"10.1.0.10"})

	// give some time to Elasticsearch to add the IPs
	// TODO: just needs _refresh=true instead?
	time.Sleep(1 * time.Second)

	elasticsearchOutput3.UpdateLocalTopologyMap()

	name2 := elasticsearchOutput3.GetNameByIP("10.1.0.9")
	if name2 != "proxy2" {
		t.Errorf("Failed to update proxy2 in topology: name=%s", name2)
	}

	elasticsearchOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	elasticsearchOutput2.PublishIPs("proxy2", []string{"10.1.0.9"})
	elasticsearchOutput3.PublishIPs("proxy3", []string{"192.168.1.2"})

	// give some time to Elasticsearch to add the IPs
	time.Sleep(1 * time.Second)

	elasticsearchOutput3.UpdateLocalTopologyMap()

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

func TestEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping events publish in short mode, because they require Elasticsearch")
	}
	ts := time.Now()

	elasticsearchOutput := createElasticsearchConnection()

	event := common.MapStr{}
	event["type"] = "redis"
	event["status"] = "OK"
	event["responsetime"] = 34
	event["dst_ip"] = "192.168.21.1"
	event["dst_port"] = 6379
	event["src_ip"] = "192.168.22.2"
	event["src_port"] = 6378
	event["agent"] = "appserver1"
	r := common.MapStr{}
	r["request"] = "MGET key1"
	r["response"] = "value1"

	index := fmt.Sprintf("%s-%d.%02d.%02d", elasticsearchOutput.Index, ts.Year(), ts.Month(), ts.Day())

	es := NewElasticsearch("http://localhost:9200")

	if es == nil {
		t.Errorf("Failed to create Elasticsearch connection")
	}
	_, err := es.DeleteIndex(index)
	if err != nil {
		t.Errorf("Failed to delete index: %s", err)
	}

	err = elasticsearchOutput.PublishEvent(ts, event)
	if err != nil {
		t.Errorf("Failed to publish the event: %s", err)
	}

	es.Refresh(index)

	resp, err := es.Search(index, "?search?q=agent:appserver1", "{}")

	if err != nil {
		t.Errorf("Failed to query elasticsearch: %s", err)
	}
	defer resp.Body.Close()
	objresp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Failed to read body from response")
	}
	var search_res ESSearchResults
	err = json.Unmarshal(objresp, &search_res)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %s", err)
	}
	if search_res.Hits.Total != 1 {
		t.Errorf("Too many results")
	}
}
