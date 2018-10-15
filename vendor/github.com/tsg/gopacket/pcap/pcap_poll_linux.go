// +build linux

package pcap

/*
#include <linux/if_packet.h>
#include <poll.h>
#include <pcap/pcap.h>
*/
import "C"
import "syscall"

// packetPoll holds all the parameters required to use poll(2) on the pcap
// file descriptor.
type packetPoll struct {
	pollfd  C.struct_pollfd
	timeout C.int
}

func captureIsTPacketV3(fildes int) bool {
	version, err := syscall.GetsockoptInt(fildes, syscall.SOL_PACKET, C.PACKET_VERSION)
	return err == nil && version == C.TPACKET_V3
}

// NewPacketPoll returns a new packetPoller if the pcap handle requires it
// in order to timeout effectively when no packets are received. This is only
// necessary when TPACKET_V3 interface is used to receive packets.
func NewPacketPoll(ptr *C.pcap_t, timeout C.int) *packetPoll {
	fildes := C.pcap_fileno(ptr)
	if !captureIsTPacketV3(int(fildes)) {
		return nil
	}
	return &packetPoll{
		pollfd: C.struct_pollfd{
			fd:      fildes,
			events:  C.POLLIN,
			revents: 0,
		},
		timeout: timeout,
	}
}

func (t *packetPoll) AwaitForPackets() bool {
	if t != nil {
		t.pollfd.revents = 0
		// block until the capture file descriptor is readable or a timeout
		// happens.
		n, err := C.poll(&t.pollfd, 1, t.timeout)
		return err != nil || n != 0
	}
	return true
}
