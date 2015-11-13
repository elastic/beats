package logstash

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
	logstashDefaultHost        = "localhost"
	logstashTestDefaultPort    = "5044"
	logstashTestDefaultTLSPort = "5055"

	elasticsearchDefaultHost = "localhost"
	elasticsearchDefaultPort = "9200"

	integrationTestWindowSize = 32
)

type esConnection struct {
	*elasticsearch.Client
	t     *testing.T
	index string
}

type testOutputer struct {
	outputs.BulkOutputer
	*esConnection
}

type esSoure interface {
	RefreshIndex()
}

type esValueReader interface {
	esSoure
	Read() ([]map[string]interface{}, error)
}

type esCountReader interface {
	esSoure
	Count() (int, error)
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

func getLogstashHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("LS_HOST", logstashDefaultHost),
		getenv("LS_TCP_PORT", logstashTestDefaultPort),
	)
}

func getLogstashTLSHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("LS_HOST", logstashDefaultHost),
		getenv("LS_LS_PORT", logstashTestDefaultTLSPort),
	)
}

func getElasticsearchHost() string {
	return fmt.Sprintf("http://%v:%v",
		getenv("ES_HOST", elasticsearchDefaultHost),
		getenv("ES_PORT", elasticsearchDefaultPort),
	)
}

