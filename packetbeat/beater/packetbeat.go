package beater

import (
	"flag"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common/droppriv"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/service"
	"github.com/tsg/gopacket/layers"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/decoder"
	"github.com/elastic/beats/packetbeat/flows"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/icmp"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/protos/udp"
	"github.com/elastic/beats/packetbeat/publish"
	"github.com/elastic/beats/packetbeat/sniffer"
)

// Beater object. Contains all objects needed to run the beat
type Packetbeat struct {
	PbConfig    config.Config
	CmdLineArgs CmdLineArgs
	Pub         *publish.PacketbeatPublisher
	Sniff       *sniffer.SnifferSetup

	services []interface {
		Start()
		Stop()
	}
}

type CmdLineArgs struct {
	File         *string
	Loop         *int
	OneAtAtime   *bool
	TopSpeed     *bool
	Dumpfile     *string
	PrintDevices *bool
	WaitShutdown *int
}

var cmdLineArgs CmdLineArgs

const (
	defaultQueueSize     = 2048
	defaultBulkQueueSize = 0
)

func init() {
	cmdLineArgs = CmdLineArgs{
		File:         flag.String("I", "", "Read packet data from specified file"),
		Loop:         flag.Int("l", 1, "Loop file. 0 - loop forever"),
		OneAtAtime:   flag.Bool("O", false, "Read packets one at a time (press Enter)"),
		TopSpeed:     flag.Bool("t", false, "Read packets as fast as possible, without sleeping"),
		Dumpfile:     flag.String("dump", "", "Write all captured packets to this libpcap file"),
		PrintDevices: flag.Bool("devices", false, "Print the list of devices and exit"),
		WaitShutdown: flag.Int("waitstop", 0, "Additional seconds to wait before shutting down"),
	}
}

func New() *Packetbeat {

	pb := &Packetbeat{}
	pb.CmdLineArgs = cmdLineArgs

	return pb
}

// Handle custom command line flags
func (pb *Packetbeat) HandleFlags(b *beat.Beat) error {
	// -devices CLI flag
	if *pb.CmdLineArgs.PrintDevices {
		devs, err := sniffer.ListDeviceNames(true)
		if err != nil {
			return fmt.Errorf("Error getting devices list: %v\n", err)
		}
		if len(devs) == 0 {
			fmt.Printf("No devices found.")
			if runtime.GOOS != "windows" {
				fmt.Printf(" You might need sudo?\n")
			} else {
				fmt.Printf("\n")
			}
		}
		for i, dev := range devs {
			fmt.Printf("%d: %s\n", i, dev)
		}
		return beat.GracefulExit
	}
	return nil
}

// Loads the beat specific config and overwrites params based on cmd line
func (pb *Packetbeat) Config(b *beat.Beat) error {

	// Read beat implementation config as needed for setup
	err := b.RawConfig.Unpack(&pb.PbConfig)
	if err != nil {
		logp.Err("fails to read the beat config: %v, %v", err, pb.PbConfig)
		return err
	}

	cfg := &pb.PbConfig.Packetbeat

	// CLI flags over-riding config
	if *pb.CmdLineArgs.TopSpeed {
		cfg.Interfaces.TopSpeed = true
	}

	if len(*pb.CmdLineArgs.File) > 0 {
		cfg.Interfaces.File = *pb.CmdLineArgs.File
	}

	cfg.Interfaces.Loop = *pb.CmdLineArgs.Loop
	cfg.Interfaces.OneAtATime = *pb.CmdLineArgs.OneAtAtime

	if len(*pb.CmdLineArgs.Dumpfile) > 0 {
		cfg.Interfaces.Dumpfile = *pb.CmdLineArgs.Dumpfile
	}

	return nil
}

