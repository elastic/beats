package tcp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/inputsource"
)

func TestCreateEvent(t *testing.T) {
	hello := "hello world"
	ip := "127.0.0.1"
	parsedIP := net.ParseIP(ip)
	addr := &net.IPAddr{IP: parsedIP, Zone: ""}

	message := []byte(hello)
	mt := inputsource.NetworkMetadata{RemoteAddr: addr}

	data := createEvent(message, mt)
	event := data.GetEvent()

	m, err := event.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, string(message), m)

	from, _ := event.GetValue("source")
	assert.Equal(t, ip, from)
}
