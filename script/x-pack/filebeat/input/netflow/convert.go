// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"encoding/base64"
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/flowhash"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/record"
)

var (
	// RFC 1918
	privateIPv4 = []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.IPv4Mask(255, 0, 0, 0)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.IPv4Mask(255, 240, 0, 0)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.IPv4Mask(255, 255, 0, 0)},
	}

	// RFC 4193
	privateIPv6 = net.IPNet{
		IP:   net.IP{0xfd, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Mask: net.IPMask{0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
)

func toBeatEvent(flow record.Record) (event beat.Event) {
	switch flow.Type {
	case record.Flow:
		return flowToBeatEvent(flow)
	case record.Options:
		return optionsToBeatEvent(flow)
	default:
		return toBeatEventCommon(flow)
	}
}

func toBeatEventCommon(flow record.Record) (event beat.Event) {
	// replace net.HardwareAddress with its String() representation
	fixMacAddresses(flow.Fields)
	// Nest Exporter into netflow fields
	flow.Fields["exporter"] = fieldNameConverter.ToSnakeCase(flow.Exporter)

	// Nest Type into netflow fields
	switch flow.Type {
	case record.Flow:
		flow.Fields["type"] = "netflow_flow"
	case record.Options:
		flow.Fields["type"] = "netflow_options"
	default:
		flow.Fields["type"] = "netflow_unknown"
	}

	// ECS Fields -- event
	ecsEvent := common.MapStr{
		"created":  flow.Timestamp,
		"kind":     "event",
		"category": "network_traffic",
		"action":   flow.Fields["type"],
	}
	// ECS Fields -- device
	ecsDevice := common.MapStr{}
	if exporter, ok := getKeyString(flow.Exporter, "address"); ok {
		ecsDevice["ip"] = extractIPFromIPPort(exporter)
	}

	event.Timestamp = flow.Timestamp
	event.Fields = common.MapStr{
		"netflow":  fieldNameConverter.ToSnakeCase(flow.Fields),
		"event":    ecsEvent,
		"observer": ecsDevice,
	}
	return
}

func extractIPFromIPPort(address string) string {
	// address can be "n.n.n.n:port" or "[hhhh:hhhh::hhhh]:port"
	if lastColon := strings.LastIndexByte(address, ':'); lastColon > -1 {
		address = address[:lastColon]
	}
	if len(address) > 0 && address[0] == '[' {
		address = address[1:]
	}
	if n := len(address); n > 0 && address[n-1] == ']' {
		address = address[:n-1]
	}
	return address
}

func optionsToBeatEvent(flow record.Record) beat.Event {
	for _, key := range []string{"options", "scope"} {
		if iface, found := flow.Fields[key]; found {
			if opts, ok := iface.(record.Map); ok {
				fixMacAddresses(opts)
				flow.Fields[key] = fieldNameConverter.ToSnakeCase(opts)
			}
		}
	}
	return toBeatEventCommon(flow)
}

func flowToBeatEvent(flow record.Record) (event beat.Event) {
	event = toBeatEventCommon(flow)

	ecsEvent, ok := event.Fields["event"].(common.MapStr)
	if !ok {
		ecsEvent = common.MapStr{}
		event.Fields["event"] = ecsEvent
	}
	sysUptime, hasSysUptime := getKeyUint64(flow.Exporter, "uptimeMillis")
	if !hasSysUptime || sysUptime == 0 {
		// Alternative update
		sysUptime, hasSysUptime = getKeyUint64(flow.Fields, "systemInitTimeMilliseconds")
	}
	startUptime, hasStartUptime := getKeyUint64(flow.Fields, "flowStartSysUpTime")
	endUptime, hasEndUptime := getKeyUint64(flow.Fields, "flowEndSysUpTime")
	if hasSysUptime {
		// Can't convert uptime values to absolute time if sysUptime is bogus
		// It will result on a flow that starts and ends in the future.
		hasStartUptime = hasStartUptime && startUptime <= sysUptime
		hasEndUptime = hasEndUptime && endUptime <= sysUptime
		if hasStartUptime {
			ecsEvent["start"] = flow.Timestamp.Add((time.Duration(startUptime) - time.Duration(sysUptime)) * time.Millisecond)
		}
		if hasEndUptime {
			ecsEvent["end"] = flow.Timestamp.Add((time.Duration(endUptime) - time.Duration(sysUptime)) * time.Millisecond)
		}
		if hasStartUptime && hasEndUptime {
			ecsEvent["duration"] = ecsEvent["end"].(time.Time).Sub(ecsEvent["start"].(time.Time)).Nanoseconds()
		}
	}
	if ecsEvent["duration"] == nil {
		if durationMillis, found := getKeyUint64(flow.Fields, "flowDurationMilliseconds"); found {
			duration := time.Duration(durationMillis) * time.Millisecond
			ecsEvent["duration"] = duration

			// Here we're missing at least one of (start, end)
			if start := ecsEvent["start"]; start != nil {
				ecsEvent["end"] = start.(time.Time).Add(duration)
			} else if end := ecsEvent["end"]; end != nil {
				ecsEvent["start"] = end.(time.Time).Add(-duration)
			}
		}
	}

	flowDirection, hasFlowDirection := getKeyUint64(flow.Fields, "flowDirection")
	// ECS Fields -- source and destination
	ecsSource := common.MapStr{}
	ecsDest := common.MapStr{}

	// Populate first with WLAN fields
	if hasFlowDirection {
		staIP, _ := getKeyIP(flow.Fields, "staIPv4Address")
		staMac, hasStaMac := getKeyString(flow.Fields, "staMacAddress")
		wtpMac, hasWtpMac := getKeyString(flow.Fields, "wtpMacAddress")
		if hasStaMac && hasWtpMac {
			srcMac := staMac
			srcIP := staIP
			dstMac := wtpMac
			var dstIP net.IP = nil
			if Direction(flowDirection) == DirectionOutbound {
				srcMac, dstMac = dstMac, srcMac
				srcIP, dstIP = dstIP, srcIP
			}
			if srcIP != nil {
				ecsSource["ip"] = srcIP
				ecsSource["locality"] = getIPLocality(srcIP).String()
			}
			ecsSource["mac"] = srcMac
			if dstIP != nil {
				ecsDest["ip"] = dstIP
				ecsDest["locality"] = getIPLocality(dstIP).String()
			}
			ecsDest["mac"] = dstMac
		}
	}

	// Regular IPv4 fields
	if ip, found := getKeyIP(flow.Fields, "sourceIPv4Address"); found {
		ecsSource["ip"] = ip
		ecsSource["locality"] = getIPLocality(ip).String()
	}
	if sourcePort, found := getKeyUint64(flow.Fields, "sourceTransportPort"); found {
		ecsSource["port"] = sourcePort
	}
	if mac, found := getKeyString(flow.Fields, "sourceMacAddress"); found {
		ecsSource["mac"] = mac
	}

	// ECS Fields -- destination
	if ip, found := getKeyIP(flow.Fields, "destinationIPv4Address"); found {
		ecsDest["ip"] = ip
		ecsDest["locality"] = getIPLocality(ip).String()
	}
	if destPort, found := getKeyUint64(flow.Fields, "destinationTransportPort"); found {
		ecsDest["port"] = destPort
	}
	if mac, found := getKeyString(flow.Fields, "destinationMacAddress"); found {
		ecsDest["mac"] = mac
	}

	// ECS Fields -- Flow
	ecsFlow := common.MapStr{}
	var srcIP, dstIP net.IP
	var srcPort, dstPort uint16
	var protocol IPProtocol
	if ip, found := getKeyIP(record.Map(ecsSource), "ip"); found {
		srcIP = ip
	}
	if ip, found := getKeyIP(record.Map(ecsDest), "ip"); found {
		dstIP = ip
	}
	if port, found := getKeyUint64(flow.Fields, "sourceTransportPort"); found {
		srcPort = uint16(port)
	}
	if port, found := getKeyUint64(flow.Fields, "destinationTransportPort"); found {
		dstPort = uint16(port)
	}
	if proto, found := getKeyUint64(flow.Fields, "protocolIdentifier"); found {
		protocol = IPProtocol(proto)
	}
	if srcIP == nil {
		srcIP = net.IPv4(0, 0, 0, 0).To4()
	}
	if dstIP == nil {
		dstIP = net.IPv4(0, 0, 0, 0).To4()
	}
	ecsFlow["id"] = flowID(srcIP, dstIP, srcPort, dstPort, uint8(protocol))
	ecsFlow["locality"] = getIPLocality(srcIP, dstIP).String()

	// ECS Fields -- network
	ecsNetwork := common.MapStr{}
	if proto, found := getKeyUint64(flow.Fields, "protocolIdentifier"); found {
		ecsNetwork["transport"] = IPProtocol(proto).String()
		ecsNetwork["iana_number"] = proto
	}
	countBytes, hasBytes := getKeyUint64(flow.Fields, "octetDeltaCount")
	if !hasBytes {
		countBytes, hasBytes = getKeyUint64(flow.Fields, "octetTotalCount")
	}
	countPkts, hasPkts := getKeyUint64(flow.Fields, "packetDeltaCount")
	if !hasPkts {
		countPkts, hasPkts = getKeyUint64(flow.Fields, "packetTotalCount")
	}
	revBytes, hasRevBytes := getKeyUint64(flow.Fields, "reverseOctetDeltaCount")
	if !hasRevBytes {
		revBytes, hasRevBytes = getKeyUint64(flow.Fields, "reverseOctetTotalCount")
	}
	revPkts, hasRevPkts := getKeyUint64(flow.Fields, "reversePacketDeltaCount")
	if !hasRevPkts {
		revPkts, hasRevPkts = getKeyUint64(flow.Fields, "reversePacketTotalCount")
	}

	if hasRevBytes {
		ecsDest["bytes"] = revBytes
	}

	if hasRevPkts {
		ecsDest["packets"] = revPkts
	}

	if hasBytes {
		ecsSource["bytes"] = countBytes
		if hasRevBytes {
			countBytes += revBytes
		}
		ecsNetwork["bytes"] = countBytes
	}
	if hasPkts {
		ecsSource["packets"] = countPkts
		if hasRevPkts {
			countPkts += revPkts
		}
		ecsNetwork["packets"] = countPkts
	}

	if biflowDir, isBiflow := getKeyUint64(flow.Fields, "biflowDirection"); isBiflow && len(ecsSource) > 0 && len(ecsDest) > 0 {
		// swap source and destination if biflowDirection is reverseInitiator
		if biflowDir == 2 {
			ecsDest, ecsSource = ecsSource, ecsDest
		}
		ecsEvent["category"] = "network_session"

		// Assume source is the client in biflows.
		event.Fields["client"] = ecsSource
		event.Fields["server"] = ecsDest
	}

	ecsNetwork["direction"] = "unknown"
	if hasFlowDirection {
		ecsNetwork["direction"] = Direction(flowDirection).String()
	}
	if ssid, found := getKeyString(flow.Fields, "wlanSSID"); found {
		ecsNetwork["name"] = ssid
	}

	ecsNetwork["community_id"] = flowhash.CommunityID.Hash(flowhash.Flow{
		SourceIP:        srcIP,
		SourcePort:      srcPort,
		DestinationIP:   dstIP,
		DestinationPort: dstPort,
		Protocol:        uint8(protocol),
	})

	if len(ecsFlow) > 0 {
		event.Fields["flow"] = ecsFlow
	}
	if len(ecsSource) > 0 {
		event.Fields["source"] = ecsSource
	}
	if len(ecsDest) > 0 {
		event.Fields["destination"] = ecsDest
	}
	if len(ecsNetwork) > 0 {
		event.Fields["network"] = ecsNetwork
	}
	return
}

func getKeyUint64(dict record.Map, key string) (value uint64, found bool) {
	iface, found := dict[key]
	if !found {
		return
	}
	value, found = iface.(uint64)
	return
}

func getKeyString(dict record.Map, key string) (value string, found bool) {
	iface, found := dict[key]
	if !found {
		return
	}
	value, found = iface.(string)
	return
}

func getKeyIP(dict record.Map, key string) (value net.IP, found bool) {
	iface, found := dict[key]
	if !found {
		return
	}
	value, found = iface.(net.IP)
	return
}

// Replaces each net.HardwareAddr in the dictionary with its string representation
// because HardwareAddr doesn't implement Marshaler interface.
func fixMacAddresses(dict map[string]interface{}) {
	for key, value := range dict {
		if asMac, ok := value.(net.HardwareAddr); ok {
			dict[key] = asMac.String()
		}
	}
}

type Locality uint8

const (
	LocalityPrivate Locality = iota + 1
	LocalityPublic
)

var localityNames = map[Locality]string{
	LocalityPrivate: "private",
	LocalityPublic:  "public",
}

func (l Locality) String() string {
	name, found := localityNames[l]
	if found {
		return name
	}
	return "unknown (" + strconv.Itoa(int(l)) + ")"
}

func isPrivateNetwork(ip net.IP) bool {
	for _, net := range privateIPv4 {
		if net.Contains(ip) {
			return true
		}
	}

	return privateIPv6.Contains(ip)
}

func isLocalOrPrivate(ip net.IP) bool {
	return isPrivateNetwork(ip) ||
		ip.IsLoopback() ||
		ip.IsUnspecified() ||
		ip.Equal(net.IPv4bcast) ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast()
}

func getIPLocality(ip ...net.IP) Locality {
	for _, addr := range ip {
		if !isLocalOrPrivate(addr) {
			return LocalityPublic
		}
	}
	return LocalityPrivate
}

// TODO: create table from https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
// They have a CSV file available for conversion.

type IPProtocol uint8

const (
	ICMP     IPProtocol = 1
	TCP      IPProtocol = 6
	UDP      IPProtocol = 17
	IPv6ICMP IPProtocol = 58
)

var ipProtocolNames = map[IPProtocol]string{
	ICMP:     "icmp",
	TCP:      "tcp",
	UDP:      "udp",
	IPv6ICMP: "ipv6-icmp",
}

func (p IPProtocol) String() string {
	name, found := ipProtocolNames[p]
	if found {
		return name
	}
	return "unknown (" + strconv.Itoa(int(p)) + ")"
}

func flowID(srcIP, dstIP net.IP, srcPort, dstPort uint16, proto uint8) string {
	h := xxhash.New()
	// Both flows will have the same ID.
	if srcPort >= dstPort {
		h.Write(srcIP)
		binary.Write(h, binary.BigEndian, srcPort)
		h.Write(dstIP)
		binary.Write(h, binary.BigEndian, dstPort)
	} else {
		h.Write(dstIP)
		binary.Write(h, binary.BigEndian, dstPort)
		h.Write(srcIP)
		binary.Write(h, binary.BigEndian, srcPort)
	}
	binary.Write(h, binary.BigEndian, proto)

	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

type Direction uint8

const (
	// According to IPFIX flowDirection field definition
	DirectionInbound Direction = iota
	DirectionOutbound
)

var directionNames = map[Direction]string{
	DirectionInbound:  "inbound",
	DirectionOutbound: "outbound",
}

func (l Direction) String() string {
	name, found := directionNames[l]
	if found {
		return name
	}
	return "unknown (" + strconv.Itoa(int(l)) + ")"
}
