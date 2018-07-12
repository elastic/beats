package tcp

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/skima"
	"github.com/elastic/beats/heartbeat/testutil"
	"github.com/elastic/beats/libbeat/common"
)

func TestUpEndpoint(t *testing.T) {
	server := httptest.NewServer(testutil.HelloWorldHandler)
	defer server.Close()

	port, err := testutil.ServerPort(server)
	if err != nil {
		t.FailNow()
	}

	config := common.NewConfig()
	config.SetString("hosts", 0, "localhost")
	config.SetInt("ports", 0, int64(port))

	jobs, err := create(monitors.Info{}, config)
	if err != nil {
		t.FailNow()
	}
	job := jobs[0]

	event, _, err := job.Run()
	if err != nil {
		t.FailNow()
	}

	skima.Strict(skima.Compose(
		testutil.MonitorChecks(
			fmt.Sprintf("tcp-tcp@localhost:%d", port),
			"127.0.0.1",
			"tcp",
			"up",
		),
		testutil.TcpChecks(port),
	))(t, event.Fields)
}