// Setup packetbeat
func (pb *Packetbeat) Setup(b *beat.Beat) error {

	cfg := &pb.PbConfig.Packetbeat

	if err := procs.ProcWatcher.Init(cfg.Procs); err != nil {
		logp.Critical(err.Error())
		return err
	}

	queueSize := defaultQueueSize
	if b.Config.Shipper.QueueSize != nil {
		queueSize = *b.Config.Shipper.QueueSize
	}
	bulkQueueSize := defaultBulkQueueSize
	if b.Config.Shipper.BulkQueueSize != nil {
		bulkQueueSize = *b.Config.Shipper.BulkQueueSize
	}
	pb.Pub = publish.NewPublisher(b.Publisher, queueSize, bulkQueueSize)
	pb.Pub.Start()

	logp.Debug("main", "Initializing protocol plugins")
	err := protos.Protos.Init(false, pb.Pub, cfg.Protocols)
	if err != nil {
		return fmt.Errorf("Initializing protocol analyzers failed: %v", err)
	}

	logp.Debug("main", "Initializing sniffer")
	if err := pb.setupSniffer(); err != nil {
		return fmt.Errorf("Initializing sniffer failed: %v", err)
	}

	// This needs to be after the sniffer Init but before the sniffer Run.
	if err := droppriv.DropPrivileges(cfg.RunOptions); err != nil {
		return err
	}

	return nil
}

func (pb *Packetbeat) setupSniffer() error {
	cfg := &pb.PbConfig.Packetbeat

	withVlans := cfg.Interfaces.With_vlans
	_, withICMP := cfg.Protocols["icmp"]
	filter := cfg.Interfaces.Bpf_filter
	if filter == "" && cfg.Flows == nil {
		filter = protos.Protos.BpfFilter(withVlans, withICMP)
	}

	pb.Sniff = &sniffer.SnifferSetup{}
	return pb.Sniff.Init(false, pb.makeWorkerFactory(filter), &pb.PbConfig.Packetbeat.Interfaces)
}

func (pb *Packetbeat) makeWorkerFactory(filter string) sniffer.WorkerFactory {
	return func(dl layers.LinkType) (sniffer.Worker, string, error) {
		var f *flows.Flows
		var err error

		if pb.PbConfig.Packetbeat.Flows != nil {
			f, err = flows.NewFlows(pb.Pub, pb.PbConfig.Packetbeat.Flows)
			if err != nil {
				return nil, "", err
			}
		}

		var icmp4 icmp.ICMPv4Processor
		var icmp6 icmp.ICMPv6Processor
		if cfg, exists := pb.PbConfig.Packetbeat.Protocols["icmp"]; exists {
			icmp, err := icmp.New(false, pb.Pub, cfg)
			if err != nil {
				return nil, "", err
			}

			icmp4 = icmp
			icmp6 = icmp
		}

		tcp, err := tcp.NewTcp(&protos.Protos)
		if err != nil {
			return nil, "", err
		}

		udp, err := udp.NewUdp(&protos.Protos)
		if err != nil {
			return nil, "", err
		}

		worker, err := decoder.NewDecoder(f, dl, icmp4, icmp6, tcp, udp)
		if err != nil {
			return nil, "", err
		}

		if f != nil {
			pb.services = append(pb.services, f)
		}
		return worker, filter, nil
	}
}

func (pb *Packetbeat) Run(b *beat.Beat) error {

	// start services
	for _, service := range pb.services {
		service.Start()
	}

	var wg sync.WaitGroup
	errC := make(chan error, 1)

	// Run the sniffer in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := pb.Sniff.Run()
		if err != nil {
			errC <- fmt.Errorf("Sniffer main loop failed: %v", err)
		}
	}()

	logp.Debug("main", "Waiting for the sniffer to finish")
	wg.Wait()
	select {
	default:
	case err := <-errC:
		return err
	}

	// kill services
	for _, service := range pb.services {
		service.Stop()
	}

	waitShutdown := pb.CmdLineArgs.WaitShutdown
	if waitShutdown != nil && *waitShutdown > 0 {
		time.Sleep(time.Duration(*waitShutdown) * time.Second)
	}

	return nil
}

func (pb *Packetbeat) Cleanup(b *beat.Beat) error {

	if service.WithMemProfile() {
		logp.Debug("main", "Waiting for streams and transactions to expire...")
		time.Sleep(time.Duration(float64(protos.DefaultTransactionExpiration) * 1.2))
		logp.Debug("main", "Streams and transactions should all be expired now.")
	}

	// TODO:
	// pb.TransPub.Stop()

	return nil
}

// Called by the Beat stop function
func (pb *Packetbeat) Stop() {
	logp.Info("Packetbeat send stop signal")
	pb.Sniff.Stop()
}
