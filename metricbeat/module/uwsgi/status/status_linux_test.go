package status

import (
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetchDataUnixSock(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "mb_uwsgi_status")
	assert.NoError(t, err)
	fname := tmpfile.Name()
	os.Remove(fname)

	listener, err := net.Listen("unix", fname)
	assert.NoError(t, err)
	defer os.Remove(fname)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		conn, err := listener.Accept()
		assert.NoError(t, err)

		data := testData(t)
		conn.Write(data)
		conn.Close()
		wg.Done()
	}()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"unix://" + listener.Addr().String()},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	assert.NoError(t, err)

	assertTestData(t, events)
	wg.Wait()
}
