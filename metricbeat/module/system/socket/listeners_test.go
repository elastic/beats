package socket

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListenerTable(t *testing.T) {
	l := NewListenerTable()

	proto := uint8(4)
	lAddr := net.ParseIP("192.0.2.1")
	httpPort := 80
	rAddr := net.ParseIP("198.18.0.1")
	ephemeralPort := 48199
	ipv6Addr := net.ParseIP("2001:db8:fe80::217:f2ff:fe07:ed62")

	// Any socket with remote port of 0 is listening.
	assert.Equal(t, Listening, l.Direction(proto, lAddr, httpPort, net.IPv4zero, 0))

	// Listener on 192.0.2.1:80
	l.Put(proto, lAddr, httpPort)

	assert.Equal(t, Incoming, l.Direction(proto, lAddr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outgoing, l.Direction(0, lAddr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outgoing, l.Direction(proto, lAddr, ephemeralPort, rAddr, ephemeralPort))

	// Listener on 0.0.0.0:80
	l.Reset()
	l.Put(proto, net.IPv4zero, httpPort)

	assert.Equal(t, Incoming, l.Direction(proto, lAddr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outgoing, l.Direction(proto, ipv6Addr, httpPort, rAddr, ephemeralPort))

	// Listener on :::80
	l.Reset()
	l.Put(proto, net.IPv6zero, httpPort)

	assert.Equal(t, Incoming, l.Direction(proto, ipv6Addr, httpPort, rAddr, ephemeralPort))
	assert.Equal(t, Outgoing, l.Direction(proto, lAddr, httpPort, rAddr, ephemeralPort))
}
