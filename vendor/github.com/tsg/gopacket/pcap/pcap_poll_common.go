// +build !linux

package pcap

/*
#include <pcap/pcap.h>
*/
import "C"

type packetPoll struct{}

func NewPacketPoll(_ *C.pcap_t, _ C.int) *packetPoll {
	return nil
}

func (t *packetPoll) AwaitForPackets() bool {
	return true
}
