// +build linux

package main

import (
	"time"

	"code.google.com/p/gopacket"
	"code.google.com/p/gopacket/afpacket"
)

type AfpacketHandle struct {
	TPacket *afpacket.TPacket
}

func NewAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	timeout time.Duration) (*AfpacketHandle, error) {

	var h AfpacketHandle
	var err error

	if device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptBlockTimeout(timeout))
	}

	h.TPacket, err = afpacket.NewTPacket(
		afpacket.OptInterface(device),
		afpacket.OptFrameSize(snaplen),
		afpacket.OptBlockSize(block_size),
		afpacket.OptNumBlocks(num_blocks),
		afpacket.OptBlockTimeout(timeout))

	return &h, err
}

func (h *AfpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ReadPacketData()
}
