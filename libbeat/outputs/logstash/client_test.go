// +build !integration

package logstash

import (
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-lumber/server/v2"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/outputs/transport/transptest"

	"github.com/stretchr/testify/assert"
)

const (
	driverCmdQuit = iota
	driverCmdPublish
	driverCmdConnect
	driverCmdClose
)

type testClientDriver interface {
	Connect()
	Close()
	Stop()
	Publish([]outputs.Data)
	Returns() []testClientReturn
}

type clientFactory func(*transport.Client) testClientDriver

type testClientReturn struct {
	n   int
	err error
}

type testDriverCommand struct {
	code int
	data []outputs.Data
}

func newLumberjackTestClient(conn *transport.Client) *client {
	c, err := newLumberjackClient(conn, 3,
		testMaxWindowSize, 100*time.Millisecond, "test")
	if err != nil {
		panic(err)
	}
	return c
}

const testMaxWindowSize = 64

func testSendZero(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})

	server := transptest.NewMockServerTCP(t, 1*time.Second, "", nil)
	defer server.Close()

	sock, transp, err := server.ConnectPair()
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}

	client := factory(transp)
	defer sock.Close()
	defer transp.Close()

	client.Publish(make([]outputs.Data, 0))

	client.Stop()
	returns := client.Returns()

	assert.Equal(t, 1, len(returns))
	if len(returns) == 1 {
		assert.Equal(t, 0, returns[0].n)
		assert.Nil(t, returns[0].err)
	}
}

func testSimpleEvent(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})
	mock := transptest.NewMockServerTCP(t, 1*time.Second, "", nil)
	server, _ := v2.NewWithListener(mock.Listener)
	defer server.Close()

	transp, err := mock.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	client := factory(transp)
	defer transp.Close()
	defer client.Stop()

	event := outputs.Data{Event: common.MapStr{"type": "test", "name": "me", "line": 10}}
	go client.Publish([]outputs.Data{event})

	// try to receive event from server
	batch := server.Receive()
	batch.ACK()

	// validate
	events := batch.Events
	assert.Equal(t, 1, len(events))
	msg := events[0].(map[string]interface{})
	assert.Equal(t, "me", msg["name"])
	assert.Equal(t, 10.0, msg["line"])
}

func testStructuredEvent(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})
	mock := transptest.NewMockServerTCP(t, 1*time.Second, "", nil)
	server, _ := v2.NewWithListener(mock.Listener)
	defer server.Close()

	transp, err := mock.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	client := factory(transp)
	defer transp.Close()
	defer client.Stop()

	event := outputs.Data{Event: common.MapStr{
		"type": "test",
		"name": "test",
		"struct": common.MapStr{
			"field1": 1,
			"field2": true,
			"field3": []int{1, 2, 3},
			"field4": []interface{}{
				1,
				"test",
				common.MapStr{
					"sub": "field",
				},
			},
			"field5": common.MapStr{
				"sub1": 2,
			},
		},
	}}
	go client.Publish([]outputs.Data{event})
	defer client.Stop()

	// try to receive event from server
	batch := server.Receive()
	batch.ACK()

	events := batch.Events
	assert.Equal(t, 1, len(events))
	msg := events[0]
	assert.Equal(t, "test", eventGet(msg, "name"))
	assert.Equal(t, 1.0, eventGet(msg, "struct.field1"))
	assert.Equal(t, true, eventGet(msg, "struct.field2"))
	assert.Equal(t, 2.0, eventGet(msg, "struct.field5.sub1"))
}

func testMultiFailMaxTimeouts(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})

	mock := transptest.NewMockServerTCP(t, 100*time.Millisecond, "", nil)
	server, _ := v2.NewWithListener(mock.Listener)
	defer server.Close()

	transp, err := mock.Transp()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	client := factory(transp)
	defer transp.Close()
	defer client.Stop()

	N := 8
	event := outputs.Data{Event: common.MapStr{"type": "test", "name": "me", "line": 10}}

	for i := 0; i < N; i++ {
		// reconnect client
		client.Close()
		client.Connect()

		// publish event. With client returning on timeout, we have to send
		// messages again
		go client.Publish([]outputs.Data{event})

		// read batch + never ACK in order to enforce timeout
		server.Receive()

		// wait for max connection timeout ensuring ACK receive fails
		time.Sleep(100 * time.Millisecond)
	}

	client.Stop()
	returns := client.Returns()
	if len(returns) != N {
		t.Fatalf("PublishEvents did not return")
	}

	for _, ret := range returns {
		assert.Equal(t, 0, ret.n)
		assert.NotNil(t, ret.err)
	}
}

func eventGet(event interface{}, path string) interface{} {
	doc := event.(map[string]interface{})
	elems := strings.Split(path, ".")
	for i := 0; i < len(elems)-1; i++ {
		doc = doc[elems[i]].(map[string]interface{})
	}
	return doc[elems[len(elems)-1]]
}
