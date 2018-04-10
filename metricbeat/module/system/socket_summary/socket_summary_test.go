package socket_summary

import (
	"syscall"
	"testing"

	"github.com/shirou/gopsutil/net"
	"github.com/stretchr/testify/assert"
)

func getMockedConns() []net.ConnectionStat {
	return []net.ConnectionStat{
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_DGRAM,
			Status: "",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_DGRAM,
			Status: "",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "LISTEN",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "ESTABLISHED",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "ESTABLISHED",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "CLOSE",
		},
		net.ConnectionStat{
			Family: syscall.AF_INET,
			Type:   syscall.SOCK_STREAM,
			Status: "LISTEN",
		},
	}
}

func TestCalculateConnStats(t *testing.T) {
	conns := getMockedConns()
	metrics := calculateConnStats(conns)

	allConns, err := metrics.GetValue("all.connections")

	if err != nil {
		t.Fail()
	}

	allListens, err := metrics.GetValue("all.listening")

	if err != nil {
		t.Fail()
	}

	udpConns, err := metrics.GetValue("udp.all.connections")

	if err != nil {
		t.Fail()
	}

	tcpConns, err := metrics.GetValue("tcp.all.connections")

	if err != nil {
		t.Fail()
	}

	tcpListens, err := metrics.GetValue("tcp.all.listening")

	if err != nil {
		t.Fail()
	}

	assert.Equal(t, allConns, 7)
	assert.Equal(t, allListens, 2)
	assert.Equal(t, udpConns, 2)
	assert.Equal(t, tcpConns, 5)
	assert.Equal(t, tcpListens, 2)
}
