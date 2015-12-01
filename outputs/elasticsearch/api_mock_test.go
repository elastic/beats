package elasticsearch

import (
	"fmt"
	"os"
	"time"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/libbeat/logp"
)

func ElasticsearchMock(code int, body []byte) *httptest.Server {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respCode := code
		if r.Method == "HEAD" { // send ok on ping
			respCode = 200
		}

		w.WriteHeader(respCode)
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

	client := NewClient(server.URL, "", nil, nil, "", "")

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

	client := NewClient(server.URL, "", nil, nil, "", "")
	err := client.Connect(1 * time.Second)
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

	client := NewClient(server.URL, "", nil, nil, "", "")

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