func esConnect(t *testing.T, index string) *esConnection {
	ts := time.Now().UTC()

	host := getElasticsearchHost()
	index = fmt.Sprintf("%s-%02d.%02d.%02d",
		index, ts.Year(), ts.Month(), ts.Day())

	client := elasticsearch.NewClient(host, "", nil, "", "")

	// try to drop old index if left over from failed test
	_, _, _ = client.Delete(index, "", "", nil) // ignore error

	_, _, err := client.CreateIndex(index, common.MapStr{
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
	es.Client = client
	es.index = index
	return es
}

func testElasticsearchIndex(test string) string {
	return fmt.Sprintf("beat-es-int-%v-%d", test, os.Getpid())
}

func newTestLogstashOutput(t *testing.T, test string, tls bool) *testOutputer {
	windowSize := integrationTestWindowSize

	config := &outputs.MothershipConfig{
		Hosts:       []string{getLogstashHost()},
		TLS:         nil,
		Index:       testLogstashIndex(test),
		BulkMaxSize: &windowSize,
	}
	if tls {
		config.Hosts = []string{getLogstashTLSHost()}
		config.TLS = &outputs.TLSConfig{
			Insecure: false,
			CAs: []string{
				"/etc/pki/tls/certs/logstash.crt",
			},
		}
	}

	lumberjack := newTestLumberjackOutput(t, test, config)
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
		Hosts:          []string{getElasticsearchHost()},
		Index:          index,
		Flush_interval: &flushInterval,
		BulkMaxSize:    &bulkSize,
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
	_, _, err := es.Delete(es.index, "", "", nil)
	if err != nil {
		es.t.Errorf("Failed to delete index: %s", err)
	}
}

func (es *esConnection) Read() ([]map[string]interface{}, error) {
	_, _, err := es.Refresh(es.index)
	if err != nil {
		es.t.Errorf("Failed to refresh: %s", err)
	}

	params := map[string]string{}
	_, resp, err := es.SearchURI(es.index, "", params)
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

func (es *esConnection) RefreshIndex() {
	es.Refresh(es.index)
}

func (es *esConnection) Count() (int, error) {
	_, _, err := es.Refresh(es.index)
	if err != nil {
		es.t.Errorf("Failed to refresh: %s", err)
	}

	params := map[string]string{}
	_, resp, err := es.CountSearchURI(es.index, "", params)
	if err != nil {
		es.t.Errorf("Failed to query elasticsearch for index(%s): %s", es.index, err)
		return 0, err
	}

	return resp.Count, nil
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

func checkIndex(reader esCountReader, minValues int) func() bool {
	return func() bool {
		reader.RefreshIndex()
		resp, err := reader.Count()
		return err != nil || resp >= minValues
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

func TestSendMessageViaLogstashTCP(t *testing.T) {
	testSendMessageViaLogstash(t, "basic-tcp", false)
}

func TestSendMessageViaLogstashTLS(t *testing.T) {
	testSendMessageViaLogstash(t, "basic-tls", true)
}

func testSendMessageViaLogstash(t *testing.T, name string, tls bool) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	ls := newTestLogstashOutput(t, name, tls)
	defer ls.Cleanup()

	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"host":       "test-host",
		"type":       "log",
		"message":    "hello world",
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

func TestSendMultipleViaLogstashTCP(t *testing.T) {
	testSendMultipleViaLogstash(t, "multiple-tcp", false)
}

func TestSendMultipleViaLogstashTLS(t *testing.T) {
	testSendMultipleViaLogstash(t, "multiple-tls", true)
}

func testSendMultipleViaLogstash(t *testing.T, name string, tls bool) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	ls := newTestLogstashOutput(t, name, tls)
	defer ls.Cleanup()
	for i := 0; i < 10; i++ {
		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"host":       "test-host",
			"type":       "log",
			"message":    fmt.Sprintf("hello world - %v", i),
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

func TestSendMultipleBigBatchesViaLogstashTCP(t *testing.T) {
	testSendMultipleBigBatchesViaLogstash(t, "multiple-big-tcp", false)
}

func TestSendMultipleBigBatchesViaLogstashTLS(t *testing.T) {
	testSendMultipleBigBatchesViaLogstash(t, "multiple-big-tls", true)
}

func testSendMultipleBigBatchesViaLogstash(t *testing.T, name string, tls bool) {
	testSendMultipleBatchesViaLogstash(t, name, 15, 4*integrationTestWindowSize, tls)
}

func TestSendMultipleSmallBatchesViaLogstashTCP(t *testing.T) {
	testSendMultipleSmallBatchesViaLogstash(t, "multiple-small-tcp", false)
}

func TestSendMultipleSmallBatchesViaLogstashTLS(t *testing.T) {
	testSendMultipleSmallBatchesViaLogstash(t, "multiple-small-tls", true)
}

func testSendMultipleSmallBatchesViaLogstash(t *testing.T, name string, tls bool) {
	testSendMultipleBatchesViaLogstash(t, name, 15, integrationTestWindowSize/2, tls)
}

func testSendMultipleBatchesViaLogstash(
	t *testing.T,
	name string,
	numBatches int,
	batchSize int,
	tls bool,
) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	ls := newTestLogstashOutput(t, name, tls)
	defer ls.Cleanup()

	batches := make([][]common.MapStr, 0, numBatches)
	for i := 0; i < numBatches; i++ {
		batch := make([]common.MapStr, 0, batchSize)
		for j := 0; j < batchSize; j++ {
			event := common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"host":       "test-host",
				"type":       "log",
				"message":    fmt.Sprintf("batch hello world - %v", i*batchSize+j),
			}
			batch = append(batch, event)
		}
		batches = append(batches, batch)
	}

	for _, batch := range batches {
		sig := outputs.NewSyncSignal()
		ls.BulkPublish(sig, time.Now(), batch)
		ok := sig.Wait()
		assert.Equal(t, true, ok)
	}

	// wait for logstash event flush + elasticsearch
	ok := waitUntilTrue(5*time.Second, checkIndex(ls, numBatches*batchSize))
	assert.True(t, ok) // check number of events matches total number of events

	// search value in logstash elasticsearch index
	resp, err := ls.Read()
	if err != nil {
		return
	}
	if len(resp) != 10 {
		t.Errorf("wrong number of results: %d", len(resp))
	}
}

func TestLogstashElasticOutputPluginCompatibleMessageTCP(t *testing.T) {
	testLogstashElasticOutputPluginCompatibleMessage(t, "cmp-tcp", false)
}

func TestLogstashElasticOutputPluginCompatibleMessageTLS(t *testing.T) {
	testLogstashElasticOutputPluginCompatibleMessage(t, "cmp-tls", true)
}

func testLogstashElasticOutputPluginCompatibleMessage(t *testing.T, name string, tls bool) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	timeout := 10 * time.Second

	ls := newTestLogstashOutput(t, name, tls)
	defer ls.Cleanup()

	es := newTestElasticsearchOutput(t, name)
	defer es.Cleanup()

	ts := time.Now()
	event := common.MapStr{
		"@timestamp": common.Time(ts),
		"host":       "test-host",
		"type":       "log",
		"message":    "hello world",
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

func TestLogstashElasticOutputPluginBulkCompatibleMessageTCP(t *testing.T) {
	testLogstashElasticOutputPluginBulkCompatibleMessage(t, "cmpblk-tcp", false)
}

func TestLogstashElasticOutputPluginBulkCompatibleMessageTLS(t *testing.T) {
	testLogstashElasticOutputPluginBulkCompatibleMessage(t, "cmpblk-tls", true)
}

func testLogstashElasticOutputPluginBulkCompatibleMessage(t *testing.T, name string, tls bool) {
	if testing.Short() {
		t.Skip("Skipping in short mode. Requires Logstash and Elasticsearch")
	}

	timeout := 10 * time.Second

	ls := newTestLogstashOutput(t, name, tls)
	defer ls.Cleanup()

	es := newTestElasticsearchOutput(t, name)
	defer es.Cleanup()

	ts := time.Now()
	events := []common.MapStr{
		common.MapStr{
			"@timestamp": common.Time(ts),
			"host":       "test-host",
			"type":       "log",
			"message":    "hello world",
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
	commonFields := []string{"@timestamp", "host", "type", "message"}
	for _, field := range commonFields {
		assert.NotNil(t, lsEvent[field])
		assert.NotNil(t, esEvent[field])
		assert.Equal(t, lsEvent[field], esEvent[field])
	}
}
