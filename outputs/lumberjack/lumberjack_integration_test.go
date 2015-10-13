package lumberjack

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/outputs/elasticsearch"
	"github.com/stretchr/testify/assert"
)

const (
	lumberjackDefaultHost     = "localhost"
	lumberjackTestDefaultPort = "12345"

	elasticsearchDefaultHost = "localhost"
	elasticsearchDefaultPort = "9200"
)

type esConnection struct {
	*elasticsearch.Elasticsearch
	t     *testing.T
	index string
}

type testOutputer struct {
	outputs.BulkOutputer
	*esConnection
}

type esValueReader interface {
	Read() ([]map[string]interface{}, error)
}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func getLumberjackHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("LS_HOST", lumberjackDefaultHost),
		getenv("LS_LUMBERJACK_TCP_PORT", lumberjackTestDefaultPort),
	)
}

func getElasticsearchHost() string {
	return fmt.Sprintf("http://%v:%v",
		getenv("ES_HOST", elasticsearchDefaultHost),
		getenv("ES_PORT", elasticsearchDefaultPort),
	)
}

func esConnect(t *testing.T, index string) *esConnection {
	ts := time.Now()

	host := getElasticsearchHost()
	index = fmt.Sprintf("%s-%02d.%02d.%02d",
		index, ts.Year(), ts.Month(), ts.Day())

	connection := elasticsearch.NewElasticsearch([]string{host}, nil, "", "")

	// try to drop old index if left over from failed test
	_, _ = connection.Delete(index, "", "", nil) // ignore error

	_, err := connection.CreateIndex(index, common.MapStr{
		"settings": common.MapStr{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	})
	if err != nil {
		t.Fatalf("failed to create test index: %s", err)
	}

	es := &esConnection{}
	es.t = t
	es.Elasticsearch = connection
	es.index = index
	return es
}

func testElasticsearchIndex(test string) string {
	return fmt.Sprintf("beat-es-int-%v-%d", test, os.Getpid())
}

func newTestLogstashOutput(t *testing.T, test string) *testOutputer {
	lumberjack := newTestLumberjackOutput(t, test, nil)
	index := testLogstashIndex(test)
	connection := esConnect(t, index)

	ls := &testOutputer{}
	ls.BulkOutputer = lumberjack
	ls.esConnection = connection
	return ls
}

func newTestElasticsearchOutput(t *testing.T, test string) *testOutputer {
	plugin := outputs.FindOutputPlugin("elasticsearch")
	if plugin == nil {
		t.Fatalf("No elasticsearch output plugin found")
	}

	index := testElasticsearchIndex(test)
	connection := esConnect(t, index)

	flushInterval := 0
	bulkSize := 0
	config := outputs.MothershipConfig{
		Enabled:        true,
		Hosts:          []string{getElasticsearchHost()},
		Index:          index,
		Flush_interval: &flushInterval,
		Bulk_size:      &bulkSize,
	}

	output, err := plugin.NewOutput("test", &config, 10)
	if err != nil {
		t.Fatalf("init elasticsearch output plugin failed: %v", err)
	}

	es := &testOutputer{}
	es.BulkOutputer = output.(outputs.BulkOutputer)
	es.esConnection = connection
	return es
}

func (es *esConnection) Cleanup() {
	_, err := es.Delete(es.index, "", "", nil)
	if err != nil {
		es.t.Errorf("Failed to delete index: %s", err)
	}
}

func (es *esConnection) Read() ([]map[string]interface{}, error) {
	_, err := es.Refresh(es.index)
	if err != nil {
		es.t.Errorf("Failed to refresh: %s", err)
	}

	params := map[string]string{}
	resp, err := es.SearchURI(es.index, "", params)
	if err != nil {
		es.t.Errorf("Failed to query elasticsearch for index(%s): %s", es.index, err)
		return nil, err
	}

	hits := make([]map[string]interface{}, len(resp.Hits.Hits))
	for i, hit := range resp.Hits.Hits {
		json.Unmarshal(hit, &hits[i])
	}

	return hits, err
}

func waitUntilTrue(duration time.Duration, fn func() bool) bool {
	end := time.Now().Add(duration)
	for time.Now().Before(end) {
		if fn() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func checkIndex(reader esValueReader, minValues int) func() bool {
	return func() bool {
		resp, err := reader.Read()
		return err != nil || len(resp) >= minValues
	}
}

func checkAll(checks ...func() bool) func() bool {
	return func() bool {
		for _, check := range checks {
			if !check() {
				return false
			}
		}
		return true
	}
}

func TestSendMessageViaLogstash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	test := "basic"
	ls := newTestLogstashOutput(t, test)
	defer ls.Cleanup()

	event := common.MapStr{
		"timestamp": common.Time(time.Now()),
		"host":      "test-host",
		"type":      "log",
		"message":   "hello world",
	}
	ls.PublishEvent(nil, time.Now(), event)

	// wait for logstash event flush + elasticsearch
	waitUntilTrue(5*time.Second, checkIndex(ls, 1))

	// search value in logstash elasticsearch index
	resp, err := ls.Read()
	if err != nil {
		return
	}
	if len(resp) != 1 {
		t.Errorf("wrong number of results: %d", len(resp))
	}
}

func TestSendMultipleViaLogstash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	test := "multiple"
	ls := newTestLogstashOutput(t, test)
	defer ls.Cleanup()

	for i := 0; i < 10; i++ {
		event := common.MapStr{
			"timestamp": common.Time(time.Now()),
			"host":      "test-host",
			"type":      "log",
			"message":   fmt.Sprintf("hello world - %v", i),
		}
		ls.PublishEvent(nil, time.Now(), event)
	}

	// wait for logstash event flush + elasticsearch
	waitUntilTrue(5*time.Second, checkIndex(ls, 10))

	// search value in logstash elasticsearch index
	resp, err := ls.Read()
	if err != nil {
		return
	}
	if len(resp) != 10 {
		t.Errorf("wrong number of results: %d", len(resp))
	}
}

func TestLogstashElasticOutputPluginCompatibleMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	test := "cmp"
	timeout := 10 * time.Second

	ls := newTestLogstashOutput(t, test)
	defer ls.Cleanup()

	es := newTestElasticsearchOutput(t, test)
	defer es.Cleanup()

	ts := time.Now()
	event := common.MapStr{
		"timestamp": common.Time(ts),
		"host":      "test-host",
		"type":      "log",
		"message":   "hello world",
	}

	es.PublishEvent(nil, ts, event)
	waitUntilTrue(timeout, checkIndex(es, 1))

	ls.PublishEvent(nil, ts, event)
	waitUntilTrue(timeout, checkIndex(ls, 1))

	// search value in logstash elasticsearch index
	lsResp, err := ls.Read()
	if err != nil {
		return
	}
	esResp, err := es.Read()
	if err != nil {
		return
	}

	// validate
	assert.Equal(t, len(lsResp), len(esResp))
	if len(lsResp) != 1 {
		t.Fatalf("wrong number of results: %d", len(lsResp))
	}

	checkEvent(t, lsResp[0], esResp[0])
}

