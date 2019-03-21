// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build linux

package sniffer

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/afpacket"
	"github.com/tsg/gopacket/layers"
)

type afpacketHandle struct {
	TPacket              *afpacket.TPacket
	promicsPreviousState bool
	device               string
}

func newAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	timeout time.Duration) (*afpacketHandle, error) {

	promiscEnabled, err := isPromiscEnabled(device)
	if err != nil {
		logp.Err("Failed to get promiscuous mode for device '%s': %v", device, err)
	}

	h := &afpacketHandle{
		promicsPreviousState: promiscEnabled,
		device:               device,
	}

	if err := setPromiscMode(device, true); err != nil {
		logp.Err("Failed to set promiscuous mode for device '%s': %v", device, err)
	}

	if device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	} else {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptInterface(device),
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	}

	return h, err
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ReadPacketData()
}

func (h *afpacketHandle) SetBPFFilter(expr string) (_ error) {
	return h.TPacket.SetBPFFilter(expr)
}

func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *afpacketHandle) Close() {
	h.TPacket.Close()
	if err := setPromiscMode(h.device, h.promicsPreviousState); err != nil {
		logp.Err("Failed to set promiscuous mode for device '%s': %v", h.device, err)
	}
}

func isPromiscEnabled(device string) (bool, error) {
	if device == "any" {
		return false, nil
	}

	s, e := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if e != nil {
		return false, e
	}

	defer syscall.Close(s)

	var ifl struct {
		name  [syscall.IFNAMSIZ]byte
		flags uint16
	}

	copy(ifl.name[:], []byte(device))
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifl)))
	if ep != 0 {
		return false, fmt.Errorf("Syscall SIOCGIFFLAGS exited with %v", ep)
	}

	return ifl.flags&uint16(syscall.IFF_PROMISC) != 0, nil
}

func setPromiscMode(device string, enabled bool) error {
	if device == "any" {
		logp.Warn("Cannot set promiscuous mode to device 'any'")
		return nil
	}

	return syscall.SetLsfPromisc(device, enabled)
}
