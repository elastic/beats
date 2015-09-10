package elasticsearch

import (
	"fmt"
	"github.com/elastic/libbeat/logp"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

const ElasticsearchDefaultHost = "localhost"
const ElasticsearchDefaultPort = "9200"

func GetEsPort() string {
	port := os.Getenv("ES_PORT")

	if len(port) == 0 {
		port = ElasticsearchDefaultPort
	}
	return port
}

// Returns
func GetEsHost() string {

	host := os.Getenv("ES_HOST")

	if len(host) == 0 {
		host = ElasticsearchDefaultHost
	}

	return host
}

func GetTestingElasticsearch() *Elasticsearch {

	var es_url = "http://" + GetEsHost() + ":" + GetEsPort()

	return NewElasticsearch([]string{es_url}, "", "")
}

func GetValidQueryResult() QueryResult {
	result := QueryResult{
		Ok:    true,
		Index: "testIndex",
		Type:  "testType",
		Id:    "12",
		Source: []byte(`{
			"ok": true,
			"_type":"testType",
			"_index":"testIndex",
			"_id":"12",
			"_version": 2,
			"found": true,
			"exists": true,
			"created": true,
			"lastname":"ruflin",
			"firstname": "nicolas"}`,
		),
		Version: 2,
		Found:   true,
		Exists:  true,
		Created: true,
		Matches: []string{"abc", "def"},
	}

	return result
}

func GetValidSearchResults() SearchResults {

	hits := Hits{
		Total: 0,
		Hits:  nil,
	}

	results := SearchResults{
		Took: 19,
		Shards: []byte(`{
    		"total" : 3,
    		"successful" : 2,
    		"failed" : 1
  		}`),
		Hits: hits,
		Aggs: nil,
	}

	return results
}

func TestUrlEncode(t *testing.T) {

	params := map[string]string{
		"q": "agent:appserver1",
	}
	url := UrlEncode(params)

	if url != "q=agent%3Aappserver1" {
		t.Errorf("Fail to encode params: %s", url)
	}

	params = map[string]string{
		"wife":    "sarah",
		"husband": "joe",
	}

	url = UrlEncode(params)

	if url != "husband=joe&wife=sarah" {
		t.Errorf("Fail to encode params: %s", url)
	}
}

func TestMakePath(t *testing.T) {
	path, err := MakePath("twitter", "tweet", "1")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter/tweet/1" {
		t.Errorf("Wrong path created: %s", path)
	}

	path, err = MakePath("twitter", "", "_refresh")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter/_refresh" {
		t.Errorf("Wrong path created: %s", path)
	}

	path, err = MakePath("", "", "_bulk")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/_bulk" {
		t.Errorf("Wrong path created: %s", path)
	}
	path, err = MakePath("twitter", "", "")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter" {
		t.Errorf("Wrong path created: %s", path)
	}

}

func TestIndex(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires Elasticsearch")
	}

	es := GetTestingElasticsearch()

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}
	params := map[string]string{
		"refresh": "true",
	}
	resp, err := es.Index(index, "test", "1", params, body)
	if err != nil {
		t.Errorf("Index() returns error: %s", err)
	}
	if !resp.Created {
		t.Errorf("Index() fails: %s", resp)
	}

	params = map[string]string{
		"q": "user:test",
	}
	result, err := es.SearchUri(index, "test", params)
	if err != nil {
		t.Errorf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total)
	}

	resp, err = es.Delete(index, "test", "1", nil)
	if err != nil {
		t.Errorf("Delete() returns error: %s", err)
	}
}

func TestReadQueryResult(t *testing.T) {

	queryResult := GetValidQueryResult()

	json := queryResult.Source
	result, err := ReadQueryResult(json)

	assert.Nil(t, err)
	assert.Equal(t, queryResult.Ok, result.Ok)
	assert.Equal(t, queryResult.Index, result.Index)
	assert.Equal(t, queryResult.Type, result.Type)
	assert.Equal(t, queryResult.Id, result.Id)
	assert.Equal(t, queryResult.Version, result.Version)
	assert.Equal(t, queryResult.Found, result.Found)
	assert.Equal(t, queryResult.Exists, result.Exists)
	assert.Equal(t, queryResult.Created, result.Created)
}

// Check empty query result object
func TestReadQueryResult_empty(t *testing.T) {
	result, err := ReadQueryResult(nil)
	assert.Nil(t, result)
	assert.Nil(t, err)
}

// Check invalid query result object
func TestReadQueryResult_invalid(t *testing.T) {

	// Invalid json string
	json := []byte(`{"name":"ruflin","234"}`)

	result, err := ReadQueryResult(json)
	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestReadSearchResult(t *testing.T) {
	resultsObject := GetValidSearchResults()

	json := []byte(`{
  		"took" : 19,
  		"_shards" : {
    		"total" : 3,
    		"successful" : 2,
    		"failed" : 1
  		},
  		"hits" : {},
  		"aggs" : {}
  	}`)

	results, err := ReadSearchResult(json)

	assert.Nil(t, err)
	assert.Equal(t, resultsObject.Took, results.Took)
	assert.Equal(t, resultsObject.Hits, results.Hits)
	assert.Equal(t, resultsObject.Shards, results.Shards)
	assert.Equal(t, resultsObject.Aggs, results.Aggs)
}

func TestReadSearchResult_empty(t *testing.T) {
	results, err := ReadSearchResult(nil)
	assert.Nil(t, results)
	assert.Nil(t, err)
}

func TestReadSearchResult_invalid(t *testing.T) {

	// Invalid json string
	json := []byte(`{"took":"19","234"}`)

	results, err := ReadSearchResult(json)
	assert.Nil(t, results)
	assert.Error(t, err)
}
