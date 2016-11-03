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
	SrcIP       net.IP
	DstIP       net.IP
	ID          uint16
	Seq         uint16
}

func (t *icmpTuple) Reverse() icmpTuple {
	return icmpTuple{
		IcmpVersion: t.IcmpVersion,
		SrcIP:       t.DstIP,
		DstIP:       t.SrcIP,
		ID:          t.ID,
		Seq:         t.Seq,
	}
}

func (t *icmpTuple) Hashable() hashableIcmpTuple {
	var hash hashableIcmpTuple
	copy(hash[0:16], t.SrcIP)
	copy(hash[16:32], t.DstIP)
	copy(hash[32:37], []byte{byte(t.ID >> 8), byte(t.ID), byte(t.Seq >> 8), byte(t.Seq), t.IcmpVersion})
	return hash
}

func (t *icmpTuple) String() string {
	return fmt.Sprintf("icmpTuple version[%d] src[%s] dst[%s] id[%d] seq[%d]",
		t.IcmpVersion, t.SrcIP, t.DstIP, t.ID, t.Seq)
}
