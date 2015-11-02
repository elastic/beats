package elasticsearch

import (
	"fmt"
	"os"
	"testing"

	"github.com/elastic/libbeat/logp"
)

func TestIndex(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"elasticsearch"})
	}

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires Elasticsearch")
	}

	client := GetTestingElasticsearch()

	index := fmt.Sprintf("packetbeat-unittest-%d", os.Getpid())

	body := map[string]interface{}{
		"user":      "test",
		"post_date": "2009-11-15T14:12:12",
		"message":   "trying out",
	}
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

	params = map[string]string{
		"q": "user:test",
	}
	_, result, err := client.SearchURI(index, "test", params)
	if err != nil {
		t.Errorf("SearchUri() returns an error: %s", err)
	}
	if result.Hits.Total != 1 {
		t.Errorf("Wrong number of search results: %d", result.Hits.Total)
	}

	_, resp, err = client.Delete(index, "test", "1", nil)
	if err != nil {
		t.Errorf("Delete() returns error: %s", err)
	}
}
