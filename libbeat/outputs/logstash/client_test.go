// +build !integration

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
	c, err := newLumberjackClient(conn, 3, testMaxWindowSize, 100*time.Millisecond)
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

	server := newMockServerTCP(t, 1*time.Second, "", nil)
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
	server := newMockServerTCP(t, 1*time.Second, "", nil)

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
	server := newMockServerTCP(t, 1*time.Second, "", nil)

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
	server := newMockServerTCP(t, 100*time.Millisecond, "", nil)

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

func testMultiFailMaxTimeouts(t *testing.T, factory clientFactory) {
	enableLogging([]string{"*"})

	server := newMockServerTCP(t, 100*time.Millisecond, "", nil)
	transp, err := server.transp()
	if err != nil {
		t.Fatalf("Failed to connect server and client: %v", err)
	}

	N := 8
	client := factory(transp)
	defer transp.Close()
	defer client.Stop()

	event := common.MapStr{"name": "me", "line": 10}

	for i := 0; i < N; i++ {
		await := server.await()
		err = transp.Connect(100 * time.Millisecond)
		if err != nil {
			t.Fatalf("Transport client Failed to connect: %v", err)
		}
		sock := <-await
		conn := &mockConn{sock, streambuf.New(nil)}

		// close socket only once test has finished
		// so no EOF error can be generated
		defer sock.Close()

		// publish event. With client returning on timeout, we have to send
		// messages again
		client.Publish([]common.MapStr{event})

		// read window
		msg, err := conn.recvMessage()
		if err != nil {
			t.Errorf("Failed receiving window size: %v", err)
			break
		}
		if msg.code != 'W' {
			t.Errorf("expected window size message")
			break
		}

		// read message
		msg, err = conn.recvMessage()
		if err != nil {
			t.Errorf("Failed receiving data message: %v", err)
			break
		}
		if msg.code != 'C' {
			t.Errorf("expected data message")
			break
		}
		// do not respond -> enforce timeout

		// check connection being closed,
		// timeout required in case of sender not closing the connection
		// correctly
		sock.SetDeadline(time.Now().Add(30 * time.Second))
		msg, err = conn.recvMessage()
		if msg != nil {
			t.Errorf("Received message on connection expected to be closed")
			break
		}
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			t.Errorf("Unexpected timeout error (client did not close connection in time?): %v", err)
			break
		}
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
