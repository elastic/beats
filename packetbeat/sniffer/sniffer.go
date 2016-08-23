package sniffer

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/config"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/layers"
	"github.com/tsg/gopacket/pcap"
)

type SnifferSetup struct {
	pcapHandle     *pcap.Handle
	afpacketHandle *AfpacketHandle
	pfringHandle   *PfringHandle
	config         *config.InterfacesConfig
	isAlive        bool
	dumper         *pcap.Dumper

	// bpf filter
	filter string

	// Decoder    *decoder.DecoderStruct
	worker     Worker
	DataSource gopacket.PacketDataSource
}

type Worker interface {
	OnPacket(data []byte, ci *gopacket.CaptureInfo)
}

type WorkerFactory func(layers.LinkType) (Worker, string, error)

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

func deviceNameFromIndex(index int, devices []string) (string, error) {
	if index >= len(devices) {
		return "", fmt.Errorf("Looking for device index %d, but there are only %d devices",
			index, len(devices))
	}

	return devices[index], nil
}

// ListDevicesNames returns the list of adapters available for sniffing on
// this computer. If the withDescription parameter is set to true, a human
// readable version of the adapter name is added. If the withIP parameter
// is set to true, IP address of the adatper is added.
func ListDeviceNames(withDescription bool, withIP bool) ([]string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return []string{}, err
	}

	ret := []string{}
	for _, dev := range devices {
		r := dev.Name

		if withDescription {
			desc := "No description available"
			if len(dev.Description) > 0 {
				desc = dev.Description
			}
			r += fmt.Sprintf(" (%s)", desc)
		}

		if withIP {
			ips := "Not assigned ip address"
			if len(dev.Addresses) > 0 {
				ips = ""

				for i, address := range []pcap.InterfaceAddress(dev.Addresses) {
					// Add a space between the IP address.
					if i > 0 {
						ips += " "
					}

					ips += fmt.Sprintf("%s", address.IP.String())
				}
			}
			r += fmt.Sprintf(" (%s)", ips)

		}
		ret = append(ret, r)
	}
	return ret, nil
}

