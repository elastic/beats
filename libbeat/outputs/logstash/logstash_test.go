// Need for unit and integration tests

package logstash

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport/transptest"
)

const (
	logstashDefaultHost     = "localhost"
	logstashTestDefaultPort = "5044"
)

type mockLSServer struct {
	*transptest.MockServer
}

var testOptions = outputs.Options{}

func newMockTLSServer(t *testing.T, to time.Duration, cert string) *mockLSServer {
	return &mockLSServer{transptest.NewMockServerTLS(t, to, cert, nil)}
}

func newMockTCPServer(t *testing.T, to time.Duration) *mockLSServer {
	return &mockLSServer{transptest.NewMockServerTCP(t, to, "", nil)}
}

func (m *mockLSServer) readMessage(buf *streambuf.Buffer, client net.Conn) *message {
	if m.Err != nil {
		return nil
	}

	m.ClientDeadline(client, m.Timeout)
	if m.Err != nil {
		return nil
	}

	msg, err := sockReadMessage(buf, client)
	m.Err = err
	return msg
}

func (m *mockLSServer) sendACK(client net.Conn, seq uint32) {
	if m.Err == nil {
		m.Err = sockSendACK(client, seq)
	}
}

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

func testEvent() common.MapStr {
	return common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "log",
		"extra":      10,
		"message":    "message",
	}
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
	output, err := plugin(cfg, 0)
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

func sockReadMessage(buf *streambuf.Buffer, in io.Reader) (*message, error) {
	for {
		// try parse message from buffered data
		msg, err := readMessage(buf)
		if msg != nil || (err != nil && err != streambuf.ErrNoMoreBytes) {
			return msg, err
		}

		// read next bytes from socket if incomplete message in buffer
		buffer := make([]byte, 1024)
		n, err := in.Read(buffer)
		if err != nil {
			return nil, err
		}

		buf.Write(buffer[:n])
	}
}

func sockSendACK(out io.Writer, seq uint32) error {
	buf := streambuf.New(nil)
	buf.WriteByte('2')
	buf.WriteByte('A')
	buf.WriteNetUint32(seq)
	_, err := out.Write(buf.Bytes())
	return err
}

func TestLogstashTCP(t *testing.T) {
	timeout := 2 * time.Second
	server := newMockTCPServer(t, timeout)

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
	server := newMockTLSServer(t, timeout, certName)

	config := map[string]interface{}{
		"hosts":                       []string{server.Addr()},
		"index":                       testLogstashIndex("logstash-conn-tls"),
		"timeout":                     2,
		"tls.certificate_authorities": []string{certName + ".pem"},
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func TestLogstashInvalidTLSInsecure(t *testing.T) {
	certName := "ca_invalid_test"
	ip := net.IP{1, 2, 3, 4}

	timeout := 2 * time.Second
	transptest.GenCertsForIPIfMIssing(t, ip, certName)
	server := newMockTLSServer(t, timeout, certName)

	config := map[string]interface{}{
		"hosts":                       []string{server.Addr()},
		"index":                       testLogstashIndex("logstash-conn-tls-invalid"),
		"timeout":                     2,
		"max_retries":                 1,
		"tls.insecure":                true,
		"tls.certificate_authorities": []string{certName + ".pem"},
	}
	testConnectionType(t, server, testOutputerFactory(t, "", config))
}

func testConnectionType(
	t *testing.T,
	server *mockLSServer,
	makeOutputer func() outputs.BulkOutputer,
) {
	var result struct {
		err       error
		win, data *message
		signal    bool
	}

	var wg struct {
		ready  sync.WaitGroup
		finish sync.WaitGroup
	}

	t.Log("testConnectionType")

	wg.ready.Add(1)  // server signaling readiness to client worker
	wg.finish.Add(2) // server/client signaling test end

	// server loop
	go func() {
		defer wg.finish.Done()
		wg.ready.Done()

		t.Log("start server loop")
		defer t.Log("stop server loop")

		client := server.Accept()
		server.Handshake(client)

		buf := streambuf.New(nil)
		result.win = server.readMessage(buf, client)
		result.data = server.readMessage(buf, client)
		server.sendACK(client, 1)
		result.err = server.Err
	}()

	// worker loop
	go func() {
		defer wg.finish.Done()
		wg.ready.Wait()

		t.Log("start worker loop")
		defer t.Log("stop worker loop")

		t.Log("make outputter")
		output := makeOutputer()
		t.Logf("new outputter: %v", output)

		signal := op.NewSignalChannel()
		t.Log("publish event")
		output.PublishEvent(signal, testOptions, testEvent())

		t.Log("wait signal")
		result.signal = signal.Wait() == op.SignalCompleted
	}()

	// wait shutdown
	wg.finish.Wait()
	server.Close()

	// validate output
	assert.Nil(t, result.err)
	assert.True(t, result.signal)

	data := result.data
	assert.NotNil(t, result.win)
	assert.NotNil(t, result.data)
	if data != nil {
		assert.Equal(t, 1, len(data.events))
		data = data.events[0]
		assert.Equal(t, 10.0, data.doc["extra"])
		assert.Equal(t, "message", data.doc["message"])
	}

}

func TestLogstashInvalidTLS(t *testing.T) {
	certName := "ca_invalid_test"
	ip := net.IP{1, 2, 3, 4}

	timeout := 2 * time.Second
	transptest.GenCertsForIPIfMIssing(t, ip, certName)
	server := newMockTLSServer(t, timeout, certName)

	config := map[string]interface{}{
		"hosts":                       []string{server.Addr()},
		"index":                       testLogstashIndex("logstash-tls-invalid"),
		"timeout":                     1,
		"max_retries":                 0,
		"tls.certificate_authorities": []string{certName + ".pem"},
	}

	var result struct {
		err           error
		handshakeFail bool
		signal        bool
	}

	var wg struct {
		ready  sync.WaitGroup
		finish sync.WaitGroup
	}

	wg.ready.Add(1)  // server signaling readiness to client worker
	wg.finish.Add(2) // server/client signaling test end

	// server loop
	go func() {
		defer wg.finish.Done()
		wg.ready.Done()

		client := server.Accept()
		if server.Err != nil {
			t.Fatalf("server error: %v", server.Err)
		}

		server.Handshake(client)
		result.handshakeFail = server.Err != nil
	}()

	// client loop
	go func() {
		defer wg.finish.Done()
		wg.ready.Wait()

		output := newTestLumberjackOutput(t, "", config)

		signal := op.NewSignalChannel()
		output.PublishEvent(signal, testOptions, testEvent())
		result.signal = signal.Wait() == op.SignalCompleted
	}()

	// wait shutdown
	wg.finish.Wait()
	server.Close()

	// validate output
	assert.True(t, result.handshakeFail)
	assert.False(t, result.signal)
}
