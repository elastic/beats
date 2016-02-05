package logstash

import (
	"net"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/stretchr/testify/assert"
)

const (
	driverCmdQuit = iota
	driverCmdPublish
)

type testClientDriver interface {
	Stop()
	Publish(events []common.MapStr)
	Returns() []testClientReturn
}

type clientFactory func(TransportClient) testClientDriver

type testClientReturn struct {
	n   int
	err error
}

type testDriverCommand struct {
	code   int
	events []common.MapStr
}

func newLumberjackTestClient(conn TransportClient) *client {
	c, err := newLumberjackClient(conn, 3, testMaxWindowSize, 250*time.Millisecond)
	if err != nil {
		panic(err)
	}
	return c
}

func enableLogging(selectors []string) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, selectors)
	}
}

const testMaxWindowSize = 64

func testSendZero(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})

	server := newMockServerTCP(t, 1*time.Second, "")
	defer server.Close()

	sock, transp, err := server.connectPair(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}

	client := factory(transp)
	defer sock.Close()
	defer transp.Close()

	client.Publish(make([]common.MapStr, 0))

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
	server := newMockServerTCP(t, 1*time.Second, "")

	sock, transp, err := server.connectPair(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}
	client := factory(transp)
	conn := &mockConn{sock, streambuf.New(nil)}
	defer transp.Close()
	defer sock.Close()

	event := common.MapStr{"name": "me", "line": 10}
	client.Publish([]common.MapStr{event})

	// receive window message
	err = sock.SetReadDeadline(time.Now().Add(1 * time.Second))
	win, err := conn.recvMessage()
	assert.Nil(t, err)

	// receive data message
	msg, err := conn.recvMessage()
	assert.Nil(t, err)

	// send ack
	conn.sendACK(1)

	client.Stop()

	// validate
	assert.NotNil(t, win)
	assert.NotNil(t, msg)
	assert.Equal(t, 1, len(msg.events))
	msg = msg.events[0]
	assert.Equal(t, "me", msg.doc["name"])
	assert.Equal(t, 10.0, msg.doc["line"])
}

func testStructuredEvent(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})
	server := newMockServerTCP(t, 1*time.Second, "")

	sock, transp, err := server.connectPair(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}
	client := factory(transp)
	conn := &mockConn{sock, streambuf.New(nil)}
	defer transp.Close()
	defer sock.Close()

	event := common.MapStr{
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
	}
	client.Publish([]common.MapStr{event})

	win, err := conn.recvMessage()
	assert.Nil(t, err)

	msg, err := conn.recvMessage()
	assert.Nil(t, err)

	conn.sendACK(1)
	defer client.Stop()

	// validate
	assert.NotNil(t, win)
	assert.NotNil(t, msg)
	assert.Equal(t, 1, len(msg.events))
	msg = msg.events[0]
	assert.Equal(t, "test", msg.doc["name"])
	assert.Equal(t, 1.0, msg.doc.get("struct.field1"))
	assert.Equal(t, true, msg.doc.get("struct.field2"))
	assert.Equal(t, 2.0, msg.doc.get("struct.field5.sub1"))
}

func testCloseAfterWindowSize(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})
	server := newMockServerTCP(t, 100*time.Millisecond, "")

	sock, transp, err := server.connectPair(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}
	client := factory(transp)
	conn := &mockConn{sock, streambuf.New(nil)}
	defer transp.Close()
	defer sock.Close()
	defer client.Stop()

	client.Publish([]common.MapStr{common.MapStr{
		"message": "hello world",
	}})

	_, err = conn.recvMessage()
	if err != nil {
		t.Fatalf("failed to read window size message: %v", err)
	}

}

func testFailAfterMaxTimeouts(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})
	server := newMockServerTCP(t, 100*time.Millisecond, "")
	sock, transp, err := server.connectPair(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}

	client := factory(transp)
	conn := &mockConn{sock, streambuf.New(nil)}
	defer sock.Close()
	defer transp.Close()
	defer client.Stop()

	// publish event
	event := common.MapStr{"name": "me", "line": 10}
	client.Publish([]common.MapStr{event})

	// force connection to time out
	for i := 0; i < maxAllowedTimeoutErr; i++ {
		// read window
		msg, err := conn.recvMessage()
		if err != nil {
			t.Fatalf("Failed receiving window size: %v", err)
		}
		if msg.code != 'W' {
			t.Fatalf("expected window size message")
		}

		// read message
		msg, err = conn.recvMessage()
		if err != nil {
			t.Fatalf("Failed receiving data message: %v", err)
		}
		if msg.code != 'C' {
			t.Fatalf("expected data message")
		}

		// do not respond -> enforce timeout
	}

	// check connection being closed
	sock.SetDeadline(time.Now().Add(100 * time.Millisecond))
	msg, err := conn.recvMessage()
	if msg != nil {
		t.Fatalf("Received message on connection expected to be closed")
	}
	if nerr, ok := err.(net.Error); err != io.EOF && !(ok && nerr.Timeout()) {
		t.Fatalf("Unexpected error type: %v", err)
	}

	client.Stop()

	returns := client.Returns()
	if len(returns) != 1 {
		t.Fatalf("PublishEvents did not return")
	}

	assert.Equal(t, 0, returns[0].n)
	assert.NotNil(t, returns[0].err)
}