func TestLogstashElasticOutputPluginBulkCompatibleMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	test := "cmpbulk"
	timeout := 10 * time.Second

	ls := newTestLogstashOutput(t, test)
	defer ls.Cleanup()

	es := newTestElasticsearchOutput(t, test)
	defer es.Cleanup()

	ts := time.Now()
	events := []common.MapStr{
		common.MapStr{
			"timestamp": common.Time(ts),
			"host":      "test-host",
			"type":      "log",
			"message":   "hello world",
		},
	}
	es.BulkPublish(nil, ts, events)
	waitUntilTrue(timeout, checkIndex(es, 1))

	ls.BulkPublish(nil, ts, events)
	waitUntilTrue(timeout, checkIndex(ls, 1))

	// search value in logstash elasticsearch index
	lsResp, err := ls.Read()
	if err != nil {
		return
	}
	esResp, err := es.Read()
	if err != nil {
		return
	}

	// validate
	assert.Equal(t, len(lsResp), len(esResp))
	if len(lsResp) != 1 {
		t.Fatalf("wrong number of results: %d", len(lsResp))
	}

	checkEvent(t, lsResp[0], esResp[0])
}

func checkEvent(t *testing.T, ls, es map[string]interface{}) {
	lsEvent := ls["_source"].(map[string]interface{})
	esEvent := es["_source"].(map[string]interface{})
	commonFields := []string{"timestamp", "host", "type", "message"}
	for _, field := range commonFields {
		assert.NotNil(t, lsEvent[field])
		assert.NotNil(t, esEvent[field])
		assert.Equal(t, lsEvent[field], esEvent[field])
	}
}
