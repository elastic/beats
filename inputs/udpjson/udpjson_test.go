package udpjson

import (
	"net"
	"testing"
	"time"

	"github.com/elastic/packetbeat/common"
	"github.com/elastic/packetbeat/logp"

	"github.com/stretchr/testify/assert"
)

func TestUdpJson(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"udpjson"})
	}

	events := make(chan common.MapStr)
	server := new(Udpjson)

	server.Config = Config{
		Port:    0,
		BindIp:  "127.0.0.1",
		Timeout: 10 * time.Millisecond,
	}
	err := server.Init(true, events)
	assert.Nil(t, err)

	ready := make(chan bool)

	go func() {
		ready <- true
		err := server.Run()
		assert.Nil(t, err, "Error: %v", err)
	}()

	// make sure the goroutine runs first
	<-ready

	logp.Debug("udpjson", server.conn.LocalAddr().String())

	// send a message
	clientConn, err := net.Dial("udp", server.conn.LocalAddr().String())
	assert.Nil(t, err, "Error: %v", err)

	_, err = clientConn.Write([]byte(`{"hello": "udpserver"}`))
	assert.Nil(t, err)

	obj := <-events
	assert.Equal(t, obj["hello"].(string), "udpserver")
	_, ok := obj["@timestamp"].(common.Time)
	assert.True(t, ok)

	_, err = clientConn.Write([]byte(`{"obj2": 4}`))
	assert.Nil(t, err)
	_, ok = obj["@timestamp"].(common.Time)
	assert.True(t, ok)

	obj = <-events
	assert.Equal(t, obj["obj2"].(float64), 4)

	server.Stop()
}
