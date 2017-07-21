package node

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchEventContents(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/testdata/")

	response, err := ioutil.ReadFile(absPath + "/node_sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "rabbitmq",
		"metricsets": []string{"node"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	disk := event["disk"].(common.MapStr)
	free := disk["free"].(common.MapStr)
	assert.EqualValues(t, 313352192, free["bytes"])

	limit := free["limit"].(common.MapStr)
	assert.EqualValues(t, 50000000, limit["bytes"])

	fd := event["fd"].(common.MapStr)
	assert.EqualValues(t, 65536, fd["total"])
	assert.EqualValues(t, 54, fd["used"])

	gc := event["gc"].(common.MapStr)
	num := gc["num"].(common.MapStr)
	assert.EqualValues(t, 3184, num["count"])
	reclaimed := gc["reclaimed"].(common.MapStr)
	assert.EqualValues(t, 270119840, reclaimed["bytes"])

	io := event["io"].(common.MapStr)
	file_handle := io["file_handle"].(common.MapStr)
	open_attempt := file_handle["open_attempt"].(common.MapStr)
	avg := open_attempt["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 10, open_attempt["count"])

	read := io["read"].(common.MapStr)
	avg = read["avg"].(common.MapStr)
	assert.EqualValues(t, 33, avg["ms"])
	assert.EqualValues(t, 1, read["bytes"])
	assert.EqualValues(t, 1, read["count"])

	reopen := io["reopen"].(common.MapStr)
	assert.EqualValues(t, 1, reopen["count"])

	seek := io["seek"].(common.MapStr)
	avg = seek["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 0, seek["count"])

	sync := io["sync"].(common.MapStr)
	avg = sync["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 0, sync["count"])

	write := io["write"].(common.MapStr)
	avg = write["avg"].(common.MapStr)
	assert.EqualValues(t, 0, avg["ms"])
	assert.EqualValues(t, 0, write["bytes"])
	assert.EqualValues(t, 0, write["count"])

	mem := event["mem"].(common.MapStr)
	limit = mem["limit"].(common.MapStr)
	assert.EqualValues(t, 413047193, limit["bytes"])
	used := mem["used"].(common.MapStr)
	assert.EqualValues(t, 57260080, used["bytes"])

	mnesia := event["mnesia"].(common.MapStr)
	disk = mnesia["disk"].(common.MapStr)
	tx := disk["tx"].(common.MapStr)
	assert.EqualValues(t, 0, tx["count"])
	ram := mnesia["ram"].(common.MapStr)
	tx = ram["tx"].(common.MapStr)
	assert.EqualValues(t, 11, tx["count"])

	msg := event["msg"].(common.MapStr)
	store_read := msg["store_read"].(common.MapStr)
	assert.EqualValues(t, 0, store_read["count"])
	store_write := msg["store_write"].(common.MapStr)
	assert.EqualValues(t, 0, store_write["count"])

	assert.EqualValues(t, "rabbit@prcdsrvv1682", event["name"])

	proc := event["proc"].(common.MapStr)
	assert.EqualValues(t, 1048576, proc["total"])
	assert.EqualValues(t, 322, proc["used"])

	assert.EqualValues(t, 2, event["processors"])

	queue := event["queue"].(common.MapStr)
	index := queue["index"].(common.MapStr)
	journal_write := index["journal_write"].(common.MapStr)
	assert.EqualValues(t, 0, journal_write["count"])
	read = index["read"].(common.MapStr)
	assert.EqualValues(t, 0, read["count"])
	write = index["write"].(common.MapStr)
	assert.EqualValues(t, 0, write["count"])

	run := event["run"].(common.MapStr)
	assert.EqualValues(t, 0, run["queue"])

	socket := event["socket"].(common.MapStr)
	assert.EqualValues(t, 58890, socket["total"])
	assert.EqualValues(t, 0, socket["used"])

	assert.EqualValues(t, "disc", event["type"])

	assert.EqualValues(t, 37139, event["uptime"])
}
