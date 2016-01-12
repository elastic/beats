package flows

import (
	"bytes"
	"encoding/binary"
	"net"
)

type FlowID struct {
	flowID []byte
	flowIDMeta
	flow *Flow // remember associated flow for faster lookup
}

type flowIDMeta struct {
	flags FlowIDFlag

	// offsets into flowID
	offEth        int8
	offOutterVlan int8
	offVlan       int8
	offOutterIPv4 int8
	offIPv4       int8
	offOutterIPv6 int8
	offIPv6       int8
	offICMPv4     int8
	offICMPv6     int8
	offUDP        int8
	offTCP        int8
	offID         int8
}

type FlowIDFlag uint16

const (
	EthFlow        FlowIDFlag = (1 << 0)
	OutterVlanFlow FlowIDFlag = (1 << 1)
	VLanFlow       FlowIDFlag = (1 << 2)
	OutterIPv4Flow FlowIDFlag = (1 << 3)
	IPv4Flow       FlowIDFlag = (1 << 4)
	OutterIPv6Flow FlowIDFlag = (1 << 5)
	IPv6Flow       FlowIDFlag = (1 << 6)
	ICMPv4Flow     FlowIDFlag = (1 << 7)
	ICMPv6Flow     FlowIDFlag = (1 << 8)
	UDPFlow        FlowIDFlag = (1 << 9)
	TCPFlow        FlowIDFlag = (1 << 10)
	ConnectionID   FlowIDFlag = (1 << 11)
)

const (
	SizeEthFlowID    = 6 + 6     // source + dest mac address
	SizeVlanFlowID   = 2         // raw vlan id
	SizeIPv4FlowID   = 4 + 4     // source + dest ip
	SizeIPv6FlowID   = 16 + 16   // source + dest ip
	SizeICMPFlowID   = 2         // icmp identifier (if present)
	SizeTCPFlowID    = 2 + 2 + 2 // source + dest port + connection id
	SizeUDPFlowID    = 2 + 2     // source + dest port
	SizeConnectionID = 8         // 64bit internal connection id

	SizeFlowIDMax = SizeEthFlowID +
		2*(SizeVlanFlowID+SizeIPv4FlowID+SizeIPv6FlowID) +
		SizeICMPFlowID +
		SizeTCPFlowID +
		SizeUDPFlowID +
		SizeConnectionID
)

func (f *FlowID) Reset(buf []byte) {
	f.flowID = buf
	f.flags = 0
	f.offEth = -1
	f.offOutterVlan = -1
	f.offVlan = -1
	f.offOutterIPv4 = -1
	f.offIPv4 = -1
	f.offOutterIPv6 = -1
	f.offIPv6 = -1
	f.offICMPv4 = -1
	f.offICMPv6 = -1
	f.offUDP = -1
	f.offTCP = -1
	f.offID = -1
	f.flow = nil
}

func (f *FlowID) Clone() *FlowID {
	n := *f
	n.flowID = make([]byte, len(f.flowID))
	copy(n.flowID, f.flowID)
	return &n
}

func FlowIDsEqual(f1, f2 *FlowID) bool {
	return f1.flags == f2.flags && bytes.Equal(f1.flowID, f2.flowID)
}

func (f *FlowID) Flags() FlowIDFlag {
	return f.flags
}

func (f *FlowID) Get(i FlowIDFlag) []byte {
	switch i {
	case EthFlow:
		return f.Eth()
	case OutterVlanFlow:
		return f.OutterVLan()
	case VLanFlow:
		return f.VLan()
	case OutterIPv4Flow:
		return f.OutterIPv4()
	case OutterIPv6Flow:
		return f.OutterIPv6()
	case IPv4Flow:
		return f.IPv4()
	case IPv6Flow:
		return f.IPv6()
	case ICMPv4Flow:
		return f.ICMPv4()
	case ICMPv6Flow:
		return f.ICMPv6()
	case UDPFlow:
		return f.UDP()
	case TCPFlow:
		return f.TCP()
	default:
		return nil
	}
}

func (f *FlowID) Eth() []byte {
	return f.extractID(f.offEth, SizeEthFlowID)
}

func (f *FlowID) AddEth(src, dst net.HardwareAddr) {
	f.addSimpleID(&f.offEth, EthFlow, src, dst)
}

func (f *FlowID) OutterVLan() []byte {
	return f.extractID(f.offOutterVlan, SizeVlanFlowID)
}

