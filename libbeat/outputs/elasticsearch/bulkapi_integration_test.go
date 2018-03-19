// +build integration

package elasticsearch

import (
	"fmt"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
)

func TestBulk(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client := getTestingElasticsearch(t)
	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	params := map[string]string{
		"refresh": "true",
	}
	_, err := client.Bulk(index, "type1", params, body)
	if err != nil {
		t.Fatalf("Bulk() returned error: %s", err)
	}

	params = map[string]string{
		"q": "field1:value1",
	}
	_, result, err := client.SearchURI(index, "type1", params)
	if err != nil {
		t.Fatalf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total)
	}

	_, _, err = client.Delete(index, "", "", nil)
	if err != nil {
		t.Errorf("Delete() returns error: %s", err)
	}
}

func TestEmptyBulk(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client := getTestingElasticsearch(t)
	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	body := make([]interface{}, 0, 10)

	params := map[string]string{
		"refresh": "true",
	}
	resp, err := client.Bulk(index, "type1", params, body)
	if err != nil {
		t.Fatalf("Bulk() returned error: %s", err)
	}
	if resp != nil {
		t.Errorf("Unexpected response: %s", resp)
	}
}

func TestBulkMoreOperations(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("elasticsearch"))

	client := getTestingElasticsearch(t)
	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		{
			"field1": "value1",
		},

		{
			"delete": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "2",
			},
		},

		{
			"create": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "3",
			},
		},
		{
			"field1": "value3",
		},

		{
			"update": map[string]interface{}{
				"_id":    "1",
				"_index": index,
				"_type":  "type1",
			},
		},
		{
			"doc": map[string]interface{}{
				"field2": "value2",
			},
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	params := map[string]string{
		"refresh": "true",
	}
	resp, err := client.Bulk(index, "type1", params, body)
	if err != nil {
		t.Fatalf("Bulk() returned error: %s [%s]", err, resp)
	}

	params = map[string]string{
		"q": "field1:value3",
	}
	_, result, err := client.SearchURI(index, "type1", params)
	if err != nil {
		t.Fatalf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total)
	}

	params = map[string]string{
		"q": "field2:value2",
	}
	_, result, err = client.SearchURI(index, "type1", params)
	if err != nil {
		t.Fatalf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total)
	}

	_, _, err = client.Delete(index, "", "", nil)
	if err != nil {
		t.Errorf("Delete() returns error: %s", err)
	}
}
