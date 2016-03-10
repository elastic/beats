// Need for unit and integration tests

package logstash

import (
	"compress/zlib"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/stretchr/testify/assert"
)

type protocolServer struct {
	*mockServer
}

type mockConn struct {
	conn net.Conn
	buf  *streambuf.Buffer
}

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

func newProtoServerTCP(t *testing.T, to time.Duration) *protocolServer {
	return &protocolServer{newMockServerTCP(t, to, "", nil)}
}

func (s *protocolServer) connectPair(compressLevel int) (*mockConn, *protocol, error) {
	client, transp, err := s.mockServer.connectPair(1 * time.Second)
	if err != nil {
		return nil, nil, err
	}

	proto, err := newClientProcol(transp, 100*time.Millisecond, compressLevel)
	if err != nil {
		return nil, nil, err
	}

	conn := &mockConn{client, streambuf.New(nil)}
	return conn, proto, nil
}

func readMessageType(buf *streambuf.Buffer) (byte, error) {
	if !buf.Avail(2) {
		return 0, nil
	}

	version, _ := buf.ReadNetUint8At(0)
	if version != '2' {
		return 0, errors.New("version error")
	}

	code, _ := buf.ReadNetUint8At(1)
	return code, nil
}

func (c *mockConn) recvMessage() (*message, error) {
	var err error
	for {
		msg, readerr := readMessage(c.buf)
		if readerr != nil {
			if err != nil {
				return nil, err
			}
			if readerr != streambuf.ErrNoMoreBytes {
				return nil, readerr
			}
		}

		if msg != nil {
			return msg, nil
		}

		var n int
		var buf [1024]byte
		n, err = c.conn.Read(buf[:])
		c.buf.Write(buf[:n])
	}
}

func (c *mockConn) recvDocs(count uint32) ([]common.MapStr, error) {
	var docs []*message
	for len(docs) < int(count) {
		msg, err := c.recvMessage()
		if err != nil {
			return nil, err
		}

		docs = append(docs, msg.events...)
	}

	var ret []common.MapStr
	for _, v := range docs {
		ret = append(ret, common.MapStr(v.doc))
	}

	return ret, nil
}

func (c *mockConn) sendACK(seq uint32) {
	buf := streambuf.New(nil)
	buf.WriteByte('2')
	buf.WriteByte('A')
	buf.WriteNetUint32(seq)
	c.conn.Write(buf.Bytes())
}

func readWindowSize(buf *streambuf.Buffer) (uint32, error) {
	return buf.ReadNetUint32At(2)
}

