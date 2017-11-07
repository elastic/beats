// +build !integration

package elasticsearch

import (
	"fmt"
	"os"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
)

func ElasticsearchMock(code int, body []byte) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" { // send ok and a minimal JSON on ping
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"version":{"number":"5.0.0"}}`))
			return
		}

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

	client := newTestClient(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, resp, err := client.Index(index, "test", "1", params, body)
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

	client := newTestClient(server.URL)
	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err = client.Index(index, "test", "1", params, body)

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

	client := newTestClient(server.URL)

	params := map[string]string{
		"refresh": "true",
	}
	_, _, err := client.Index(index, "test", "1", params, body)
	if err == nil {
		t.Errorf("Index() should return error.")
	}

	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Errorf("Should return <503 Service Unavailable> instead of %v", err)
	}
}
