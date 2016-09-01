package beater

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
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
	Config      config.Config
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
	WaitShutdown *int
}

var cmdLineArgs CmdLineArgs

func init() {
	cmdLineArgs = CmdLineArgs{
		File:         flag.String("I", "", "Read packet data from specified file"),
		Loop:         flag.Int("l", 1, "Loop file. 0 - loop forever"),
		OneAtAtime:   flag.Bool("O", false, "Read packets one at a time (press Enter)"),
		TopSpeed:     flag.Bool("t", false, "Read packets as fast as possible, without sleeping"),
		Dumpfile:     flag.String("dump", "", "Write all captured packets to this libpcap file"),
		WaitShutdown: flag.Int("waitstop", 0, "Additional seconds to wait before shutting down"),
	}
}

func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	config := config.Config{
		Interfaces: config.InterfacesConfig{
			File:       *cmdLineArgs.File,
			Loop:       *cmdLineArgs.Loop,
			TopSpeed:   *cmdLineArgs.TopSpeed,
			OneAtATime: *cmdLineArgs.OneAtAtime,
			Dumpfile:   *cmdLineArgs.Dumpfile,
		},
	}
	err := rawConfig.Unpack(&config)
	if err != nil {
		logp.Err("fails to read the beat config: %v, %v", err, config)
		return nil, err
	}

	pb := &Packetbeat{
		Config:      config,
		CmdLineArgs: cmdLineArgs,
	}
	err = pb.init(b)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

// init packetbeat components
func (pb *Packetbeat) init(b *beat.Beat) error {

	cfg := &pb.Config
	err := procs.ProcWatcher.Init(cfg.Procs)
	if err != nil {
		logp.Critical(err.Error())
		return err
	}

	// This is required as init Beat is called before the beat publisher is initialised
	b.Config.Shipper.InitShipperConfig()

	pb.Pub, err = publish.NewPublisher(b.Publisher, *b.Config.Shipper.QueueSize, *b.Config.Shipper.BulkQueueSize, pb.Config.IgnoreOutgoing)
	if err != nil {
		return fmt.Errorf("Initializing publisher failed: %v", err)
	}

	logp.Debug("main", "Initializing protocol plugins")
	err = protos.Protos.Init(false, pb.Pub, cfg.Protocols)
	if err != nil {
		return fmt.Errorf("Initializing protocol analyzers failed: %v", err)
	}

	logp.Debug("main", "Initializing sniffer")
	err = pb.setupSniffer()
	if err != nil {
		return fmt.Errorf("Initializing sniffer failed: %v", err)
	}

	return nil
}

func (pb *Packetbeat) Run(b *beat.Beat) error {
	defer func() {
		if service.WithMemProfile() {
			logp.Debug("main", "Waiting for streams and transactions to expire...")
			time.Sleep(time.Duration(float64(protos.DefaultTransactionExpiration) * 1.2))
			logp.Debug("main", "Streams and transactions should all be expired now.")
		}

		// TODO:
		// pb.TransPub.Stop()
	}()

	pb.Pub.Start()

	// This needs to be after the sniffer Init but before the sniffer Run.
	if err := droppriv.DropPrivileges(pb.Config.RunOptions); err != nil {
		return err
	}

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

// Called by the Beat stop function
func (pb *Packetbeat) Stop() {
	logp.Info("Packetbeat send stop signal")
	pb.Sniff.Stop()
	pb.Pub.Stop()
}

func (pb *Packetbeat) setupSniffer() error {
	config := &pb.Config

	withVlans := config.Interfaces.With_vlans
	withICMP := config.Protocols["icmp"].Enabled()

	filter := config.Interfaces.Bpf_filter
	if filter == "" && !config.Flows.IsEnabled() {
		filter = protos.Protos.BpfFilter(withVlans, withICMP)
	}

	pb.Sniff = &sniffer.SnifferSetup{}
	return pb.Sniff.Init(false, pb.makeWorkerFactory(filter), &config.Interfaces)
}

func (pb *Packetbeat) makeWorkerFactory(filter string) sniffer.WorkerFactory {
	return func(dl layers.LinkType) (sniffer.Worker, string, error) {
		var f *flows.Flows
		var err error
		config := &pb.Config

		if config.Flows.IsEnabled() {
			f, err = flows.NewFlows(pb.Pub, config.Flows)
			if err != nil {
				return nil, "", err
			}
		}

		var icmp4 icmp.ICMPv4Processor
		var icmp6 icmp.ICMPv6Processor
		if cfg := config.Protocols["icmp"]; cfg.Enabled() {
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
