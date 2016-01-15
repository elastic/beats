package flows

import (
	"bytes"
	"encoding/binary"
	"net"
)

type FlowID struct {
	flowID []byte
	flowIDMeta

	dir  flowDirection
	flow Flow // remember associated flow for faster lookup
}

type flowIDMeta struct {
	flags FlowIDFlag

	// offsets into flowID
	offEth        uint8
	offOutterVlan uint8
	offVlan       uint8
	offOutterIPv4 uint8
	offIPv4       uint8
	offOutterIPv6 uint8
	offIPv6       uint8
	offICMPv4     uint8
	offICMPv6     uint8
	offUDP        uint8
	offTCP        uint8
	offID         uint8
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
	SizeChecksum     = 4         // checksum
	SizeEthFlowID    = 6 + 6     // source + dest mac address
	SizeVlanFlowID   = 2         // raw vlan id
	SizeIPv4FlowID   = 4 + 4     // source + dest ip
	SizeIPv6FlowID   = 16 + 16   // source + dest ip
	SizeICMPFlowID   = 2         // icmp identifier (if present)
	SizeTCPFlowID    = 2 + 2 + 2 // source + dest port + connection id
	SizeUDPFlowID    = 2 + 2     // source + dest port
	SizeConnectionID = 8         // 64bit internal connection id

	SizeFlowIDMax int = SizeChecksum + SizeEthFlowID +
		2*(SizeVlanFlowID+SizeIPv4FlowID+SizeIPv6FlowID) +
		SizeICMPFlowID +
		SizeTCPFlowID +
		SizeUDPFlowID +
		SizeConnectionID
)

const offUnset uint8 = 0xff

var flowIDEmptyMeta = flowIDMeta{
	flags: 0,

	offEth:        offUnset,
	offOutterVlan: offUnset,
	offVlan:       offUnset,
	offOutterIPv4: offUnset,
	offIPv4:       offUnset,
	offOutterIPv6: offUnset,
	offIPv6:       offUnset,
	offICMPv4:     offUnset,
	offICMPv6:     offUnset,
	offUDP:        offUnset,
	offTCP:        offUnset,
	offID:         offUnset,
}

type flowDirection int8

const (
	flowDirUnset flowDirection = iota - 1
	flowDirForward
	flowDirReversed
)

func init() {
	if SizeFlowIDMax > 255 {
		panic("SizeFlowIDMax exceeds size limit")
	}
}

func (f *FlowID) Reset(buf []byte) {
	f.flowID = buf
	f.flowIDMeta = flowIDEmptyMeta
	f.dir = flowDirUnset
	f.flow.stats = nil
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
	f.addID(&f.offEth, EthFlow, src, dst, flowDirUnset)
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
	f.addMultLayerID(
		&f.offVlan, &f.offOutterVlan,
		VLanFlow, OutterVlanFlow,
		tmp[:], nil, flowDirUnset)
}

func (f *FlowID) OutterIPv4() []byte {
	return f.extractID(f.offOutterIPv4, SizeIPv4FlowID)
}

func (f *FlowID) IPv4() []byte {
	return f.extractID(f.offIPv4, SizeIPv4FlowID)
}

func (f *FlowID) AddIPv4(src, dst net.IP) {
	f.addMultLayerID(
		&f.offIPv4, &f.offOutterIPv4,
		IPv4Flow, OutterIPv4Flow,
		src, dst, flowDirUnset)
}

func (f *FlowID) OutterIPv6() []byte {
	return f.extractID(f.offOutterIPv6, SizeIPv6FlowID)
}

func (f *FlowID) IPv6() []byte {
	return f.extractID(f.offIPv6, SizeIPv6FlowID)
}

func (f *FlowID) AddIPv6(src, dst net.IP) {
	f.addMultLayerID(
		&f.offIPv6, &f.offOutterIPv6,
		IPv6Flow, OutterIPv6Flow,
		src, dst, flowDirUnset)
}

func (f *FlowID) ICMPv4() []byte {
	return f.extractID(f.offICMPv4, SizeICMPFlowID)
}

func (f *FlowID) AddICMPv4Request(id uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], id)
	f.addID(&f.offICMPv4, ICMPv4Flow, tmp[:], nil, flowDirForward)
}

func (f *FlowID) AddICMPv4Response(id uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], id)
	f.addID(&f.offICMPv4, ICMPv4Flow, tmp[:], nil, flowDirReversed)
}

func (f *FlowID) ICMPv6() []byte {
	return f.extractID(f.offICMPv6, SizeICMPFlowID)
}

func (f *FlowID) AddICMPv6Request(id uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], id)
	f.addID(&f.offICMPv6, ICMPv6Flow, tmp[:], nil, flowDirForward)
}

func (f *FlowID) AddICMPv6Response(id uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], id)
	f.addID(&f.offICMPv6, ICMPv6Flow, tmp[:], nil, flowDirReversed)
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
	f.addID(&f.offID, ConnectionID, tmp[:], nil, flowDirUnset)
}

func (f *FlowID) addWithPorts(
	off *uint8,
	flag FlowIDFlag,
	src, dst uint16,
) {
	var a, b [2]byte
	binary.LittleEndian.PutUint16(a[:], src)
	binary.LittleEndian.PutUint16(b[:], dst)
	f.addID(off, flag, a[:], b[:], flowDirUnset)
}

func (f *FlowID) extractID(off, sz uint8) []byte {
	if off == offUnset {
		return nil
	}

	{
		off := int(off)
		sz := int(sz)
		return f.flowID[off : off+sz]
	}
}

func (f *FlowID) addMultLayerID(
	off, outterOff *uint8,
	flag, outterFlag FlowIDFlag,
	a, b []byte,
	hint flowDirection,
) {
	a, b = f.sortAddr(a, b, hint)

	flags := f.flags & (flag | outterFlag)
	switch flags {
	case flag | outterFlag:
		*outterOff, *off = *off, *outterOff
		la := copy(f.flowID[(*off):], a)
		copy(f.flowID[la+int(*off):], b)

	case flag:
		*outterOff = *off
		*off = uint8(len(f.flowID))
		f.flowID = append(append(f.flowID, a...), b...)
		f.flags |= outterFlag

	default:
		*off = uint8(len(f.flowID))
		f.flowID = append(append(f.flowID, a...), b...)
		f.flags |= flag

	}
}

func (f *FlowID) addID(
	off *uint8,
	flag FlowIDFlag,
	a, b []byte,
	hint flowDirection,
) {
	a, b = f.sortAddr(a, b, hint)

	if *off < 0 {
		*off = uint8(len(f.flowID))
		f.flowID = append(append(f.flowID, a...), b...)
		f.flags |= flag
	} else {
		la := copy(f.flowID[(*off):], a)
		copy(f.flowID[la+int(*off):], b)
	}
}

func (f *FlowID) sortAddr(a, b []byte, hint flowDirection) ([]byte, []byte) {
	if b == nil {
		if f.dir == flowDirUnset {
			f.dir = hint
		}
		return a, b
	}

	switch f.dir {
	case flowDirForward:
		return a, b
	case flowDirReversed:
		return b, a
	}

	switch bytes.Compare(a, b) {
	case -1:
		f.dir = flowDirForward
	case 1:
		f.dir = flowDirReversed
		a, b = b, a
	case 0:
		f.dir = hint
	}
	return a, b
}
