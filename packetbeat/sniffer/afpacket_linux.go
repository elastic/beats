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

//go:build linux
// +build linux

package sniffer

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/bpf"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type afpacketHandle struct {
	TPacket                      *afpacket.TPacket
	frameSize                    int
	promiscPreviousState         bool
	promiscPreviousStateDetected bool
	device                       string
}

func newAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	timeout time.Duration, autoPromiscMode bool) (*afpacketHandle, error,
) {
	var err error
	var promiscEnabled bool

	if autoPromiscMode {
		promiscEnabled, err = isPromiscEnabled(device)
		if err != nil {
			logp.Err("Failed to get promiscuous mode for device '%s': %v", device, err)
		}

		if !promiscEnabled {
			if setPromiscErr := setPromiscMode(device, true); setPromiscErr != nil {
				logp.Warn("Failed to set promiscuous mode for device '%s'. Packetbeat may be unable to see any network traffic. Please follow packetbeat FAQ to learn about mitigation: Error: %v", device, err)
			}
		}
	}

	h := &afpacketHandle{
		promiscPreviousState:         promiscEnabled,
		frameSize:                    snaplen,
		device:                       device,
		promiscPreviousStateDetected: autoPromiscMode && err == nil,
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

func (h *afpacketHandle) SetBPFFilter(expr string) error {
	prog, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, h.frameSize, expr)
	if err != nil {
		return err
	}
	p := make([]bpf.RawInstruction, len(prog))
	for i, ins := range prog {
		p[i] = bpf.RawInstruction{
			Op: ins.Code,
			Jt: ins.Jt,
			Jf: ins.Jf,
			K:  ins.K,
		}
	}
	return h.TPacket.SetBPF(p)
}

func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *afpacketHandle) Close() {
	h.TPacket.Close()
	// previous state detected only if auto mode was on
	if h.promiscPreviousStateDetected {
		if err := setPromiscMode(h.device, h.promiscPreviousState); err != nil {
			logp.Warn("Failed to reset promiscuous mode for device '%s'. Your device might be in promiscuous mode.: %v", h.device, err)
		}
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

	var ifreq struct {
		name  [syscall.IFNAMSIZ]byte
		flags uint16
	}

	copy(ifreq.name[:], []byte(device))
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifreq)))
	if ep != 0 {
		return false, fmt.Errorf("ioctl command SIOCGIFFLAGS failed to get device flags for %v: return code %d", device, ep)
	}

	return ifreq.flags&uint16(syscall.IFF_PROMISC) != 0, nil
}

// setPromiscMode enables promisc mode if configured.
// this makes maintenance for user simpler without any additional manual steps
// issue [700](https://github.com/elastic/beats/issues/700)
func setPromiscMode(device string, enabled bool) error {
	if device == "any" {
		logp.Warn("Cannot set promiscuous mode to device 'any'")
		return nil
	}

	// SetLsfPromisc is marked as deprecated but used to improve readability (bpf)
	// and avoid Cgo (pcap)
	// TODO: replace with x/net/bpf or pcap
	return syscall.SetLsfPromisc(device, enabled)
}
