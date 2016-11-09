// +build !integration

package icmp

import (
	"encoding/hex"
	"net"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"

	"github.com/stretchr/testify/assert"
)

func TestIcmpIsLocalIp(t *testing.T) {
	icmp := icmpPlugin{localIps: []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(192, 168, 0, 2)}}

	assert.True(t, icmp.isLocalIP(net.IPv4(127, 0, 0, 1)), "loopback IP")
	assert.True(t, icmp.isLocalIP(net.IPv4(192, 168, 0, 1)), "local IP")
	assert.False(t, icmp.isLocalIP(net.IPv4(10, 0, 0, 1)), "remote IP")
}

func TestIcmpDirection(t *testing.T) {
	icmp := icmpPlugin{}

	trans1 := &icmpTransaction{tuple: icmpTuple{srcIP: net.IPv4(127, 0, 0, 1), dstIP: net.IPv4(127, 0, 0, 1)}}
	assert.Equal(t, uint8(directionLocalOnly), icmp.direction(trans1), "local communication")

	trans2 := &icmpTransaction{tuple: icmpTuple{srcIP: net.IPv4(10, 0, 0, 1), dstIP: net.IPv4(127, 0, 0, 1)}}
	assert.Equal(t, uint8(directionFromOutside), icmp.direction(trans2), "client to server")

	trans3 := &icmpTransaction{tuple: icmpTuple{srcIP: net.IPv4(127, 0, 0, 1), dstIP: net.IPv4(10, 0, 0, 1)}}
	assert.Equal(t, uint8(directionFromInside), icmp.direction(trans3), "server to client")
}

func BenchmarkIcmpProcessICMPv4(b *testing.B) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"icmp", "icmpdetailed"})
	}

	results := &publish.ChanTransactions{make(chan common.MapStr, 10)}
	icmp, err := New(true, results, common.NewConfig())
	if err != nil {
		b.Error("Failed to create ICMP processor")
		return
	}

	icmpRequestData := createICMPv4Layer(b, "08"+"00"+"0000"+"ffff"+"0001")
	packetRequestData := new(protos.Packet)

	icmpResponseData := createICMPv4Layer(b, "00"+"00"+"0000"+"ffff"+"0001")
	packetResponseData := new(protos.Packet)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		icmp.ProcessICMPv4(nil, icmpRequestData, packetRequestData)
		icmp.ProcessICMPv4(nil, icmpResponseData, packetResponseData)

		client := icmp.results.(*publish.ChanTransactions)
		<-client.Channel
	}
}

func createICMPv4Layer(b *testing.B, hexstr string) *layers.ICMPv4 {
	data, err := hex.DecodeString(hexstr)
	if err != nil {
		b.Error("Failed to decode hex string")
		return nil
	}

	var df gopacket.DecodeFeedback
	var icmp4 layers.ICMPv4
	err = icmp4.DecodeFromBytes(data, df)
	if err != nil {
		b.Error("Failed to decode ICMPv4 data")
		return nil
	}

	return &icmp4
}