func (sniffer *SnifferSetup) setFromConfig(config *config.InterfacesConfig) error {
	var err error

	sniffer.config = config

	if len(sniffer.config.File) > 0 {
		logp.Debug("sniffer", "Reading from file: %s", sniffer.config.File)
		// we read file with the pcap provider
		sniffer.config.Type = "pcap"
	}

	// set defaults
	if len(sniffer.config.Device) == 0 {
		sniffer.config.Device = "any"
	}

	if index, err := strconv.Atoi(sniffer.config.Device); err == nil { // Device is numeric
		devices, err := ListDeviceNames(false, false)
		if err != nil {
			return fmt.Errorf("Error getting devices list: %v", err)
		}
		sniffer.config.Device, err = deviceNameFromIndex(index, devices)
		if err != nil {
			return fmt.Errorf("Couldn't understand device index %d: %v", index, err)
		}
		logp.Info("Resolved device index %d to device: %s", index, sniffer.config.Device)
	}

	if sniffer.config.Snaplen == 0 {
		sniffer.config.Snaplen = 65535
	}

	if sniffer.config.Type == "autodetect" || sniffer.config.Type == "" {
		sniffer.config.Type = "pcap"
	}

	logp.Debug("sniffer", "Sniffer type: %s device: %s", sniffer.config.Type, sniffer.config.Device)

	switch sniffer.config.Type {
	case "pcap":
		if len(sniffer.config.File) > 0 {
			sniffer.pcapHandle, err = pcap.OpenOffline(sniffer.config.File)
			if err != nil {
				return err
			}
		} else {
			sniffer.pcapHandle, err = pcap.OpenLive(
				sniffer.config.Device,
				int32(sniffer.config.Snaplen),
				true,
				500*time.Millisecond)
			if err != nil {
				return err
			}
			err = sniffer.pcapHandle.SetBPFFilter(sniffer.filter)
			if err != nil {
				return err
			}
		}

		sniffer.DataSource = gopacket.PacketDataSource(sniffer.pcapHandle)

	case "af_packet":
		if sniffer.config.Buffer_size_mb == 0 {
			sniffer.config.Buffer_size_mb = 24
		}

		frame_size, block_size, num_blocks, err := afpacketComputeSize(
			sniffer.config.Buffer_size_mb,
			sniffer.config.Snaplen,
			os.Getpagesize())
		if err != nil {
			return err
		}

		sniffer.afpacketHandle, err = NewAfpacketHandle(
			sniffer.config.Device,
			frame_size,
			block_size,
			num_blocks,
			500*time.Millisecond)
		if err != nil {
			return err
		}

		err = sniffer.afpacketHandle.SetBPFFilter(sniffer.filter)
		if err != nil {
			return fmt.Errorf("SetBPFFilter failed: %s", err)
		}

		sniffer.DataSource = gopacket.PacketDataSource(sniffer.afpacketHandle)
	case "pfring", "pf_ring":
		sniffer.pfringHandle, err = NewPfringHandle(
			sniffer.config.Device,
			sniffer.config.Snaplen,
			true)

		if err != nil {
			return err
		}

		err = sniffer.pfringHandle.SetBPFFilter(sniffer.filter)
		if err != nil {
			return fmt.Errorf("SetBPFFilter failed: %s", err)
		}

		err = sniffer.pfringHandle.Enable()
		if err != nil {
			return fmt.Errorf("Enable failed: %s", err)
		}

		sniffer.DataSource = gopacket.PacketDataSource(sniffer.pfringHandle)

	default:
		return fmt.Errorf("Unknown sniffer type: %s", sniffer.config.Type)
	}

	return nil
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

func (sniffer *SnifferSetup) Datalink() layers.LinkType {
	if sniffer.config.Type == "pcap" {
		return sniffer.pcapHandle.LinkType()
	}
	return layers.LinkTypeEthernet
}

func (sniffer *SnifferSetup) Init(test_mode bool, factory WorkerFactory, interfaces *config.InterfacesConfig) error {
	var err error

	if !test_mode {
		err = sniffer.setFromConfig(interfaces)
		if err != nil {
			return fmt.Errorf("Error creating sniffer: %v", err)
		}
	}

	sniffer.worker, sniffer.filter, err = factory(sniffer.Datalink())
	if err != nil {
		return fmt.Errorf("Error creating decoder: %v", err)
	}
	logp.Debug("sniffer", "BPF filter: '%s'", sniffer.filter)

	if sniffer.config.Dumpfile != "" {
		p, err := pcap.OpenDead(sniffer.Datalink(), 65535)
		if err != nil {
			return err
		}
		sniffer.dumper, err = p.NewDumper(sniffer.config.Dumpfile)
		if err != nil {
			return err
		}
	}

	sniffer.isAlive = true

	return nil
}

func (sniffer *SnifferSetup) Run() error {
	counter := 0
	loopCount := 1
	var lastPktTime *time.Time = nil
	var ret_error error

	for sniffer.isAlive {
		if sniffer.config.OneAtATime {
			fmt.Println("Press enter to read packet")
			fmt.Scanln()
		}

		data, ci, err := sniffer.DataSource.ReadPacketData()

		if err == pcap.NextErrorTimeoutExpired || err == syscall.EINTR {
			logp.Debug("sniffer", "Interrupted")
			continue
		}

		if err == io.EOF {
			logp.Debug("sniffer", "End of file")
			loopCount += 1
			if sniffer.config.Loop > 0 && loopCount > sniffer.config.Loop {
				// give a bit of time to the publish goroutine
				// to flush
				time.Sleep(300 * time.Millisecond)
				sniffer.isAlive = false
				continue
			}

			logp.Debug("sniffer", "Reopening the file")
			err = sniffer.Reopen()
			if err != nil {
				ret_error = fmt.Errorf("Error reopening file: %s", err)
				sniffer.isAlive = false
				continue
			}
			lastPktTime = nil
			continue
		}

		if err != nil {
			ret_error = fmt.Errorf("Sniffing error: %s", err)
			sniffer.isAlive = false
			continue
		}

		if len(data) == 0 {
			// Empty packet, probably timeout from afpacket
			continue
		}

		if sniffer.config.File != "" {
			if lastPktTime != nil && !sniffer.config.TopSpeed {
				sleep := ci.Timestamp.Sub(*lastPktTime)
				if sleep > 0 {
					time.Sleep(sleep)
				} else {
					logp.Warn("Time in pcap went backwards: %d", sleep)
				}
			}
			_lastPktTime := ci.Timestamp
			lastPktTime = &_lastPktTime
			if !sniffer.config.TopSpeed {
				ci.Timestamp = time.Now() // overwrite what we get from the pcap
			}
		}
		counter++

		if sniffer.dumper != nil {
			sniffer.dumper.WritePacketData(data, ci)
		}
		logp.Debug("sniffer", "Packet number: %d", counter)

		sniffer.worker.OnPacket(data, &ci)
	}

	logp.Info("Input finish. Processed %d packets. Have a nice day!", counter)

	if sniffer.dumper != nil {
		sniffer.dumper.Close()
	}

	return ret_error
}

func (sniffer *SnifferSetup) Close() error {
	switch sniffer.config.Type {
	case "pcap":
		sniffer.pcapHandle.Close()
	case "af_packet":
		sniffer.afpacketHandle.Close()
	case "pfring", "pf_ring":
		sniffer.pfringHandle.Close()
	}
	return nil
}

func (sniffer *SnifferSetup) Stop() error {
	sniffer.isAlive = false
	return nil
}

func (sniffer *SnifferSetup) IsAlive() bool {
	return sniffer.isAlive
}
