package elasticsearch

import (
	"fmt"
	"os"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/libbeat/logp"
)

func ElasticsearchMock(code int, body []byte) *httptest.Server {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		if body != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
		}
	}))

	return server
}

func TestOneHostSuccessResp(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}
	expectedResp, _ := json.Marshal(QueryResult{Ok: true, Index: index, Type: "test", ID: "1", Version: 1, Created: true})

	server := ElasticsearchMock(200, expectedResp)

	es := NewElasticsearch([]string{server.URL}, nil, "", "")

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
}

func TestOneHost500Resp(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}

	server := ElasticsearchMock(http.StatusInternalServerError, []byte("Something wrong happened"))

	es := NewElasticsearch([]string{server.URL}, nil, "", "")

	params := map[string]string{
		"refresh": "true",
	}
	_, err := es.Index(index, "test", "1", params, body)
	if err == nil {
		t.Errorf("Index() should return error.")
	}

	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Errorf("Should return <500 Internal Server Error> instead of %v", err)
	}
}

func TestOneHost503Resp(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}

	server := ElasticsearchMock(503, []byte("Something wrong happened"))

	es := NewElasticsearch([]string{server.URL}, nil, "", "")

	params := map[string]string{
		"refresh": "true",
	}
	_, err := es.Index(index, "test", "1", params, body)
	if err == nil {
		t.Errorf("Index() should return error.")
	}

	if !strings.Contains(err.Error(), "retries. Errors") {
		t.Errorf("Should return <Request fails after 3 retries. Errors: > instead of %v", err)
	}
}

func TestMultipleHosts(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}
	expectedResp, _ := json.Marshal(QueryResult{Ok: true, Index: index, Type: "test", ID: "1", Version: 1, Created: true})

	server1 := ElasticsearchMock(503, []byte("Something went wrong"))
	server2 := ElasticsearchMock(200, expectedResp)

	logp.Debug("elasticsearch", "%s, %s", server1.URL, server2.URL)
	es := NewElasticsearch([]string{server1.URL, server2.URL}, nil, "", "")

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

}

func TestMultipleFailingHosts(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())
	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}
	server1 := ElasticsearchMock(503, []byte("Something went wrong"))
	server2 := ElasticsearchMock(500, []byte("Something went wrong"))

	logp.Debug("elasticsearch", "%s, %s", server1.URL, server2.URL)
	es := NewElasticsearch([]string{server1.URL, server2.URL}, nil, "", "")

	params := map[string]string{
		"refresh": "true",
	}
	_, err := es.Index(index, "test", "1", params, body)
	if err == nil {
		t.Errorf("Index() should return error.")
	}

	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Errorf("Should return <500 Internal Server Error> instead of %v", err)
	}

}