func readMessage(buf *streambuf.Buffer) (*message, error) {
	if !buf.Avail(2) {
		return nil, streambuf.ErrNoMoreBytes
	}

	version, _ := buf.ReadNetUint8At(0)
	if version != '2' {
		return nil, errors.New("version error")
	}

	code, _ := buf.ReadNetUint8At(1)
	switch code {
	case 'W':
		if !buf.Avail(6) {
			return nil, streambuf.ErrNoMoreBytes
		}
		size, _ := buf.ReadNetUint32At(2)
		buf.Advance(6)
		buf.Reset()
		return &message{code: code, size: size}, buf.Err()
	case 'C':
		if !buf.Avail(6) {
			return nil, streambuf.ErrNoMoreBytes
		}
		len, _ := buf.ReadNetUint32At(2)
		if !buf.Avail(int(len) + 6) {
			return nil, streambuf.ErrNoMoreBytes
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
	case 'J':
		if !buf.Avail(10) {
			return nil, streambuf.ErrNoMoreBytes
		}

		seq, _ := buf.ReadNetUint32At(2)
		payloadLen, _ := buf.ReadNetUint32At(6)

		if !buf.Avail(10 + int(payloadLen)) {
			return nil, streambuf.ErrNoMoreBytes
		}

		buf.Advance(10)
		tmp, _ := buf.Collect(int(payloadLen))

		var doc interface{}
		err := json.Unmarshal(tmp, &doc)
		if err != nil {
			return nil, err
		}

		event := &message{
			code: 'J',
			seq:  seq,
			doc:  doc.(map[string]interface{}),
		}
		return &message{code: 'J', events: []*message{event}}, nil
	default:
		return nil, errors.New("unknown code")
	}
}

func TestInvalidCompressionLevel(t *testing.T) {
	conn := (net.Conn)(nil)
	p, err := newClientProcol(conn, 5*time.Second, 10)
	assert.Nil(t, p)
	assert.NotNil(t, err)
}

func TestProtocolZeroEvent(t *testing.T) {
	server := newProtoServerTCP(t, 100*time.Millisecond)
	defer server.Close()

	client, transp, err := server.connectPair(3)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	defer client.conn.Close()
	defer transp.conn.Close()

	events, err := transp.sendEvents(nil)
	assert.Nil(t, events)
	assert.Nil(t, err)
}

func TestProtocolCloseAfterWindowSize(t *testing.T) {
	server := newProtoServerTCP(t, 100*time.Millisecond)
	defer server.Close()

	client, transp, err := server.connectPair(3)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	// defer client.conn.Close()
	// defer transp.conn.Close()

	transp.sendEvents([]common.MapStr{common.MapStr{
		"message": "hello world",
	}})

	msg, err := client.recvMessage()
	if err != nil {
		t.Fatalf("failed to read window size message: %v", err)
	}
	client.conn.Close()

	_, err = transp.awaitACK(1)

	assert.Equal(t, 1, int(msg.size))
	assert.NotNil(t, err)
}

func testProtocolReturnWindowSizes(
	t *testing.T,
	n int, acks []int,
	expectErr bool,
	compressionLevel int,
) {
	server := newProtoServerTCP(t, 100*time.Millisecond)
	defer server.Close()

	client, transp, err := server.connectPair(compressionLevel)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	defer client.conn.Close()
	defer transp.conn.Close()

	events := []common.MapStr{}
	for i := 0; i < n; i++ {
		events = append(events, common.MapStr{"message": string(i)})
	}

	outEvents, err := transp.sendEvents(events)
	assert.NoError(t, err)

	msg, err := client.recvMessage()
	if err != nil {
		t.Fatalf("failed to read window size message: %v", err)
	}

	docs, err := client.recvDocs(uint32(n))
	if err != nil {
		t.Fatalf("failed to read events: %v", err)
	}

	for _, ack := range acks {
		client.sendACK(uint32(ack))
	}

	seq, err := transp.awaitACK(uint32(n))
	assert.Equal(t, outEvents, events)
	assert.Equal(t, docs, events)
	assert.Equal(t, n, int(seq))
	assert.Equal(t, n, int(msg.size))
	if expectErr {
		assert.NotNil(t, err)
	} else {
		assert.NoError(t, err)
	}
}

func TestProtocolReturnPartialWindowSizes(t *testing.T) {
	testProtocolReturnWindowSizes(t, 10, []int{2, 4, 6, 8, 10}, false, 0)
	testProtocolReturnWindowSizes(t, 10, []int{2, 4, 6, 8, 10}, false, 3)
}

func TestProtocolReturnCompleteWindowSize(t *testing.T) {
	testProtocolReturnWindowSizes(t, 10, []int{10}, false, 0)
	testProtocolReturnWindowSizes(t, 10, []int{10}, false, 3)
}

func TestProtocolReturnFalseWindowSizes(t *testing.T) {
	testProtocolReturnWindowSizes(t, 2, []int{0, 5}, true, 0)
	testProtocolReturnWindowSizes(t, 2, []int{0, 5}, true, 3)
}

func TestProtocolFailOnClosedConnection(t *testing.T) {
	N := 10
	server := newProtoServerTCP(t, 100*time.Millisecond)
	defer server.Close()

	client, transp, err := server.connectPair(3)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	defer client.conn.Close()
	defer transp.conn.Close()

	events := []common.MapStr{}
	for i := 0; i < N; i++ {
		events = append(events, common.MapStr{"message": i})
	}

	transp.conn.Close()
	outEvents, err := transp.sendEvents(events)
	assert.Len(t, outEvents, 0)
	assert.NotNil(t, err)
}
