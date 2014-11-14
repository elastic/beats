// +build !linux

package main

import (
	"fmt"
	"time"

	"code.google.com/p/gopacket"
)

type AfpacketHandle struct {
}

func NewAfpacketHandle(devices []string, snaplen int32, block_size int, num_blocks int,
	timeout time.Duration) (*AfpacketHandle, error) {

	return nil, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *AfpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}
