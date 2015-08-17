package elasticsearch

import (
	"fmt"
	"os"
	"testing"

	"github.com/elastic/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

func GetTestingElasticsearch() *Elasticsearch {
	var es_url string

	// read the Elasticsearch port from the ES_PORT env variable
	port := os.Getenv("ES_PORT")
	if len(port) > 0 {
		es_url = "http://localhost:" + port
	} else {
		// empty variable
		es_url = "http://localhost:9200"
	}

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

func GetInvalidQueryResult() {

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
