package logstash

// TODO:
//  - test window increase for multiple sends
//  - test window decrease on timeout
//  - test with connection timeout

import (
	"compress/zlib"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/common/streambuf"
	"github.com/elastic/libbeat/outputs/mode"

	"github.com/stretchr/testify/assert"
)

type mockAddr string

const (
	driverCmdQuit = iota
	driverCmdPublish
)

type testClientReturn struct {
	n   int
	err error
}

type testDriverCommand struct {
	code   int
	events []common.MapStr
}

type testClientDriver struct {
	client  mode.ProtocolClient
	ch      chan testDriverCommand
	returns []testClientReturn
}

const (
	cmdError = iota
	cmdOK
	cmdMessage
)

type mockTransportCommand struct {
	code    uint8
	message []byte
	err     error
}

type mockTransport struct {
	buf     streambuf.Buffer
	ch      chan []byte
	control chan mockTransportCommand
}

func newClientTestDriver(client mode.ProtocolClient) *testClientDriver {
	driver := &testClientDriver{
		client:  client,
		ch:      make(chan testDriverCommand),
		returns: nil,
	}

	go func() {
		for {
			cmd, ok := <-driver.ch
			if !ok {
				return
			}

			switch cmd.code {
			case driverCmdQuit:
				close(driver.ch)
				return
			case driverCmdPublish:
				events, err := driver.client.PublishEvents(cmd.events)
				n := len(cmd.events) - len(events)
				driver.returns = append(driver.returns, testClientReturn{n, err})
			}
		}
	}()

	return driver
}

func (t *testClientDriver) Stop() {
	t.ch <- testDriverCommand{code: driverCmdQuit}
}

func (t *testClientDriver) Publish(events []common.MapStr) {
	t.ch <- testDriverCommand{code: driverCmdPublish, events: events}
}

func (a mockAddr) Network() string { return "fake" }
func (a mockAddr) String() string  { return string(a) }

func newMockTransport() *mockTransport {
	return &mockTransport{
		ch:      make(chan []byte),
		control: make(chan mockTransportCommand),
	}
}

func (m *mockTransport) Connect(timeout time.Duration) error {
	return nil
}

func (m *mockTransport) IsConnected() bool {
	return true
}

func (m *mockTransport) Close() error {
	close(m.ch)
	close(m.control)
	return nil
}

func (m *mockTransport) Read(b []byte) (n int, err error) {
	cmd := <-m.control
	switch cmd.code {
	case cmdError:
		return 0, cmd.err
	case cmdOK:
		return 0, nil
	case cmdMessage:
		m.buf.Write(cmd.message)
		return m.buf.Read(b)
	}
	return 0, nil
}

func (m *mockTransport) Write(b []byte) (int, error) {
	m.ch <- b
	cmd := <-m.control
	switch cmd.code {
	case cmdError:
		return 0, cmd.err
	case cmdOK:
		return len(b), nil
	case cmdMessage:
		m.buf.Write(cmd.message)
		return len(b), nil
	}
	return 0, nil
}

func (m *mockTransport) recv(into io.Writer) {
	bytes, ok := <-m.ch
	if ok && len(bytes) > 0 {
		into.Write(bytes)
	}
}

func (m *mockTransport) sendError(e error) {
	m.control <- mockTransportCommand{code: cmdError, err: e}
}

func (m *mockTransport) sendOK() {
	m.control <- mockTransportCommand{code: cmdOK}
}

func (m *mockTransport) sendBytes(b []byte) {
	m.control <- mockTransportCommand{code: cmdMessage, message: b}
}

func (m *mockTransport) LocalAddr() net.Addr  { return mockAddr("client") }
func (m *mockTransport) RemoteAddr() net.Addr { return mockAddr("server") }

func (m *mockTransport) SetDeadline(t time.Time) error      { return nil }
func (m *mockTransport) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockTransport) SetWriteDeadline(t time.Time) error { return nil }

type message struct {
	code   uint8
	size   uint32
	seq    uint32
	events []*message
	doc    document
}

type document map[string]interface{}

func (d document) get(path string) interface{} {
	doc := d
	elems := strings.Split(path, ".")
	for i := 0; i < len(elems)-1; i++ {
		doc = doc[elems[i]].(map[string]interface{})
	}
	return doc[elems[len(elems)-1]]
}

