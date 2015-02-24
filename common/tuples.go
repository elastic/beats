package common

import (
	"fmt"
	"net"
)

// In order for the IpPortTuple and the TcpTuple to be used as
// hashtable keys, they need to have a fixed size. This means the
// net.IP is problematic because it's internally represented as a slice.
// We're introducing the HashableIpPortTuple and the HashableTcpTuple
// types which are internally simple byte arrays.

const MaxIpPortTupleRawSize = 16 + 16 + 2 + 2

type HashableIpPortTuple [MaxIpPortTupleRawSize]byte

type IpPortTuple struct {
	Ip_length          int
	Src_ip, Dst_ip     net.IP
	Src_port, Dst_port uint16

	raw    HashableIpPortTuple // Src_ip:Src_port:Dst_ip:Dst_port
	revRaw HashableIpPortTuple // Dst_ip:Dst_port:Src_ip:Src_port
}

func NewIpPortTuple(ip_length int, src_ip net.IP, src_port uint16,
	dst_ip net.IP, dst_port uint16) IpPortTuple {

	tuple := IpPortTuple{
		Ip_length: ip_length,
		Src_ip:    src_ip,
		Dst_ip:    dst_ip,
		Src_port:  src_port,
		Dst_port:  dst_port,
	}
	tuple.ComputeHashebles()

	return tuple
}

func (t *IpPortTuple) ComputeHashebles() {
	copy(t.raw[0:16], t.Src_ip)
	copy(t.raw[16:18], []byte{byte(t.Src_port >> 8), byte(t.Src_port)})
	copy(t.raw[18:34], t.Dst_ip)
	copy(t.raw[34:36], []byte{byte(t.Dst_port >> 8), byte(t.Dst_port)})

	copy(t.revRaw[0:16], t.Dst_ip)
	copy(t.revRaw[16:18], []byte{byte(t.Dst_port >> 8), byte(t.Dst_port)})
	copy(t.revRaw[18:34], t.Src_ip)
	copy(t.revRaw[34:36], []byte{byte(t.Src_port >> 8), byte(t.Src_port)})
}

func (t *IpPortTuple) String() string {
	return fmt.Sprintf("IpPortTuple src[%s:%d] dst[%s:%d]",
		t.Src_ip.String(),
		t.Src_port,
		t.Dst_ip.String(),
		t.Dst_port)
}

// Hashable returns a hashable value that uniquely identifies
// the IP-port tuple.
func (t *IpPortTuple) Hashable() HashableIpPortTuple {
	return t.raw
}

// Hashable returns a hashable value that uniquely identifies
// the IP-port tuple after swapping the source and destination.
func (t *IpPortTuple) RevHashable() HashableIpPortTuple {
	return t.revRaw
}

const MaxTcpTupleRawSize = 16 + 16 + 2 + 2 + 4

type HashableTcpTuple [MaxTcpTupleRawSize]byte

type TcpTuple struct {
	Ip_length          int
	Src_ip, Dst_ip     net.IP
	Src_port, Dst_port uint16
	Stream_id          uint32

	raw HashableTcpTuple // Src_ip:Src_port:Dst_ip:Dst_port:stream_id
}

func TcpTupleFromIpPort(t *IpPortTuple, tcp_id uint32) TcpTuple {
	tuple := TcpTuple{
		Ip_length: t.Ip_length,
		Src_ip:    t.Src_ip,
		Dst_ip:    t.Dst_ip,
		Src_port:  t.Src_port,
		Dst_port:  t.Dst_port,
		Stream_id: tcp_id,
	}
	tuple.ComputeHashebles()

	return tuple
}

func (t *TcpTuple) ComputeHashebles() {
	copy(t.raw[0:16], t.Src_ip)
	copy(t.raw[16:18], []byte{byte(t.Src_port >> 8), byte(t.Src_port)})
	copy(t.raw[18:34], t.Dst_ip)
	copy(t.raw[34:36], []byte{byte(t.Dst_port >> 8), byte(t.Dst_port)})
	copy(t.raw[36:40], []byte{byte(t.Stream_id >> 24), byte(t.Stream_id >> 16),
		byte(t.Stream_id >> 8), byte(t.Stream_id)})
}

func (t TcpTuple) String() string {
	return fmt.Sprintf("TcpTuple src[%s:%d] dst[%s:%d] stream_id[%d]",
		t.Src_ip.String(),
		t.Src_port,
		t.Dst_ip.String(),
		t.Dst_port,
		t.Stream_id)
}

// Returns a pointer to the equivalent IpPortTuple.
func (t TcpTuple) IpPort() *IpPortTuple {
	ipport := NewIpPortTuple(t.Ip_length, t.Src_ip, t.Src_port,
		t.Dst_ip, t.Dst_port)
	return &ipport
}

// Hashable() returns a hashable value that uniquely identifies
// the TCP tuple.
func (t *TcpTuple) Hashable() HashableTcpTuple {
	return t.raw
}

// Source and destination process names, as found by the proc module.
type CmdlineTuple struct {
	Src, Dst []byte
}
