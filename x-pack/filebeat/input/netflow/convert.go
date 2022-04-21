// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/flowhash"
	"github.com/elastic/beats/v7/libbeat/conditions"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
)

func toBeatEvent(flow record.Record, internalNetworks []string) (event beat.Event) {
	switch flow.Type {
	case record.Flow:
		return flowToBeatEvent(flow, internalNetworks)
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
		"created":  time.Now().UTC(),
		"kind":     "event",
		"category": []string{"network_traffic", "network"},
		"action":   flow.Fields["type"],
	}
	if ecsEvent["action"] == "netflow_flow" {
		ecsEvent["type"] = []string{"connection"}
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

func flowToBeatEvent(flow record.Record, internalNetworks []string) (event beat.Event) {
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
	// ECS Fields -- source, destination & related.ip
	ecsSource := common.MapStr{}
	ecsDest := common.MapStr{}
	var relatedIP []net.IP

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
				ecsSource["locality"] = getIPLocality(internalNetworks, srcIP).String()
			}
			ecsSource["mac"] = srcMac
			if dstIP != nil {
				ecsDest["ip"] = dstIP
				ecsDest["locality"] = getIPLocality(internalNetworks, dstIP).String()
			}
			ecsDest["mac"] = dstMac
		}
	}

	// Regular IPv4 fields
	if ip, found := getKeyIP(flow.Fields, "sourceIPv4Address"); found {
		ecsSource["ip"] = ip
		relatedIP = append(relatedIP, ip)
		ecsSource["locality"] = getIPLocality(internalNetworks, ip).String()
	} else if ip, found := getKeyIP(flow.Fields, "sourceIPv6Address"); found {
		ecsSource["ip"] = ip
		relatedIP = append(relatedIP, ip)
		ecsSource["locality"] = getIPLocality(internalNetworks, ip).String()
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
		relatedIP = append(relatedIP, ip)
		ecsDest["locality"] = getIPLocality(internalNetworks, ip).String()
	} else if ip, found := getKeyIP(flow.Fields, "destinationIPv6Address"); found {
		ecsDest["ip"] = ip
		relatedIP = append(relatedIP, ip)
		ecsDest["locality"] = getIPLocality(internalNetworks, ip).String()
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
	ecsFlow["locality"] = getIPLocality(internalNetworks, srcIP, dstIP).String()

	// ECS Fields -- network
	ecsNetwork := common.MapStr{}
	if proto, found := getKeyUint64(flow.Fields, "protocolIdentifier"); found {
		ecsNetwork["transport"] = IPProtocol(proto).String()
		ecsNetwork["iana_number"] = proto
	}
	countBytes, hasBytes := getKeyUint64Alternatives(flow.Fields, "octetDeltaCount", "octetTotalCount", "initiatorOctets")
	countPkts, hasPkts := getKeyUint64Alternatives(flow.Fields, "packetDeltaCount", "packetTotalCount", "initiatorPackets")
	revBytes, hasRevBytes := getKeyUint64Alternatives(flow.Fields, "reverseOctetDeltaCount", "reverseOctetTotalCount", "responderOctets")
	revPkts, hasRevPkts := getKeyUint64Alternatives(flow.Fields, "reversePacketDeltaCount", "reversePacketTotalCount", "responderPackets")

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
	if len(relatedIP) > 0 {
		event.Fields["related"] = common.MapStr{"ip": uniqueIPs(relatedIP)}
	}
	return
}

// unique returns ips lexically sorted and with repeated elements
// omitted.
func uniqueIPs(ips []net.IP) []net.IP {
	if len(ips) < 2 {
		return ips
	}
	sort.Slice(ips, func(i, j int) bool { return bytes.Compare(ips[i], ips[j]) < 0 })
	curr := 0
	for i, ip := range ips {
		if ip.Equal(ips[curr]) {
			continue
		}
		curr++
		if curr < i {
			ips[curr], ips[i] = ips[i], nil
		}
	}
	return ips[:curr+1]
}

func getKeyUint64(dict record.Map, key string) (value uint64, found bool) {
	iface, found := dict[key]
	if !found {
		return
	}
	value, found = iface.(uint64)
	return
}

func getKeyUint64Alternatives(dict record.Map, keys ...string) (value uint64, found bool) {
	var iface interface{}
	for _, key := range keys {
		if iface, found = dict[key]; found {
			if value, found = iface.(uint64); found {
				return
			}
		}
	}
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

// Locality is an enum representing the locality of a network address.
type Locality uint8

const (
	// LocalityInternal identifies addresses that are internal to the organization.
	LocalityInternal Locality = iota + 1

	// LocalityExternal identifies addresses that are outside of the organization.
	LocalityExternal
)

var localityNames = map[Locality]string{
	LocalityInternal: "internal",
	LocalityExternal: "external",
}

func (l Locality) String() string {
	name, found := localityNames[l]
	if found {
		return name
	}
	return "unknown (" + strconv.Itoa(int(l)) + ")"
}

func isLocal(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsUnspecified() ||
		ip.Equal(net.IPv4bcast) ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast()
}

func getIPLocality(internalNetworks []string, ips ...net.IP) Locality {
	for _, ip := range ips {
		contains, err := conditions.NetworkContains(ip, internalNetworks...)
		if err != nil {
			return LocalityExternal
		}
		// always consider loopback/link-local private
		if !contains && !isLocal(ip) {
			return LocalityExternal
		}
	}
	return LocalityInternal
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
