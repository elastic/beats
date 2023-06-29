package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type IndexTemplateResult struct {
	IndexTemplates []IndexTemplateEntry `json:"index_templates"`
}

type IndexTemplateEntry struct {
	Name          string        `json:"name"`
	IndexTemplate IndexTemplate `json:"index_template"`
}

type IndexTemplate struct {
	IndexPatterns []string `json:"index_patterns"`
}

type GetDocsResult struct {
	Hits Hits `json:"hits"`
}

type Hits struct {
	Total Total `json:"total"`
}

type Total struct {
	Value int `json:"value"`
}

var mockbeatConfig = `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
`

func startMockBeat(t *testing.T, msg string, cfg string, args ...string) BeatProc {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", args...)
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs(msg, 60*time.Second)
	return mockbeat
}

func ESDeleteIndexTemplate(t *testing.T, esUrl url.URL, templateName string) {
	// create a new HTTP client
	client := &http.Client{}

	// create a new DELETE request
	u := fmt.Sprintf("%s/_index_template/%s", esUrl.String(), templateName)
	req, _ := http.NewRequest("DELETE", u, nil)

	// send the request
	client.Do(req)
}

func ESGetIndexTemplate(t *testing.T, esUrl url.URL, templateName string) IndexTemplateResult {
	// create a new HTTP client
	client := &http.Client{}

	// create a new DELETE request
	u := fmt.Sprintf("%s/_index_template/%s", esUrl.String(), templateName)
	req, _ := http.NewRequest("GET", u, nil)

	// send the request
	r, _ := client.Do(req)
	require.Equal(t, 200, r.StatusCode, "incorrect status code")

	// read the response
	body, _ := ioutil.ReadAll(r.Body)
	var m IndexTemplateResult
	json.Unmarshal(body, &m)

	return m
}

func ESPostRefresh(t *testing.T, esUrl url.URL) {
	// create a new HTTP client
	client := &http.Client{}

	// create a new DELETE request
	u := fmt.Sprintf("%s/_refresh", esUrl.String())
	req, _ := http.NewRequest("POST", u, nil)

	// send the request
	r, _ := client.Do(req)
	require.Equal(t, 200, r.StatusCode, "incorrect status code")
}

func ESGetDocumentsFromDataStream(t *testing.T, esUrl url.URL, dataStream string) GetDocsResult {
	// create a new HTTP client
	client := &http.Client{}

	// create a new DELETE request
	u := fmt.Sprintf("%s/%s/_search", esUrl.String(), dataStream)
	req, _ := http.NewRequest("GET", u, nil)

	// send the request
	r, _ := client.Do(req)
	require.Equal(t, 200, r.StatusCode, "incorrect status code")

	// read the response
	body, _ := ioutil.ReadAll(r.Body)
	var m GetDocsResult
	json.Unmarshal(body, &m)

	return m
}
