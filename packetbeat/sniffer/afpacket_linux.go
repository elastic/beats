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

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type afpacketHandle struct {
	TPacket                      *afpacket.TPacket
	frameSize                    int
	promiscPreviousState         bool
	promiscPreviousStateDetected bool
	device                       string
	log                          *logp.Logger
	metrics                      *metrics
}

func newAfpacketHandle(c afPacketConfig) (*afpacketHandle, error) {
	var err error
	var promiscEnabled bool
	log := logp.NewLogger("sniffer")

	if c.Promiscuous {
		promiscEnabled, err = isPromiscEnabled(c.Device)
		if err != nil {
			log.Errorf("Failed to get promiscuous mode for device '%s': %v", c.Device, err)
		}

		if !promiscEnabled {
			if setPromiscErr := setPromiscMode(c.Device, true); setPromiscErr != nil {
				log.Warnf("Failed to set promiscuous mode for device '%s'. "+
					"Packetbeat may be unable to see any network traffic. Please follow packetbeat "+
					"FAQ to learn about mitigation: Error: %v", c.Device, err)
			}
		}
	}

	h := &afpacketHandle{
		promiscPreviousState:         promiscEnabled,
		frameSize:                    c.FrameSize,
		device:                       c.Device,
		promiscPreviousStateDetected: c.Promiscuous && err == nil,
		log:                          log,
	}

	if c.Device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(c.FrameSize),
			afpacket.OptBlockSize(c.BlockSize),
			afpacket.OptNumBlocks(c.NumBlocks),
			afpacket.OptPollTimeout(c.PollTimeout))
	} else {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptInterface(c.Device),
			afpacket.OptFrameSize(c.FrameSize),
			afpacket.OptBlockSize(c.BlockSize),
			afpacket.OptNumBlocks(c.NumBlocks),
			afpacket.OptPollTimeout(c.PollTimeout))
	}
	if err != nil {
		return nil, fmt.Errorf("failed creating af_packet socket: %w", err)
	}
	h.metrics = newMetrics(c.ID, c.Device, c.MetricsInterval, h.TPacket, log)

	if c.FanoutGroupID != nil {
		if err = h.TPacket.SetFanout(afpacket.FanoutHashWithDefrag, *c.FanoutGroupID); err != nil {
			return nil, fmt.Errorf("failed setting af_packet fanout group: %w", err)
		}
		log.Infof("Joined af_packet fanout group %v", *c.FanoutGroupID)
	}

	return h, nil
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
	h.metrics.close()

	h.TPacket.Close()
	// previous state detected only if auto mode was on
	if h.promiscPreviousStateDetected {
		if err := setPromiscMode(h.device, h.promiscPreviousState); err != nil {
			h.log.Warnf("Failed to reset promiscuous mode for device '%s'. Your device might be in promiscuous mode.: %v", h.device, err)
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

	copy(ifreq.name[:], device)
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, uintptr(s), syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifreq)))
	if ep != 0 {
		return false, fmt.Errorf("ioctl command SIOCGIFFLAGS failed to get device flags for %v: return code %d", device, ep)
	}

	return ifreq.flags&uint16(syscall.IFF_PROMISC) != 0, nil
}

// setPromiscMode enables promisc mode if configured. This is a no-op when device is 'any'.
func setPromiscMode(device string, enabled bool) error {
	if device == "any" {
		logp.L().Named("sniffer").Warn("Cannot set promiscuous mode for device 'any'")
		return nil
	}

	// SetLsfPromisc is marked as deprecated but used to improve readability (bpf)
	// and avoid Cgo (pcap)
	// TODO: replace with x/net/bpf or pcap
	return syscall.SetLsfPromisc(device, enabled)
}

// isAfpacketErrTimeout returns whether err is afpacket.ErrTimeout.
func isAfpacketErrTimeout(err error) bool {
	return err == afpacket.ErrTimeout
}

type metrics struct {
	unregister func()
	done       chan struct{} // used to signal to polling goroutine to stop

	device             *monitoring.String // name of the device being monitored
	socketPackets      *monitoring.Uint   // number of packets delivered by kernel
	socketDrops        *monitoring.Uint   // number of packets dropped by kernel (i.e., buffer full)
	socketQueueFreezes *monitoring.Uint   // number of queue freezes
	packets            *monitoring.Uint   // number of packets read off buffer by packetbeat
	polls              *monitoring.Uint   // number of blocking syscalls made by packetbeat waiting for packets
}

func (m *metrics) close() {
	if m == nil {
		return
	}
	m.unregister()
	if m.done != nil {
		close(m.done)
		m.done = nil
	}
}

func newMetrics(id, device string, interval time.Duration, handle *afpacket.TPacket, log *logp.Logger) *metrics {
	devID := fmt.Sprintf("%s-af_packet::%s", id, device)
	reg, unreg := inputmon.NewInputRegistry("af_packet", devID, nil)
	out := &metrics{
		unregister:         unreg,
		device:             monitoring.NewString(reg, "device"),
		socketPackets:      monitoring.NewUint(reg, "socket_packets"),
		socketDrops:        monitoring.NewUint(reg, "socket_drops"),
		socketQueueFreezes: monitoring.NewUint(reg, "socket_queue_freezes"),
		packets:            monitoring.NewUint(reg, "packets"),
		polls:              monitoring.NewUint(reg, "polls"),
		done:               make(chan struct{}),
	}

	out.device.Set(device)

	go func() {
		log.Debug("Starting stats collection goroutine, collection interval: %v", interval)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		if err := handle.InitSocketStats(); err != nil {
			log.Errorw("Failed to init socket stats", "error", err)
		}

		for {
			select {
			case <-out.done:
				log.Debug("Shutting down stats collection goroutine")
				return
			case <-ticker.C:
				_, sockStats, err := handle.SocketStats()
				if err != nil {
					log.Debugw("Error getting socket stats", "error", err)
				} else {
					out.socketPackets.Set(uint64(sockStats.Packets()))
					out.socketDrops.Set(uint64(sockStats.Drops()))
					out.socketQueueFreezes.Set(uint64(sockStats.QueueFreezes()))
				}

				stats, err := handle.Stats()
				if err != nil {
					log.Debugw("Error getting packetbeat stats", "error", err)
				} else {
					out.packets.Set(uint64(stats.Packets))
					out.polls.Set(uint64(stats.Polls))
				}
			}
		}
	}()

	return out
}
