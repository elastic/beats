// +build integration

package stats

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	err := loadData()
	if err != nil {
		t.Fatal("write", err)
	}

	f := mbtest.NewEventFetcher(t, elasticsearch.GetConfig("stats"))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.NotNil(t, event)
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func TestData(t *testing.T) {
	err := loadData()
	if err != nil {
		t.Fatal("write", err)
	}

	f := mbtest.NewEventFetcher(t, elasticsearch.GetConfig("stats"))
	err = mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func loadData() error {
	client := &http.Client{}
	url := fmt.Sprintf("http://%s:%s/tests/stats", elasticsearch.GetEnvHost(), elasticsearch.GetEnvPort())

	request, err := http.NewRequest("POST", url, strings.NewReader(`{"hello":"world"}`))
	request.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}
	res, err := client.Do(request)
	defer res.Body.Close()
	return err
}
