// +build !linux

package sniffer

import (
	"fmt"
	"time"

	"github.com/tsg/gopacket"
)

type AfpacketHandle struct {
}

func NewAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	timeout time.Duration) (*AfpacketHandle, error) {

	return nil, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *AfpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *AfpacketHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *AfpacketHandle) Close() {
}
