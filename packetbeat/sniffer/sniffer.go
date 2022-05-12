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

package sniffer

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/config"
)

// Sniffer provides packet sniffing capabilities, forwarding packets read
// to a Worker.
type Sniffer struct {
	config config.InterfacesConfig

	state atomic.Int32 // store snifferState

	// bpf filter
	filter string

	factory WorkerFactory
}

// WorkerFactory constructs a new worker instance for use with a Sniffer.
type WorkerFactory func(layers.LinkType) (Worker, error)

// Worker defines the callback interfaces a Sniffer instance will use
// to forward packets.
type Worker interface {
	OnPacket(data []byte, ci *gopacket.CaptureInfo)
}

type snifferHandle interface {
	gopacket.PacketDataSource
	LinkType() layers.LinkType
	Close()
}

// sniffer state values
const (
	snifferInactive = 0
	snifferClosing  = 1
	snifferActive   = 2
)

// New create a new Sniffer instance. Settings are validated in a best effort
// only, but no device is opened yet. Accessing and configuring the actual device
// is done by the Run method.
func New(
	testMode bool,
	filter string,
	factory WorkerFactory,
	interfaces config.InterfacesConfig,
) (*Sniffer, error) {
	s := &Sniffer{
		filter:  filter,
		config:  interfaces,
		factory: factory,
		state:   atomic.MakeInt32(snifferInactive),
	}

	logp.Debug("sniffer", "BPF filter: '%s'", filter)

	// pre-check and normalize configuration:
	// - resolve potential device name
	// - check for file output
	// - set some defaults
	if s.config.File != "" {
		logp.Debug("sniffer", "Reading from file: %s", s.config.File)

		if s.config.BpfFilter != "" {
			logp.Warn("Packet filters are not applied to pcap files.")
		}

		// we read file with the pcap provider
		s.config.Type = "pcap"
		s.config.Device = ""
	} else {
		// try to resolve device name (ignore error if testMode is enabled)
		if name, err := resolveDeviceName(s.config.Device); err != nil {
			if !testMode {
				return nil, err
			}
		} else {
			s.config.Device = name
			if name == "any" && !deviceAnySupported {
				return nil, fmt.Errorf("any interface is not supported on %s", runtime.GOOS)
			}

			if s.config.Snaplen == 0 {
				s.config.Snaplen = 65535
			}
			if s.config.BufferSizeMb <= 0 {
				s.config.BufferSizeMb = 24
			}

			if t := s.config.Type; t == "autodetect" || t == "" {
				s.config.Type = "pcap"
			}
			logp.Debug("sniffer", "Sniffer type: %s device: %s", s.config.Type, s.config.Device)
		}
	}

	err := validateConfig(filter, &s.config)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Run opens the sniffing device and processes packets being read from that device.
// Worker instances are instantiated as needed.
func (s *Sniffer) Run() error {
	handle, err := s.open()
	if err != nil {
		return fmt.Errorf("Error starting sniffer: %s", err)
	}
	defer handle.Close()

	var w *pcapgo.Writer
	if s.config.Dumpfile != "" {
		f, err := os.Create(s.config.Dumpfile)
		if err != nil {
			return err
		}
		defer f.Close()

		w = pcapgo.NewWriterNanos(f)
		w.WriteFileHeader(65535, handle.LinkType())
	}

	worker, err := s.factory(handle.LinkType())
	if err != nil {
		return err
	}

	// Mark inactive sniffer as active. In case of the sniffer/packetbeat closing
	// before/while Run is executed, the state will be snifferClosing.
	// => return if state is already snifferClosing.
	if !s.state.CAS(snifferInactive, snifferActive) {
		return nil
	}
	defer s.state.Store(snifferInactive)

	var packets int
	for s.state.Load() == snifferActive {
		if s.config.OneAtATime {
			fmt.Println("Press enter to read packet")
			fmt.Scanln()
		}

		data, ci, err := handle.ReadPacketData()
		if err == pcap.NextErrorTimeoutExpired || isAfpacketErrTimeout(err) {
			logp.Debug("sniffer", "timedout")
			continue
		}

		if err != nil {
			// ignore EOF, if sniffer was driven from file
			if err == io.EOF && s.config.File != "" {
				return nil
			}

			s.state.Store(snifferInactive)
			return fmt.Errorf("Sniffing error: %w", err)
		}

		if len(data) == 0 {
			// Empty packet, probably timeout from afpacket
			continue
		}

		packets++

		if w != nil {
			err = w.WritePacket(ci, data)
			if err != nil {
				return fmt.Errorf("failed to write packet %d: %w", packets, err)
			}
		}

		logp.Debug("sniffer", "Packet number: %d", packets)
		worker.OnPacket(data, &ci)
	}

	return nil
}

func (s *Sniffer) open() (snifferHandle, error) {
	if s.config.File != "" {
		return newFileHandler(s.config.File, s.config.TopSpeed, s.config.Loop)
	}

	switch s.config.Type {
	case "pcap":
		return openPcap(s.filter, &s.config)
	case "af_packet":
		return openAFPacket(s.filter, &s.config)
	default:
		return nil, fmt.Errorf("Unknown sniffer type: %s", s.config.Type)
	}
}

// Stop marks a sniffer as stopped. The Run method will return once the stop
// signal has been given.
func (s *Sniffer) Stop() error {
	s.state.Store(snifferClosing)
	return nil
}

func validateConfig(filter string, cfg *config.InterfacesConfig) error {
	if cfg.File == "" {
		if err := validatePcapFilter(filter); err != nil {
			return err
		}
	}

	switch cfg.Type {
	case "pcap":
		return validatePcapConfig(cfg)
	case "af_packet":
		return validateAfPacketConfig(cfg)
	default:
		return fmt.Errorf("Unknown sniffer type: %s", cfg.Type)
	}
}

func validatePcapConfig(cfg *config.InterfacesConfig) error {
	return nil
}

func validateAfPacketConfig(cfg *config.InterfacesConfig) error {
	_, _, _, err := afpacketComputeSize(cfg.BufferSizeMb, cfg.Snaplen, os.Getpagesize())
	return err
}

func validatePcapFilter(expr string) error {
	if expr == "" {
		return nil
	}
	_, err := pcap.NewBPF(layers.LinkTypeEthernet, 65535, expr)
	return err
}

func openPcap(filter string, cfg *config.InterfacesConfig) (snifferHandle, error) {
	snaplen := int32(cfg.Snaplen)
	timeout := 500 * time.Millisecond
	h, err := pcap.OpenLive(cfg.Device, snaplen, true, timeout)
	if err != nil {
		return nil, err
	}

	err = h.SetBPFFilter(filter)
	if err != nil {
		h.Close()
		return nil, err
	}

	return h, nil
}

func openAFPacket(filter string, cfg *config.InterfacesConfig) (snifferHandle, error) {
	szFrame, szBlock, numBlocks, err := afpacketComputeSize(cfg.BufferSizeMb, cfg.Snaplen, os.Getpagesize())
	if err != nil {
		return nil, err
	}

	timeout := 500 * time.Millisecond
	h, err := newAfpacketHandle(cfg.Device, szFrame, szBlock, numBlocks, timeout, cfg.EnableAutoPromiscMode)
	if err != nil {
		return nil, err
	}

	err = h.SetBPFFilter(filter)
	if err != nil {
		h.Close()
		return nil, err
	}

	return h, nil
}
