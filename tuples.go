package main

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
	ip_length          int
	Src_ip, Dst_ip     net.IP
	Src_port, Dst_port uint16

	raw    HashableIpPortTuple // Src_ip:Src_port:Dst_ip:Dst_port
	revRaw HashableIpPortTuple // Dst_ip:Dst_port:Src_ip:Src_port
}

func NewIpPortTuple(ip_length int, src_ip net.IP, src_port uint16,
	dst_ip net.IP, dst_port uint16) IpPortTuple {

	tuple := IpPortTuple{
		ip_length: ip_length,
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

const MaxTcpTupleRawSize = 16 + 16 + 2 + 2 + 4

type HashableTcpTuple [MaxTcpTupleRawSize]byte

type TcpTuple struct {
	ip_length          int
	Src_ip, Dst_ip     net.IP
	Src_port, Dst_port uint16
	stream_id          uint32

	raw HashableTcpTuple // Src_ip:Src_port:Dst_ip:Dst_port:stream_id
}

func TcpTupleFromIpPort(t *IpPortTuple, tcp_id uint32) TcpTuple {
	tuple := TcpTuple{
		ip_length: t.ip_length,
		Src_ip:    t.Src_ip,
		Dst_ip:    t.Dst_ip,
		Src_port:  t.Src_port,
		Dst_port:  t.Dst_port,
		stream_id: tcp_id,
	}
	tuple.ComputeHashebles()

	return tuple
}

func (t *TcpTuple) ComputeHashebles() {
	if t.ip_length == 4 {
		copy(t.raw[0:4], []byte(t.Src_ip))
		copy(t.raw[4:6], []byte{byte(t.Src_port >> 8), byte(t.Src_port)})
		copy(t.raw[6:10], []byte(t.Dst_ip))
		copy(t.raw[10:12], []byte{byte(t.Dst_port >> 8), byte(t.Dst_port)})
		copy(t.raw[12:16], []byte{byte(t.stream_id >> 24), byte(t.stream_id >> 16),
			byte(t.stream_id >> 8), byte(t.stream_id)})

	} else if t.ip_length == 16 {
		copy(t.raw[0:16], []byte(t.Src_ip))
		copy(t.raw[16:18], []byte{byte(t.Src_port >> 8), byte(t.Src_port)})
		copy(t.raw[18:34], []byte(t.Dst_ip))
		copy(t.raw[34:36], []byte{byte(t.Dst_port >> 8), byte(t.Dst_port)})
		copy(t.raw[36:38], []byte{byte(t.stream_id >> 24), byte(t.stream_id >> 16),
			byte(t.stream_id >> 8), byte(t.stream_id)})

	} else {
		panic("Unkown length")
	}
}

func (t TcpTuple) String() string {
	return fmt.Sprintf("TcpTuple src[%s:%d] dst[%s:%d] stream_id[%d]",
		t.Src_ip.String(),
		t.Src_port,
		t.Dst_ip.String(),
		t.Dst_port,
		t.stream_id)
}
