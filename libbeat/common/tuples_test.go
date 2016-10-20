// +build !integration

package common

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTuples_tuples_ipv4(t *testing.T) {
	assert := assert.New(t)

	var tuple IPPortTuple

	// from net/ip.go
	var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

	tuple = NewIPPortTuple(4, net.IPv4(192, 168, 0, 1), 9200, net.IPv4(192, 168, 0, 2), 9201)

	assert.Equal(v4InV6Prefix, tuple.raw[0:12], "prefix_src")
	assert.Equal([]byte{192, 168, 0, 1}, tuple.raw[12:16], "src_ip")
	assert.Equal([]byte{0x23, 0xf0}, tuple.raw[16:18], "src_port")

	assert.Equal(v4InV6Prefix, tuple.raw[18:30], "prefix_dst")
	assert.Equal([]byte{192, 168, 0, 2}, tuple.raw[30:34], "dst_ip")
	assert.Equal([]byte{0x23, 0xf1}, tuple.raw[34:36], "dst_port")
	assert.Equal(36, len(tuple.raw))

	assert.Equal(v4InV6Prefix, tuple.revRaw[0:12], "rev prefix_dst")
	assert.Equal([]byte{192, 168, 0, 2}, tuple.revRaw[12:16], "rev dst_ip")
	assert.Equal([]byte{0x23, 0xf1}, tuple.revRaw[16:18], "rev dst_port")

	assert.Equal(v4InV6Prefix, tuple.revRaw[18:30], "rev prefix_src")
	assert.Equal([]byte{192, 168, 0, 1}, tuple.revRaw[30:34], "rev src_ip")
	assert.Equal([]byte{0x23, 0xf0}, tuple.revRaw[34:36], "rev src_port")
	assert.Equal(36, len(tuple.revRaw))

	tcpTuple := TCPTupleFromIPPort(&tuple, 1)
	assert.Equal(tuple.raw[:], tcpTuple.raw[0:36], "Wrong TCP tuple hashable")
	assert.Equal([]byte{0, 0, 0, 1}, tcpTuple.raw[36:40], "stream_id")
}

func TestTuples_tuples_ipv6(t *testing.T) {
	assert := assert.New(t)

	var tuple IPPortTuple

	tuple = NewIPPortTuple(16, net.ParseIP("2001:db8::1"),
		9200, net.ParseIP("2001:db8::123:12:1"), 9201)

	ip1 := []byte{0x20, 0x1, 0xd, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x1}
	ip2 := []byte{0x20, 0x1, 0xd, 0xb8, 0, 0, 0, 0, 0, 0, 0x1, 0x23, 0, 0x12, 0, 0x1}

	assert.Equal(ip1, tuple.raw[0:16], "src_ip")
	assert.Equal([]byte{0x23, 0xf0}, tuple.raw[16:18], "src_port")

	assert.Equal(ip2, tuple.raw[18:34], "dst_ip")
	assert.Equal([]byte{0x23, 0xf1}, tuple.raw[34:36], "dst_port")
	assert.Equal(36, len(tuple.raw))

	assert.Equal(ip2, tuple.revRaw[0:16], "rev dst_ip")
	assert.Equal([]byte{0x23, 0xf1}, tuple.revRaw[16:18], "rev dst_port")

	assert.Equal(ip1, tuple.revRaw[18:34], "rev src_ip")
	assert.Equal([]byte{0x23, 0xf0}, tuple.revRaw[34:36], "rev src_port")
	assert.Equal(36, len(tuple.revRaw))

	tcpTuple := TCPTupleFromIPPort(&tuple, 1)
	assert.Equal(tuple.raw[:], tcpTuple.raw[0:36], "Wrong TCP tuple hashable")
	assert.Equal([]byte{0, 0, 0, 1}, tcpTuple.raw[36:40], "stream_id")
}
