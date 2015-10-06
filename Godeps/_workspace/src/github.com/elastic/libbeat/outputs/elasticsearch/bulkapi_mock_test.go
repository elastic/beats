package elasticsearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/elastic/libbeat/logp"
)

func TestOneHostSuccessResp_Bulk(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	expectedResp, _ := json.Marshal(QueryResult{Ok: true, Index: index, Type: "type1", ID: "1", Version: 1, Created: true})

	ops := []map[string]interface{}{
		map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		map[string]interface{}{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	server := ElasticsearchMock(200, expectedResp)

	es := NewElasticsearch([]string{server.URL}, nil, "", "")

	params := map[string]string{
		"refresh": "true",
	}
	resp, err := es.Bulk(index, "type1", params, body)
	if err != nil {
		t.Errorf("Bulk() returns error: %s", err)
	}
	if !resp.Created {
		t.Errorf("Bulk() fails: %s", resp)
	}
}

func TestOneHost500Resp_Bulk(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		map[string]interface{}{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	server := ElasticsearchMock(http.StatusInternalServerError, []byte("Something wrong happened"))

	es := NewElasticsearch([]string{server.URL}, nil, "", "")

	params := map[string]string{
		"refresh": "true",
	}
	_, err := es.Bulk(index, "type1", params, body)
	if err == nil {
		t.Errorf("Bulk() should return error.")
	}

	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Errorf("Should return <500 Internal Server Error> instead of %v", err)
	}
}

func TestOneHost503Resp_Bulk(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	ops := []map[string]interface{}{
		map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		map[string]interface{}{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	server := ElasticsearchMock(503, []byte("Something wrong happened"))

	es := NewElasticsearch([]string{server.URL}, nil, "", "")

	params := map[string]string{
		"refresh": "true",
	}
	_, err := es.Bulk(index, "type1", params, body)
	if err == nil {
		t.Errorf("Bulk() should return error.")
	}

	if !strings.Contains(err.Error(), "retries. Errors") {
		t.Errorf("Should return <Request fails after 3 retries. Errors: > instead of %v", err)
	}
}

func TestMultipleHost_Bulk(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	expectedResp, _ := json.Marshal(QueryResult{Ok: true, Index: index, Type: "type1", ID: "1", Version: 1, Created: true})

	ops := []map[string]interface{}{
		map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
				"_type":  "type1",
				"_id":    "1",
			},
		},
		map[string]interface{}{
			"field1": "value1",
		},
	}

	body := make([]interface{}, 0, 10)
	for _, op := range ops {
		body = append(body, op)
	}

	server1 := ElasticsearchMock(503, []byte("Somehting went wrong"))
	server2 := ElasticsearchMock(200, expectedResp)

	es := NewElasticsearch([]string{server1.URL, server2.URL}, nil, "", "")

	params := map[string]string{
		"refresh": "true",
	}
	resp, err := es.Bulk(index, "type1", params, body)
	if err != nil {
		t.Errorf("Bulk() returns error: %s", err)
	}
	if !resp.Created {
		t.Errorf("Bulk() fails: %s", resp)
	}
}