func (f *FlowID) VLan() []byte {
	return f.extractID(f.offVlan, SizeVlanFlowID)
}

func (f *FlowID) AddVLan(id uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], id)
	f.addID(&f.offVlan, &f.offOutterVlan, VLanFlow, OutterVlanFlow, tmp[:], nil)
}

func (f *FlowID) OutterIPv4() []byte {
	return f.extractID(f.offOutterIPv4, SizeIPv4FlowID)
}

func (f *FlowID) IPv4() []byte {
	return f.extractID(f.offIPv4, SizeIPv4FlowID)
}

func (f *FlowID) AddIPv4(src, dst net.IP) {
	f.addID(&f.offIPv4, &f.offOutterIPv4, IPv4Flow, OutterIPv4Flow, src, dst)
}

func (f *FlowID) OutterIPv6() []byte {
	return f.extractID(f.offOutterIPv6, SizeIPv6FlowID)
}

func (f *FlowID) IPv6() []byte {
	return f.extractID(f.offIPv6, SizeIPv6FlowID)
}

func (f *FlowID) AddIPv6(src, dst net.IP) {
	f.addID(&f.offIPv6, &f.offOutterIPv6, IPv6Flow, OutterIPv6Flow, src, dst)
}

func (f *FlowID) ICMPv4() []byte {
	return f.extractID(f.offICMPv4, SizeICMPFlowID)
}

func (f *FlowID) AddICMPv4(id uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], id)
	f.addSimpleID(&f.offICMPv4, ICMPv4Flow, tmp[:], nil)
}

func (f *FlowID) ICMPv6() []byte {
	return f.extractID(f.offICMPv6, SizeICMPFlowID)
}

func (f *FlowID) AddICMPv6(id uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], id)
	f.addSimpleID(&f.offICMPv6, ICMPv6Flow, tmp[:], nil)
}

func (f *FlowID) UDP() []byte {
	return f.extractID(f.offUDP, SizeUDPFlowID)
}

func (f *FlowID) AddUDP(src, dst uint16) {
	f.addWithPorts(&f.offUDP, UDPFlow, src, dst)
}

func (f *FlowID) TCP() []byte {
	return f.extractID(f.offTCP, SizeTCPFlowID)
}

func (f *FlowID) AddTCP(src, dst uint16) {
	f.addWithPorts(&f.offTCP, TCPFlow, src, dst)
}

func (f *FlowID) ConnectionID() []byte {
	return f.extractID(f.offID, SizeConnectionID)
}

func (f *FlowID) AddConnectionID(id uint64) {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], id)
	f.addSimpleID(&f.offID, ConnectionID, tmp[:], nil)
}

func (f *FlowID) addWithPorts(off *int8, flag FlowIDFlag, src, dst uint16) {
	var a, b [2]byte
	binary.LittleEndian.PutUint16(a[:], src)
	binary.LittleEndian.PutUint16(b[:], dst)
	f.addSimpleID(off, flag, a[:], b[:])
}

func (f *FlowID) extractID(off, sz int8) []byte {
	if off < 0 {
		return nil
	}
	return f.flowID[off : off+sz]
}

func (f *FlowID) addID(
	off, outterOff *int8,
	flag, outterFlag FlowIDFlag,
	a, b []byte,
) {
	if bytes.Compare(a, b) > 0 {
		a, b = b, a
	}

	flags := f.flags & (flag | outterFlag)
	switch flags {
	case flag | outterFlag:
		*outterOff, *off = *off, *outterOff
		la := copy(f.flowID[(*off):], a)
		copy(f.flowID[la+int(*off):], b)

	case flag:
		*outterOff = *off
		*off = int8(len(f.flowID))
		f.flowID = append(append(f.flowID, a...), b...)
		f.flags |= outterFlag

	default:
		*off = int8(len(f.flowID))
		f.flowID = append(append(f.flowID, a...), b...)
		f.flags |= flag

	}
}

func (f *FlowID) addSimpleID(
	off *int8,
	flag FlowIDFlag,
	a, b []byte,
) {
	if bytes.Compare(a, b) > 0 {
		a, b = b, a
	}

	if *off < 0 {
		*off = int8(len(f.flowID))
		f.flowID = append(append(f.flowID, a...), b...)
		f.flags |= flag
	} else {
		la := copy(f.flowID[(*off):], a)
		copy(f.flowID[la+int(*off):], b)
	}
}
