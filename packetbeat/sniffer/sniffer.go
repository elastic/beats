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
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/decoder"
)

// Sniffer provides packet sniffing capabilities, forwarding packets read
// to a Worker.
type Sniffer struct {
	config []config.InterfacesConfig

	state atomic.Int32  // store snifferState
	done  chan struct{} // done is required to wire state into a select.

	// device is the first active device after calling New.
	// It is not updated by default route polling.
	device string

	// followDefault indicates that the sniffer has
	// been configured to follow the default route.
	followDefault bool

	// filter is the bpf filter program used by the sniffer.
	filter string

	decoders Decoders
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
func New(testMode bool, _ string, decoders Decoders, interfaces []config.InterfacesConfig) (*Sniffer, error) {
	s := &Sniffer{
		config:        interfaces,
		decoders:      decoders,
		state:         atomic.MakeInt32(snifferInactive),
		followDefault: interfaces[0].PollDefaultRoute > 0 && strings.HasPrefix(interfaces[0].Device, "default_route"),
	}

	for i, iface := range s.config {
		logp.Debug("sniffer", "interface: %d, BPF filter: '%s'", i, iface.BpfFilter)

		// pre-check and normalize configuration:
		// - resolve potential device name
		// - check for file output
		// - set some defaults
		if iface.File != "" {
			logp.Debug("sniffer", "Reading from file: %s", iface.File)

			if iface.BpfFilter != "" {
				logp.Warn("Packet filters are not applied to pcap files.")
			}

			// we read file with the pcap provider
			s.config[i].Type = "pcap"
			s.config[i].Device = ""
		} else {
			// try to resolve device name (ignore error if testMode is enabled)
			if name, err := resolveDeviceName(iface.Device); err != nil {
				if !testMode {
					return nil, err
				}
			} else {
				s.device = name
				if name == "any" && !deviceAnySupported {
					return nil, fmt.Errorf("any interface is not supported on %s", runtime.GOOS)
				}

				if iface.Snaplen == 0 {
					s.config[i].Snaplen = 65535
				}
				if iface.BufferSizeMb <= 0 {
					s.config[i].BufferSizeMb = 24
				}

				if t := iface.Type; t == "autodetect" || t == "" {
					s.config[i].Type = "pcap"
				}
				logp.Debug("sniffer", "Sniffer type: %s device: %s", iface.Type, s.device)
			}
		}

		err := validateConfig(iface.BpfFilter, &iface) //nolint:gosec // Bad linter! validateConfig completes before the next iteration.
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Run opens the sniffing device and processes packets being read from that device.
// Worker instances are instantiated as needed.
func (s *Sniffer) Run() error {
	var (
		defaultRoute chan string
		refresh      chan struct{}
	)
	if s.followDefault {
		s.done = make(chan struct{})
		defaultRoute = make(chan string)
		refresh = make(chan struct{}, 1)
		go s.pollDefaultRoute(defaultRoute, refresh)
	}
	if defaultRoute == nil {
		return s.sniffStatic(s.device)
	}
	return s.sniffDynamic(defaultRoute, refresh)
}

// pollDefaultRoute repeatedly polls the default route's device at intervals
// specified in config.PollDefaultRoute. The poller is terminated by closing
// done and the device chan can be read for changes in the default route.
// Changes in default route will put the Sniffer into the inactive state to
// trigger a new sniffer connection. Termination of the sniffer is not under
// the control of the poller.
func (s *Sniffer) pollDefaultRoute(device chan<- string, refresh <-chan struct{}) {
	go func() {
		logp.Info("starting default route poller")

		// Prime the channel.
		current := s.device
		device <- current
		defaultRouteMetric.Set(current)

		tick := time.NewTicker(s.config[0].PollDefaultRoute)
		for {
			select {
			case <-tick.C:
				logp.Debug("sniffer", "polling default route")
				current = s.poll(current, device)
			case <-refresh:
				logp.Debug("sniffer", "requested new default route")
				current = s.poll(current, device)
			case <-s.done:
				logp.Info("closing default route poller")
				close(device)
				tick.Stop()
				return
			}
			// Purge any unused refresh request. The chan has a cap
			// of one and the send is conditional so we don't need
			// to do this in a loop.
			select {
			case <-refresh:
			default:
			}
		}
	}()
}

// poll returns the current default route interface and sends it on device
// if it has change from the old default route interface. If device resolution
// fails, the default route interface is left unchanged.
func (s *Sniffer) poll(old string, device chan<- string) (current string) {
	current, err := resolveDeviceName(s.config[0].Device)
	if err != nil {
		logp.Warn("sniffer failed to poll default route device: %v", err)
		return old
	}
	if current != old {
		logp.Info("sniffer changing default route device: %s -> %s", old, current)
		s.state.Store(snifferInactive) // Mark current device as stale. ¯\_(ツ)_/¯
		device <- current              // Pass the new device name.
		defaultRouteMetric.Set(current)
	}
	return current
}

// sniffStatic performs the sniffing work on a single static interface.
func (s *Sniffer) sniffStatic(device string) error {
	handle, err := s.open(device)
	if err != nil {
		return fmt.Errorf("failed to start sniffer: %w", err)
	}
	defer handle.Close()

	dec, err := s.decoders(handle.LinkType())
	if err != nil {
		return err
	}

	return s.sniffHandle(handle, dec, nil)
}

// sniffDynamic performs sniffing work on a stream of dynamic interfaces from
// defaultRoute decoders are retained between successive interfaces if they are
// the same link type.
func (s *Sniffer) sniffDynamic(defaultRoute <-chan string, refresh chan<- struct{}) error {
	var (
		last layers.LinkType
		dec  *decoder.Decoder
	)
	for device := range defaultRoute {
		var err error
		last, dec, err = s.sniffOneDynamic(device, last, dec, refresh)
		if err != nil {
			return err
		}
	}
	return nil
}

// sniffOneDynamic handles sniffing a single device that may change link type.
// If the link type associated with the device differs from the last link
// type or dec is nil, a new decoder is returned. The link type associated
// with the device is returned.
func (s *Sniffer) sniffOneDynamic(device string, last layers.LinkType, dec *decoder.Decoder, refresh chan<- struct{}) (layers.LinkType, *decoder.Decoder, error) {
	handle, err := s.open(device)
	if err != nil {
		return last, dec, fmt.Errorf("failed to start sniffer: %w", err)
	}
	defer handle.Close()

	linkType := handle.LinkType()
	if dec == nil || linkType != last {
		logp.Info("changing link type: %d -> %d", last, linkType)
		dec, err = s.decoders(linkType)
		if err != nil {
			return linkType, dec, err
		}
	}

	err = s.sniffHandle(handle, dec, refresh)
	return linkType, dec, err
}

// sniff performs the sniffing work and writing dump files if requested.
func (s *Sniffer) sniffHandle(handle snifferHandle, dec *decoder.Decoder, refresh chan<- struct{}) error {
	var w *pcapgo.Writer
	if s.config[0].Dumpfile != "" {
		const timeSuffixFormat = "20060102150405"
		filename := fmt.Sprintf("%s-%s.pcap", s.config[0].Dumpfile, time.Now().Format(timeSuffixFormat))
		logp.Info("creating new dump file %s", filename)
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		w = pcapgo.NewWriterNanos(f)
		err = w.WriteFileHeader(65535, handle.LinkType())
		if err != nil {
			return fmt.Errorf("failed to write dump file header to %s: %w", s.config[0].Dumpfile, err)
		}
	}

	// Mark inactive sniffer as active. In case of the sniffer/packetbeat closing
	// before/while Run is executed, the state will be snifferClosing.
	// => return if state is already snifferClosing.
	if !s.state.CAS(snifferInactive, snifferActive) {
		return nil
	}
	defer s.state.Store(snifferInactive)

	var (
		packets  int
		timeouts int
	)
	for s.state.Load() == snifferActive {
		if s.config[0].OneAtATime {
			fmt.Fprintln(os.Stdout, "Press enter to read packet")
			fmt.Scanln()
		}

		data, ci, err := handle.ReadPacketData()
		if err == pcap.NextErrorTimeoutExpired || isAfpacketErrTimeout(err) { //nolint:errorlint // pcap.NextErrorTimeoutExpired is not wrapped.
			logp.Debug("sniffer", "timed out")

			// If we have timed out too many times and we are following
			// a default route, request a new default route interface.
			const maxTimeouts = 10 // Place-holder until we have a sensible notion of how big this should be.
			timeouts++
			if s.followDefault && timeouts > maxTimeouts {
				select {
				case refresh <- struct{}{}:
				default:
					// Don't request to refresh if already requested.
				}
				timeouts = 0
			}
			continue
		}
		timeouts = 0

		if err != nil {
			// ignore EOF, if sniffer was driven from file
			if err == io.EOF && s.config[0].File != "" { //nolint:errorlint // io.EOF should never be wrapped.
				return nil
			}

			// If we are following a default route, request an interface
			// refresh and log the error.
			if s.followDefault {
				select {
				case refresh <- struct{}{}:
				default:
					// Don't request to refresh if already requested.
				}
				logp.Warn("error during packet capture: %v", err)
				continue
			}

			s.state.Store(snifferInactive)
			return fmt.Errorf("sniffing error: %w", err)
		}

		if len(data) == 0 {
			// Empty packet, probably timeout from afpacket.
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
		dec.OnPacket(data, &ci)
	}

	return nil
}

func (s *Sniffer) open(device string) (snifferHandle, error) {
	if s.config[0].File != "" {
		return newFileHandler(s.config[0].File, s.config[0].TopSpeed, s.config[0].Loop)
	}

	switch s.config[0].Type {
	case "pcap":
		return openPcap(device, s.filter, &s.config[0])
	case "af_packet":
		return openAFPacket(device, s.filter, &s.config[0])
	default:
		return nil, fmt.Errorf("unknown sniffer type: %s", s.config[0].Type)
	}
}

// Stop marks a sniffer as stopped. The Run method will return once the stop
// signal has been given.
func (s *Sniffer) Stop() {
	s.state.Store(snifferClosing)
	if s.done != nil {
		close(s.done)
	}
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
		return fmt.Errorf("unknown sniffer type: %s", cfg.Type)
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

func openPcap(device, filter string, cfg *config.InterfacesConfig) (snifferHandle, error) {
	snaplen := int32(cfg.Snaplen)
	timeout := 500 * time.Millisecond
	h, err := pcap.OpenLive(device, snaplen, true, timeout)
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

func openAFPacket(device, filter string, cfg *config.InterfacesConfig) (snifferHandle, error) {
	szFrame, szBlock, numBlocks, err := afpacketComputeSize(cfg.BufferSizeMb, cfg.Snaplen, os.Getpagesize())
	if err != nil {
		return nil, err
	}

	timeout := 500 * time.Millisecond
	h, err := newAfpacketHandle(device, szFrame, szBlock, numBlocks, timeout, cfg.EnableAutoPromiscMode)
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