func readMessage(buf *streambuf.Buffer) (*message, error) {
	if !buf.Avail(2) {
		return nil, nil
	}

	version, _ := buf.ReadNetUint8At(0)
	if version != '2' {
		return nil, errors.New("version error")
	}

	code, _ := buf.ReadNetUint8At(1)
	switch code {
	case 'W':
		if !buf.Avail(6) {
			return nil, nil
		}
		size, _ := buf.ReadNetUint32At(2)
		buf.Advance(6)
		buf.Reset()
		return &message{code: code, size: size}, buf.Err()
	case 'C':
		if !buf.Avail(6) {
			return nil, nil
		}
		len, _ := buf.ReadNetUint32At(2)
		if !buf.Avail(int(len) + 6) {
			return nil, nil
		}
		buf.Advance(6)

		tmp, _ := buf.Collect(int(len))
		buf.Reset()

		dataBuf := streambuf.New(nil)
		// decompress data
		decomp, err := zlib.NewReader(streambuf.NewFixed(tmp))
		if err != nil {
			return nil, err
		}
		// dataBuf.ReadFrom(streambuf.NewFixed(tmp))
		dataBuf.ReadFrom(decomp)
		decomp.Close()

		// unpack data
		dataBuf.Fix()
		var events []*message
		for dataBuf.Len() > 0 {
			version, _ := dataBuf.ReadNetUint8()
			if version != '2' {
				return nil, errors.New("version error 2")
			}

			code, _ := dataBuf.ReadNetUint8()
			if code != 'J' {
				return nil, errors.New("expected json data frame")
			}

			seq, _ := dataBuf.ReadNetUint32()
			payloadLen, _ := dataBuf.ReadNetUint32()
			jsonRaw, _ := dataBuf.Collect(int(payloadLen))

			var doc interface{}
			err = json.Unmarshal(jsonRaw, &doc)
			if err != nil {
				return nil, err
			}

			events = append(events, &message{
				code: code,
				seq:  seq,
				doc:  doc.(map[string]interface{}),
			})
		}
		return &message{code: 'C', events: events}, nil
	default:
		return nil, errors.New("unknown code")
	}
}

func recvMessage(buf *streambuf.Buffer, transp *mockTransport) (*message, error) {
	for {
		transp.recv(buf)
		resp, err := readMessage(buf)
		transp.sendOK()
		if resp != nil || err != nil {
			return resp, err
		}
	}
}

func sendAck(transp *mockTransport, seq uint32) {
	buf := streambuf.New(nil)
	buf.WriteByte('2')
	buf.WriteByte('A')
	buf.WriteNetUint32(seq)
	transp.sendBytes(buf.Bytes())
}

const testMaxWindowSize = 64

func TestSendZero(t *testing.T) {
	transp := newMockTransport()
	client := newClientTestDriver(
		newLumberjackClient(transp, testMaxWindowSize, 5*time.Second))

	client.Publish(make([]common.MapStr, 0))

	client.Stop()
	transp.Close()

	assert.Equal(t, 1, len(client.returns))
	assert.Equal(t, 0, client.returns[0].n)
	assert.Nil(t, client.returns[0].err)
}

func TestSimpleEvent(t *testing.T) {
	transp := newMockTransport()
	client := newClientTestDriver(
		newLumberjackClient(transp, testMaxWindowSize, 5*time.Second))

	event := common.MapStr{"name": "me", "line": 10}
	client.Publish([]common.MapStr{event})

	// receive window message
	buf := streambuf.New(nil)
	win, err := recvMessage(buf, transp)
	assert.Nil(t, err)

	// receive data message
	msg, err := recvMessage(buf, transp)
	assert.Nil(t, err)

	// send ack
	sendAck(transp, 1)

	// stop test driver
	transp.Close()
	client.Stop()

	// validate
	assert.NotNil(t, win)
	assert.NotNil(t, msg)
	assert.Equal(t, 1, len(msg.events))
	msg = msg.events[0]
	assert.Equal(t, "me", msg.doc["name"])
	assert.Equal(t, 10.0, msg.doc["line"])
}

func TestStructuredEvent(t *testing.T) {
	transp := newMockTransport()
	client := newClientTestDriver(
		newLumberjackClient(transp, testMaxWindowSize, 5*time.Second))
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

	buf := streambuf.New(nil)
	win, err := recvMessage(buf, transp)
	assert.Nil(t, err)

	msg, err := recvMessage(buf, transp)
	assert.Nil(t, err)

	sendAck(transp, 1)

	transp.Close()
	client.Stop()

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
