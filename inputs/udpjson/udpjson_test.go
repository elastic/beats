package udpjson

import (
	"net"
	"packetbeat/common"
	"packetbeat/logp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUdpJson(t *testing.T) {
	t.Skip("Skipped because it seems to hang on Travis CI")
	return

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, []string{"udpjson"})
	}

	events := make(chan common.MapStr)
	server, err := NewServer(Config{
		Port:   0,
		BindIp: "127.0.0.1",
	}, 10, events)
	defer server.Close()

	assert.Nil(t, err)
	assert.NotNil(t, server)

	go func() {
		err := server.ReceiveForever()
		assert.Nil(t, err)
	}()

	// send a message
	clientConn, err := net.Dial("udp", server.conn.LocalAddr().String())
	assert.Nil(t, err)

	_, err = clientConn.Write([]byte(`{"hello": "udpserver"}`))
	assert.Nil(t, err)

	obj := <-events
	assert.Equal(t, obj, common.MapStr{"hello": "udpserver"})

	_, err = clientConn.Write([]byte(`{"obj2": 4}`))
	assert.Nil(t, err)

	obj = <-events
	assert.Equal(t, obj, common.MapStr{"obj2": 4})

	server.Stop()
}
