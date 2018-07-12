package tcp

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/valschema"
	"github.com/elastic/beats/libbeat/common"
)

func TestUpEndpoint(t *testing.T) {
	server := httptest.NewServer(valschema.HelloWorldHandler)
	defer server.Close()

	port, err := valschema.ServerPort(server)
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

	valschema.Validate(t, valschema.MonitorChecks(fmt.Sprintf("tcp-tcp@localhost:%d", port), "127.0.0.1", "tcp", "up"), event.Fields)
	valschema.Validate(t, valschema.TcpChecks(port), event.Fields)
}
