package icmp

import (
	"fmt"
	"net"
)

// In order for the icmpTuple to be used as hashtable key, it needs to have
// a fixed size. This means the net.IP is problematic because it's internally
// represented as a slice. Therefore the hashableIcmpTuple type is introduced
// which internally is a simple byte array.

const maxIcmpTupleRawSize = 1 + 16 + 16 + 2 + 2

type hashableIcmpTuple [maxIcmpTupleRawSize]byte

type icmpTuple struct {
	IcmpVersion uint8
	SrcIp       net.IP
	DstIp       net.IP
	Id          uint16
	Seq         uint16
}

func (t *icmpTuple) Reverse() icmpTuple {
	return icmpTuple{
		IcmpVersion: t.IcmpVersion,
		SrcIp:       t.DstIp,
		DstIp:       t.SrcIp,
		Id:          t.Id,
		Seq:         t.Seq,
	}
}

func (t *icmpTuple) Hashable() hashableIcmpTuple {
	var hash hashableIcmpTuple
	copy(hash[0:16], t.SrcIp)
	copy(hash[16:32], t.DstIp)
	copy(hash[32:37], []byte{byte(t.Id >> 8), byte(t.Id), byte(t.Seq >> 8), byte(t.Seq), t.IcmpVersion})
	return hash
}

func (t *icmpTuple) String() string {
	return fmt.Sprintf("icmpTuple version[%d] src[%s] dst[%s] id[%d] seq[%d]",
		t.IcmpVersion, t.SrcIp, t.DstIp, t.Id, t.Seq)
}
