package sniffer

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"
	"github.com/tsg/gopacket/pcap"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/config"
)

// Sniffer provides packet sniffing capabilities, forwarding packets read
// to a Worker.
type Sniffer struct {
	config config.InterfacesConfig
	dumper *pcap.Dumper

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
	var (
		counter = 0
		dumper  *pcap.Dumper
	)

	handle, err := s.open()
	if err != nil {
		return fmt.Errorf("Error starting sniffer: %s", err)
	}
	defer handle.Close()

	if s.config.Dumpfile != "" {
		dumper, err = openDumper(s.config.Dumpfile, handle.LinkType())
		if err != nil {
			return err
		}

		defer dumper.Close()
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

	for s.state.Load() == snifferActive {
		if s.config.OneAtATime {
			fmt.Println("Press enter to read packet")
			fmt.Scanln()
		}

		data, ci, err := handle.ReadPacketData()
		if err == pcap.NextErrorTimeoutExpired || err == syscall.EINTR {
			logp.Debug("sniffer", "Interrupted")
			continue
		}

		if err != nil {
			// ignore EOF, if sniffer was driven from file
			if err == io.EOF && s.config.File != "" {
				return nil
			}

			s.state.Store(snifferInactive)
			return fmt.Errorf("Sniffing error: %s", err)
		}

		if len(data) == 0 {
			// Empty packet, probably timeout from afpacket
			continue
		}

		if dumper != nil {
			dumper.WritePacketData(data, ci)
		}

		counter++
		logp.Debug("sniffer", "Packet number: %d", counter)
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

	// Open a dummy pcap handle to compile the filter
	p, err := pcap.OpenDead(layers.LinkTypeEthernet, 65535)
	if err != nil {
		return fmt.Errorf("OpenDead: %s", err)
	}

	defer p.Close()

	_, err = p.NewBPF(expr)
	if err != nil {
		return fmt.Errorf("invalid filter '%s': %v", expr, err)
	}
	return nil
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
	h, err := newAfpacketHandle(cfg.Device, szFrame, szBlock, numBlocks, timeout)
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

func openDumper(file string, linkType layers.LinkType) (*pcap.Dumper, error) {
	p, err := pcap.OpenDead(linkType, 65535)
	if err != nil {
		return nil, err
	}

	return p.NewDumper(file)
}
