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
	"context"
	"encoding/json"
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
	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/decoder"
)

// Sniffer provides packet sniffing capabilities, forwarding packets read
// to a Worker.
type Sniffer struct {
	sniffers []sniffer
	cancel   func()
	log      *logp.Logger
}

type sniffer struct {
	config config.InterfaceConfig

	state atomic.Int32 // store snifferState

	// device is the first active device after calling New.
	// It is not updated by default route polling.
	device string

	// followDefault indicates that the sniffer has
	// been configured to follow the default route.
	followDefault bool

	// filter is the bpf filter program used by the sniffer.
	filter string

	// id and idx identify the sniffer for metric collection.
	id  string
	idx int

	decoders Decoders

	log *logp.Logger
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
// is done by the Run method. The id parameter is used to specify the metric
// collection ID for AF_PACKET sniffers on Linux.
func New(id string, testMode bool, _ string, decoders map[string]Decoders, interfaces []config.InterfaceConfig) (*Sniffer, error) {
	s := &Sniffer{
		sniffers: make([]sniffer, len(interfaces)),
		log:      logp.NewLogger("sniffer"),
	}

	for i, iface := range interfaces {
		dec, ok := decoders[iface.Device]
		if !ok {
			// This should never happen.
			return nil, fmt.Errorf("no decoder for %s", iface.Device)
		}
		child := sniffer{
			state:         atomic.MakeInt32(snifferInactive),
			followDefault: iface.PollDefaultRoute > 0 && strings.HasPrefix(iface.Device, "default_route"),
			id:            id,
			idx:           i,
			decoders:      dec,
			log:           s.log,
		}

		s.log.Debugf("interface: %d, BPF filter: '%s'", i, iface.BpfFilter)

		// pre-check and normalize configuration:
		// - resolve potential device name
		// - check for file output
		// - set some defaults
		if iface.File != "" {
			s.log.Debugf("Reading from file: %s", iface.File)

			if iface.BpfFilter != "" {
				s.log.Warn("Packet filters are not applied to pcap files. Ignoring BFP filter.")
			}

			// we read file with the pcap provider
			iface.Type = "pcap"
			iface.Device = ""
		} else {
			// try to resolve device name (ignore error if testMode is enabled)
			if name, err := resolveDeviceName(iface.Device); err != nil {
				if !testMode {
					return nil, err
				}
			} else {
				child.device = name
				if name == "any" && !deviceAnySupported {
					return nil, fmt.Errorf("any interface is not supported on %s", runtime.GOOS)
				}

				if iface.Snaplen == 0 {
					iface.Snaplen = 65535
				}
				if iface.BufferSizeMb <= 0 {
					iface.BufferSizeMb = 24
				}
				if iface.MetricsInterval <= 0 {
					iface.MetricsInterval = 5 * time.Second
				}

				if t := iface.Type; t == "autodetect" || t == "" {
					iface.Type = "pcap"
				}
				s.log.Debugf("Sniffer type: %s device: %s", iface.Type, child.device)
			}
		}

		err := validateConfig(iface.BpfFilter, &iface) //nolint:gosec // Bad linter! validateConfig completes before the next iteration.
		if err != nil {
			cfg, _ := json.Marshal(iface)
			return nil, fmt.Errorf("validate: %w: %s", err, cfg)
		}

		child.config = iface
		child.filter = iface.BpfFilter
		s.sniffers[i] = child
	}

	return s, nil
}

func validateConfig(filter string, cfg *config.InterfaceConfig) error {
	if cfg.File == "" {
		if err := validatePcapFilter(filter); err != nil {
			return err
		}
	}

	switch cfg.Type {
	case "pcap":
		return nil
	case "af_packet":
		return validateAfPacketConfig(cfg)
	default:
		return fmt.Errorf("unknown sniffer type for %s: %q", cfg.Device, cfg.Type)
	}
}

func validatePcapFilter(expr string) error {
	if expr == "" {
		return nil
	}
	_, err := pcap.NewBPF(layers.LinkTypeEthernet, 65535, expr)
	return err
}

func validateAfPacketConfig(cfg *config.InterfaceConfig) error {
	_, _, _, err := afpacketComputeSize(cfg.BufferSizeMb, cfg.Snaplen, os.Getpagesize())
	return err
}

// Run opens the sniffing device and processes packets being read from that device.
// Worker instances are instantiated as needed.
func (s *Sniffer) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	g, ctx := errgroup.WithContext(ctx)
	for i := range s.sniffers {
		c := &s.sniffers[i]
		g.Go(func() error {
			var (
				defaultRoute chan string
				refresh      chan struct{}
			)
			if c.followDefault {
				defaultRoute = make(chan string)
				refresh = make(chan struct{}, 1)
				go c.pollDefaultRoute(ctx, defaultRoute, refresh)
			}
			if defaultRoute == nil {
				return c.sniffStatic(ctx, c.device)
			}
			return c.sniffDynamic(ctx, defaultRoute, refresh)
		})
	}
	return g.Wait()
}

