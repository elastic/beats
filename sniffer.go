package main

import (
	"fmt"
	"os"
	"time"

	"github.com/packetbeat/gopacket"
	"github.com/packetbeat/gopacket/layers"
	"github.com/packetbeat/gopacket/pcap"
)

type SnifferSetup struct {
	pcapHandle     *pcap.Handle
	afpacketHandle *AfpacketHandle
	pfringHandle   *PfringHandle
	config         *tomlInterfaces

	DataSource gopacket.PacketDataSource
}

type tomlInterfaces struct {
	Device         string
	Devices        []string
	Type           string
	File           string
	With_vlans     bool
	Bpf_filter     string
	Snaplen        int
	Buffer_size_mb int
}

// Computes the block_size and the num_blocks in such a way that the
// allocated mmap buffer is close to but smaller than target_size_mb.
// The restriction is that the block_size must be divisible by both the
// frame size and page size.
func afpacketComputeSize(target_size_mb int, snaplen int, page_size int) (
	frame_size int, block_size int, num_blocks int, err error) {

	if snaplen < page_size {
		frame_size = page_size / (page_size / snaplen)
	} else {
		frame_size = (snaplen/page_size + 1) * page_size
	}

	// 128 is the default from the gopacket library so just use that
	block_size = frame_size * 128
	num_blocks = (target_size_mb * 1024 * 1024) / block_size

	if num_blocks == 0 {
		return 0, 0, 0, fmt.Errorf("Buffer size too small")
	}

	return frame_size, block_size, num_blocks, nil
}

func CreateSniffer(config *tomlInterfaces, file *string) (*SnifferSetup, error) {
	var sniffer SnifferSetup
	var err error

	sniffer.config = config

	if file != nil && len(*file) > 0 {
		DEBUG("sniffer", "Reading from file: %s", *file)
		// we read file with the pcap provider
		sniffer.config.Type = "pcap"
		sniffer.config.File = *file
	}

	// set defaults
	if len(sniffer.config.Device) == 0 {
		sniffer.config.Device = "any"
	}

	if len(sniffer.config.Devices) == 0 {
		// 'devices' not set but 'device' is set. For backwards compatibility,
		// use the one configured device
		if len(sniffer.config.Device) > 0 {
			sniffer.config.Devices = []string{sniffer.config.Device}
		}
	}
	if sniffer.config.Snaplen == 0 {
		if sniffer.config.Devices[0] == "any" || sniffer.config.Devices[0] == "lo" {
			sniffer.config.Snaplen = 16436
		} else {
			sniffer.config.Snaplen = 1514
		}
	}

	if sniffer.config.Type == "autodetect" || sniffer.config.Type == "" {
		sniffer.config.Type = "pcap"
	}

	DEBUG("sniffer", "Sniffer type: %s devices: %s", sniffer.config.Type, sniffer.config.Devices)

	switch sniffer.config.Type {
	case "pcap":
		if len(sniffer.config.File) > 0 {
			sniffer.pcapHandle, err = pcap.OpenOffline(sniffer.config.File)
			if err != nil {
				return nil, err
			}
		} else {
			if len(sniffer.config.Devices) > 1 {
				return nil, fmt.Errorf("Pcap sniffer only supports one device. You can use 'any' if you want")
			}
			sniffer.pcapHandle, err = pcap.OpenLive(
				sniffer.config.Devices[0],
				int32(sniffer.config.Snaplen),
				true,
				500*time.Millisecond)
			if err != nil {
				return nil, err
			}
			err = sniffer.pcapHandle.SetBPFFilter(sniffer.config.Bpf_filter)
			if err != nil {
				return nil, err
			}
		}

		sniffer.DataSource = gopacket.PacketDataSource(sniffer.pcapHandle)

	case "af_packet":
		if sniffer.config.Buffer_size_mb == 0 {
			sniffer.config.Buffer_size_mb = 24
		}

		if len(sniffer.config.Devices) > 1 {
			return nil, fmt.Errorf("Afpacket sniffer only supports one device. You can use 'any' if you want")
		}

		frame_size, block_size, num_blocks, err := afpacketComputeSize(
			sniffer.config.Buffer_size_mb,
			sniffer.config.Snaplen,
			os.Getpagesize())
		if err != nil {
			return nil, err
		}

		sniffer.afpacketHandle, err = NewAfpacketHandle(
			sniffer.config.Devices[0],
			frame_size,
			block_size,
			num_blocks,
			500*time.Millisecond)
		if err != nil {
			return nil, err
		}

		err = sniffer.afpacketHandle.SetBPFFilter(sniffer.config.Bpf_filter)
		if err != nil {
			return nil, fmt.Errorf("SetBPFFilter failed: %s", err)
		}

		sniffer.DataSource = gopacket.PacketDataSource(sniffer.afpacketHandle)
	case "pfring":
		if len(sniffer.config.Devices) > 1 {
			return nil, fmt.Errorf("Afpacket sniffer only supports one device. You can use 'any' if you want")
		}

		sniffer.pfringHandle, err = NewPfringHandle(
			sniffer.config.Devices[0],
			sniffer.config.Snaplen,
			true)

		if err != nil {
			return nil, err
		}

		err = sniffer.pfringHandle.SetBPFFilter(sniffer.config.Bpf_filter)
		if err != nil {
			return nil, fmt.Errorf("SetBPFFilter failed: %s", err)
		}

		err = sniffer.pfringHandle.Enable()
		if err != nil {
			return nil, fmt.Errorf("Enable failed: %s", err)
		}

		sniffer.DataSource = gopacket.PacketDataSource(sniffer.pfringHandle)

	default:
		return nil, fmt.Errorf("Unknown sniffer type: %s", sniffer.config.Type)
	}

	return &sniffer, nil
}

func (sniffer *SnifferSetup) Reopen() error {
	var err error

	if sniffer.config.Type != "pcap" || sniffer.config.File == "" {
		return fmt.Errorf("Reopen is only possible for files")
	}

	sniffer.pcapHandle.Close()
	sniffer.pcapHandle, err = pcap.OpenOffline(sniffer.config.File)
	if err != nil {
		return err
	}

	sniffer.DataSource = gopacket.PacketDataSource(sniffer.pcapHandle)

	return nil
}

func (sniffer *SnifferSetup) Close() {
	switch sniffer.config.Type {
	case "pcap":
		sniffer.pcapHandle.Close()
	case "af_packet":
		sniffer.afpacketHandle.Close()
	case "pfring":
		sniffer.pfringHandle.Close()
	}
}

func (sniffer *SnifferSetup) Datalink() layers.LinkType {
	if sniffer.config.Type == "pcap" {
		return sniffer.pcapHandle.LinkType()
	}
	return layers.LinkTypeEthernet
}
