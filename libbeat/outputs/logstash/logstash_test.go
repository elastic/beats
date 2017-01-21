// Need for unit and integration tests
package logstash

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-lumber/server/v2"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport/transptest"
)

const (
	logstashDefaultHost     = "localhost"
	logstashTestDefaultPort = "5044"
)

var testOptions = outputs.Options{}

func strDefault(a, defaults string) string {
	if len(a) == 0 {
		return defaults
	}
	return a
}

func getenv(name, defaultValue string) string {
	return strDefault(os.Getenv(name), defaultValue)
}

func getLogstashHost() string {
	return fmt.Sprintf("%v:%v",
		getenv("LS_HOST", logstashDefaultHost),
		getenv("LS_TCP_PORT", logstashTestDefaultPort),
	)
}

func testEvent() outputs.Data {
	return outputs.Data{Event: common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "log",
		"extra":      10,
		"message":    "message",
	}}
}

func testLogstashIndex(test string) string {
	return fmt.Sprintf("beat-logstash-int-%v-%d", test, os.Getpid())
}

func newTestLumberjackOutput(
	t *testing.T,
	test string,
	config map[string]interface{},
) outputs.BulkOutputer {
	if config == nil {
		config = map[string]interface{}{
			"hosts": []string{getLogstashHost()},
			"index": testLogstashIndex(test),
		}
	}

	plugin := outputs.FindOutputPlugin("logstash")
	if plugin == nil {
		t.Fatalf("No logstash output plugin found")
	}

	cfg, _ := common.NewConfigFrom(config)
	output, err := plugin("", cfg, 0)
	if err != nil {
		t.Fatalf("init logstash output plugin failed: %v", err)
	}

	return output.(outputs.BulkOutputer)
}

func testOutputerFactory(
	t *testing.T,
	test string,
	config map[string]interface{},
) func() outputs.BulkOutputer {
	return func() outputs.BulkOutputer {
		return newTestLumberjackOutput(t, test, config)
	}
}

func TestLogstashTCP(t *testing.T) {
	timeout := 2 * time.Second
	server := transptest.NewMockServerTCP(t, timeout, "", nil)

	// create lumberjack output client
	config := map[string]interface{}{
		"hosts":   []string{server.Addr()},
		"index":   testLogstashIndex("logstash-conn-tcp"),
		"timeout": 2,
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func TestLogstashTLS(t *testing.T) {
	certName := "ca_test"
	ip := net.IP{127, 0, 0, 1}

	timeout := 2 * time.Second
	transptest.GenCertsForIPIfMIssing(t, ip, certName)
	server := transptest.NewMockServerTLS(t, timeout, certName, nil)

	config := map[string]interface{}{
		"hosts":                       []string{server.Addr()},
		"index":                       testLogstashIndex("logstash-conn-tls"),
		"timeout":                     2,
		"ssl.certificate_authorities": []string{certName + ".pem"},
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func TestLogstashInvalidTLSInsecure(t *testing.T) {
	certName := "ca_invalid_test"
	ip := net.IP{1, 2, 3, 4}

	timeout := 2 * time.Second
	transptest.GenCertsForIPIfMIssing(t, ip, certName)
	server := transptest.NewMockServerTLS(t, timeout, certName, nil)

	config := map[string]interface{}{
		"hosts":                       []string{server.Addr()},
		"index":                       testLogstashIndex("logstash-conn-tls-invalid"),
		"timeout":                     2,
		"max_retries":                 1,
		"ssl.verification_mode":       "none",
		"ssl.certificate_authorities": []string{certName + ".pem"},
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func testConnectionType(
	t *testing.T,
	mock *transptest.MockServer,
	makeOutputer func() outputs.BulkOutputer,
) {
	t.Log("testConnectionType")
	server, _ := v2.NewWithListener(mock.Listener)

	// worker loop
	go func() {
		t.Log("start worker loop")
		defer t.Log("stop worker loop")

		t.Log("make outputter")
		output := makeOutputer()
		t.Logf("new outputter: %v", output)

		signal := op.NewSignalChannel()
		t.Log("publish event")
		output.PublishEvent(signal, testOptions, testEvent())

		t.Log("wait signal")
		assert.True(t, signal.Wait() == op.SignalCompleted)

		server.Close()
	}()

	for batch := range server.ReceiveChan() {
		batch.ACK()

		events := batch.Events
		assert.Equal(t, 1, len(events))
		msg := events[0].(map[string]interface{})
		assert.Equal(t, 10.0, msg["extra"])
		assert.Equal(t, "message", msg["message"])
	}
}