// pollDefaultRoute repeatedly polls the default route's device at intervals
// specified in config.PollDefaultRoute. The poller is terminated by cancelling
// the context and the device chan can be read for changes in the default route.
// Changes in default route will put the Sniffer into the inactive state to
// trigger a new sniffer connection. Termination of the sniffer is not under
// the control of the poller.
func (s *sniffer) pollDefaultRoute(ctx context.Context, device chan<- string, refresh <-chan struct{}) {
	go func() {
		s.log.Info("starting default route poller")

		// Prime the channel.
		current := s.device
		device <- current
		defaultRouteMetric.Set(current)

		tick := time.NewTicker(s.config.PollDefaultRoute)
		for {
			select {
			case <-tick.C:
				s.log.Debug("polling default route")
				current = s.poll(current, device)
			case <-refresh:
				s.log.Debug("requested new default route")
				current = s.poll(current, device)
			case <-ctx.Done():
				s.log.Info("closing default route poller")
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
// if it has a change from the old default route interface. If device resolution
// fails, the default route interface is left unchanged.
func (s *sniffer) poll(old string, device chan<- string) (current string) {
	current, err := resolveDeviceName(s.config.Device)
	if err != nil {
		s.log.Warnf("sniffer failed to poll default route device: %v", err)
		return old
	}
	if current != old {
		s.log.Infof("sniffer changing default route device: %s -> %s", old, current)
		s.state.Store(snifferInactive) // Mark current device as stale. ¯\_(ツ)_/¯
		device <- current              // Pass the new device name.
		defaultRouteMetric.Set(current)
	}
	return current
}

// sniffStatic performs the sniffing work on a single static interface.
func (s *sniffer) sniffStatic(ctx context.Context, device string) error {
	handle, err := s.open(device)
	if err != nil {
		return fmt.Errorf("failed to start sniffer: %w", err)
	}
	defer handle.Close()

	dec, cleanup, err := s.decoders(handle.LinkType(), device, s.idx)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	return s.sniffHandle(ctx, handle, dec, nil)
}

// sniffDynamic performs sniffing work on a stream of dynamic interfaces from
// defaultRoute decoders are retained between successive interfaces if they are
// the same link type.
func (s *sniffer) sniffDynamic(ctx context.Context, defaultRoute <-chan string, refresh chan<- struct{}) error {
	var (
		last layers.LinkType
		dec  *decoder.Decoder
	)
	for device := range defaultRoute {
		var err error
		last, dec, err = s.sniffOneDynamic(ctx, device, last, dec, refresh)
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
func (s *sniffer) sniffOneDynamic(ctx context.Context, device string, last layers.LinkType, dec *decoder.Decoder, refresh chan<- struct{}) (layers.LinkType, *decoder.Decoder, error) {
	handle, err := s.open(device)
	if err != nil {
		return last, dec, fmt.Errorf("failed to start sniffer: %w", err)
	}
	defer handle.Close()

	linkType := handle.LinkType()
	if dec == nil || linkType != last {
		s.log.Infof("changing link type: %d -> %d", last, linkType)
		var cleanup func()
		dec, cleanup, err = s.decoders(linkType, device, s.idx)
		if err != nil {
			return linkType, dec, err
		}
		if cleanup != nil {
			defer cleanup()
		}
	}

	err = s.sniffHandle(ctx, handle, dec, refresh)
	return linkType, dec, err
}

// sniff performs the sniffing work and writing dump files if requested.
func (s *sniffer) sniffHandle(ctx context.Context, handle snifferHandle, dec *decoder.Decoder, refresh chan<- struct{}) error {
	var w *pcapgo.Writer
	if s.config.Dumpfile != "" {
		const timeSuffixFormat = "20060102150405"
		filename := fmt.Sprintf("%s-%s.pcap", s.config.Dumpfile, time.Now().Format(timeSuffixFormat))
		s.log.Infof("creating new dump file %s", filename)
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		w = pcapgo.NewWriterNanos(f)
		err = w.WriteFileHeader(65535, handle.LinkType())
		if err != nil {
			return fmt.Errorf("failed to write dump file header to %s: %w", s.config.Dumpfile, err)
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
		select {
		case <-ctx.Done():
			s.log.Infof("sniffing cancelled: %q", s.config.Device)

			// Return nil since this must have been due to an errgroup
			// termination and any error that caused that will already
			// have been captured by the errgroup.
			return nil
		default:
		}

		if s.config.OneAtATime {
			fmt.Fprintln(os.Stdout, "Press enter to read packet")
			fmt.Scanln()
		}

		data, ci, err := handle.ReadPacketData()
		if err == pcap.NextErrorTimeoutExpired || isAfpacketErrTimeout(err) { //nolint:errorlint // pcap.NextErrorTimeoutExpired is not wrapped.
			// If we have timed out too many times, and we are following
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
			if err == io.EOF && s.config.File != "" { //nolint:errorlint // io.EOF should never be wrapped.
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
				s.log.Warnf("error during packet capture: %v", err)
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

		if s.config.OneAtATime {
			s.log.Debugw("Packet received.", "network.packets", packets)
		}
		dec.OnPacket(data, &ci)
	}

	return nil
}

func (s *sniffer) open(device string) (snifferHandle, error) {
	if s.config.File != "" {
		return newFileHandler(s.config.File, s.config.TopSpeed, s.config.Loop)
	}

	switch s.config.Type {
	case "pcap":
		return openPcap(device, s.filter, &s.config)
	case "af_packet":
		return openAFPacket(fmt.Sprintf("%s_%d", s.id, s.idx), device, s.filter, &s.config)
	default:
		return nil, fmt.Errorf("unknown sniffer type for %s: %q", device, s.config.Type)
	}
}

// Stop marks a sniffer as stopped. The Run method will return once the stop
// signal has been given.
func (s *Sniffer) Stop() {
	s.log.Debug("sending stop to all sniffers")
	for _, c := range s.sniffers {
		s.log.Debugf("sending closing to %s", c.config.Device)
		c.state.Store(snifferClosing)
	}
	if s.cancel != nil {
		s.log.Debug("cancelling sniffers")
		s.cancel()
	}
}

func openPcap(device, filter string, cfg *config.InterfaceConfig) (snifferHandle, error) {
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

func openAFPacket(id, device, filter string, cfg *config.InterfaceConfig) (snifferHandle, error) {
	szFrame, szBlock, numBlocks, err := afpacketComputeSize(cfg.BufferSizeMb, cfg.Snaplen, os.Getpagesize())
	if err != nil {
		return nil, err
	}

	timeout := 500 * time.Millisecond
	h, err := newAfpacketHandle(afPacketConfig{
		ID:              id,
		Device:          device,
		FrameSize:       szFrame,
		BlockSize:       szBlock,
		NumBlocks:       numBlocks,
		PollTimeout:     timeout,
		MetricsInterval: cfg.MetricsInterval,
		FanoutGroupID:   cfg.FanoutGroup,
		Promiscuous:     cfg.EnableAutoPromiscMode,
	})
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
