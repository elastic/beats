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
	icmpVersion uint8
	srcIP       net.IP
	dstIP       net.IP
	id          uint16
	seq         uint16
}

func (t *icmpTuple) Reverse() icmpTuple {
	return icmpTuple{
		icmpVersion: t.icmpVersion,
		srcIP:       t.dstIP,
		dstIP:       t.srcIP,
		id:          t.id,
		seq:         t.seq,
	}
}

func (t *icmpTuple) Hashable() hashableIcmpTuple {
	var hash hashableIcmpTuple
	copy(hash[0:16], t.srcIP)
	copy(hash[16:32], t.dstIP)
	copy(hash[32:37], []byte{byte(t.id >> 8), byte(t.id), byte(t.seq >> 8), byte(t.seq), t.icmpVersion})
	return hash
}

func (t *icmpTuple) String() string {
	return fmt.Sprintf("icmpTuple version[%d] src[%s] dst[%s] id[%d] seq[%d]",
		t.icmpVersion, t.srcIP, t.dstIP, t.id, t.seq)
}
