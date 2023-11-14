package network

import (
	"context"
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
)

// StartPacketHandle starts the packet capture process
// As of now, this uses AF_PACKET, which is linux-only. However, unlike pcap, we're not
// importing libpcap, which would introduce another runtime dependency into metricbeat.
func StartPacketHandle(ctx context.Context, watcher *procs.ProcessesWatcher, procTracker *Tracker) error {
	afHandle, err := afpacket.NewTPacket(afpacket.SocketRaw)
	if err != nil {
		return fmt.Errorf("error creating afpacket interface: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			afHandle.Close()
			return nil
		default:
		}

		packet, ci, perr := afHandle.ZeroCopyReadPacketData()
		if perr != nil {
			return fmt.Errorf("error reading packet data: %w", err)
		}

		parsed := gopacket.NewPacket(packet, layers.LinkTypeEthernet, gopacket.NoCopy)

		tuple, valid := createTuple(parsed)
		if !valid {
			continue
		}
		layerType := applayer.TransportTCP
		if transLayer := parsed.TransportLayer(); transLayer != nil {
			if transLayer.LayerType() == layers.LayerTypeUDP {
				layerType = applayer.TransportUDP
			}
		}

		procInfo := watcher.FindProcessesTuple(&tuple, layerType)
		procTracker.Update(ci.CaptureLength, layerType, procInfo)

	}

}

func createTuple(parsed gopacket.Packet) (common.IPPortTuple, bool) {
	// all the gopacket.Packet methods love to panic if you've done something wrong,
	// so unpack things carefully. Don't skip nil interface checks.
	networkData := parsed.NetworkLayer()
	dstIP := net.IP{}
	srcIP := net.IP{}
	ipType := 4
	valid := false
	if networkData != nil {
		if ipv4handle, ok := networkData.(*layers.IPv4); ok {
			dstIP = ipv4handle.DstIP
			srcIP = ipv4handle.SrcIP
		}
		if ipv6handle, ok := networkData.(*layers.IPv6); ok {
			dstIP = ipv6handle.DstIP
			srcIP = ipv6handle.SrcIP
		}

		if networkData.LayerType() == layers.LayerTypeIPv6 {
			ipType = 16
		}
	}

	transportData := parsed.TransportLayer()
	var srcPort, dstPort uint16
	if transportData != nil {
		if udpHandle, ok := transportData.(*layers.UDP); ok {
			valid = true
			srcPort = uint16(udpHandle.SrcPort)
			dstPort = uint16(udpHandle.DstPort)
		}
		if tcpHandle, ok := transportData.(*layers.TCP); ok {
			valid = true
			srcPort = uint16(tcpHandle.SrcPort)
			dstPort = uint16(tcpHandle.DstPort)
		}
	}

	return common.NewIPPortTuple(ipType, srcIP, srcPort, dstIP, dstPort), valid
}
