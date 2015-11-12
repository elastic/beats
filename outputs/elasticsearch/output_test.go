package elasticsearch

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

func createElasticsearchConnection(flushInterval int, bulkSize int) elasticsearchOutput {
	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	esPort, err := strconv.Atoi(GetEsPort())

	if err != nil {
		logp.Err("Invalid port. Cannot be converted to in: %s", GetEsPort())
	}

	var output elasticsearchOutput
	output.init("packetbeat", outputs.MothershipConfig{
		Save_topology:  true,
		Host:           GetEsHost(),
		Port:           esPort,
		Username:       "",
		Password:       "",
		Path:           "",
		Index:          index,
		Protocol:       "http",
		Flush_interval: &flushInterval,
		BulkMaxSize:    &bulkSize,
	}, 10)

	return output
}

func TestTopologyInES(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping topology tests in short mode, because they require Elasticsearch")
	}
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"topology", "output_elasticsearch"})
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
	if testing.Short() {
		t.Skip("Skipping events publish in short mode, because they require Elasticsearch")
	}
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch", "output_elasticsearch"})
	}

	ts := time.Now()

	output := createElasticsearchConnection(0, 0)

	event := common.MapStr{}
	event["@timestamp"] = common.Time(time.Now())
	event["type"] = "redis"
	event["status"] = "OK"
	event["responsetime"] = 34
	event["dst_ip"] = "192.168.21.1"
	event["dst_port"] = 6379
	event["src_ip"] = "192.168.22.2"
	event["src_port"] = 6378
	event["shipper"] = "appserver1"
	r := common.MapStr{}
	r["request"] = "MGET key1"
	r["response"] = "value1"

	index := fmt.Sprintf("%s-%d.%02d.%02d", output.index, ts.Year(), ts.Month(), ts.Day())
	logp.Debug("output_elasticsearch", "index = %s", index)

	client := output.randomClient()
	client.CreateIndex(index, common.MapStr{
		"settings": common.MapStr{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	})

	err := output.PublishEvent(nil, ts, event)
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
		"q": "shipper:appserver1",
	}
	_, resp, err := client.SearchURI(index, "", params)

	if err != nil {
		t.Errorf("Failed to query elasticsearch for index(%s): %s", index, err)
		return
	}
	logp.Debug("output_elasticsearch", "resp = %s", resp)
	if resp.Hits.Total != 1 {
		t.Errorf("Wrong number of results: %d", resp.Hits.Total)
	}

}

func TestEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping events publish in short mode, because they require Elasticsearch")
	}
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"topology", "output_elasticsearch"})
	}

	ts := time.Now()

	output := createElasticsearchConnection(0, 0)

	event := common.MapStr{}
	event["@timestamp"] = common.Time(time.Now())
	event["type"] = "redis"
	event["status"] = "OK"
	event["responsetime"] = 34
	event["dst_ip"] = "192.168.21.1"
	event["dst_port"] = 6379
	event["src_ip"] = "192.168.22.2"
	event["src_port"] = 6378
	event["shipper"] = "appserver1"
	r := common.MapStr{}
	r["request"] = "MGET key1"
	r["response"] = "value1"
	event["redis"] = r

	index := fmt.Sprintf("%s-%d.%02d.%02d", output.index, ts.Year(), ts.Month(), ts.Day())
	output.randomClient().CreateIndex(index, common.MapStr{
		"settings": common.MapStr{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	})

	err := output.PublishEvent(nil, ts, event)
	if err != nil {
		t.Errorf("Failed to publish the event: %s", err)
	}

	r = common.MapStr{}
	r["request"] = "MSET key1 value1"
	r["response"] = 0
	event["redis"] = r

	err = output.PublishEvent(nil, ts, event)
	if err != nil {
		t.Errorf("Failed to publish the event: %s", err)
	}

	// give control to the other goroutine, otherwise the refresh happens
	// before the refresh. We should find a better solution for this.
	time.Sleep(200 * time.Millisecond)

	output.randomClient().Refresh(index)

	params := map[string]string{
		"q": "shipper:appserver1",
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
	index := fmt.Sprintf("%s-%d.%02d.%02d", output.index, ts.Year(), ts.Month(), ts.Day())

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

		err := output.PublishEvent(nil, ts, event)
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
	if testing.Short() {
		t.Skip("Skipping events publish in short mode, because they require Elasticsearch")
	}
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

func TestEnableTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping events publish in short mode, because they require Elasticsearch")
	}
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"topology", "output_elasticsearch", "elasticsearch"})
	}

	output := createElasticsearchConnection(0, 0)
	output.randomClient().Delete(".packetbeat-topology", "", "", nil)

	err := output.EnableTTL()
	if err != nil {
		t.Errorf("Fail to enable TTL: %s", err)
	}

	// should succeed also when index already exists
	err = output.EnableTTL()
	if err != nil {
		t.Errorf("Fail to enable TTL: %s", err)
	}
}
