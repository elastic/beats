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
type packetbeat struct {
	config      config.Config
	cmdLineArgs flags
	pub         *publish.PacketbeatPublisher
	sniff       *sniffer.SnifferSetup

	services []interface {
		Start()
		Stop()
	}
}

type flags struct {
	file         *string
	loop         *int
	oneAtAtime   *bool
	topSpeed     *bool
	dumpfile     *string
	waitShutdown *int
}

var cmdLineArgs flags

func init() {
	cmdLineArgs = flags{
		file:         flag.String("I", "", "Read packet data from specified file"),
		loop:         flag.Int("l", 1, "Loop file. 0 - loop forever"),
		oneAtAtime:   flag.Bool("O", false, "Read packets one at a time (press Enter)"),
		topSpeed:     flag.Bool("t", false, "Read packets as fast as possible, without sleeping"),
		dumpfile:     flag.String("dump", "", "Write all captured packets to this libpcap file"),
		waitShutdown: flag.Int("waitstop", 0, "Additional seconds to wait before shutting down"),
	}
}

func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	config := config.Config{
		Interfaces: config.InterfacesConfig{
			File:       *cmdLineArgs.file,
			Loop:       *cmdLineArgs.loop,
			TopSpeed:   *cmdLineArgs.topSpeed,
			OneAtATime: *cmdLineArgs.oneAtAtime,
			Dumpfile:   *cmdLineArgs.dumpfile,
		},
	}
	err := rawConfig.Unpack(&config)
	if err != nil {
		logp.Err("fails to read the beat config: %v, %v", err, config)
		return nil, err
	}

	pb := &packetbeat{
		config:      config,
		cmdLineArgs: cmdLineArgs,
	}
	err = pb.init(b)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

// init packetbeat components
func (pb *packetbeat) init(b *beat.Beat) error {

	cfg := &pb.config
	err := procs.ProcWatcher.Init(cfg.Procs)
	if err != nil {
		logp.Critical(err.Error())
		return err
	}

	// This is required as init Beat is called before the beat publisher is initialised
	b.Config.Shipper.InitShipperConfig()

	pb.pub, err = publish.NewPublisher(b.Publisher, *b.Config.Shipper.QueueSize, *b.Config.Shipper.BulkQueueSize, pb.config.IgnoreOutgoing)
	if err != nil {
		return fmt.Errorf("Initializing publisher failed: %v", err)
	}

	logp.Debug("main", "Initializing protocol plugins")
	err = protos.Protos.Init(false, pb.pub, cfg.Protocols)
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

func (pb *packetbeat) Run(b *beat.Beat) error {
	defer func() {
		if service.ProfileEnabled() {
			logp.Debug("main", "Waiting for streams and transactions to expire...")
			time.Sleep(time.Duration(float64(protos.DefaultTransactionExpiration) * 1.2))
			logp.Debug("main", "Streams and transactions should all be expired now.")
		}

		// TODO:
		// pb.TransPub.Stop()
	}()

	pb.pub.Start()

	// This needs to be after the sniffer Init but before the sniffer Run.
	if err := droppriv.DropPrivileges(pb.config.RunOptions); err != nil {
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
		err := pb.sniff.Run()
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

	waitShutdown := pb.cmdLineArgs.waitShutdown
	if waitShutdown != nil && *waitShutdown > 0 {
		time.Sleep(time.Duration(*waitShutdown) * time.Second)
	}

	return nil
}

// Called by the Beat stop function
func (pb *packetbeat) Stop() {
	logp.Info("Packetbeat send stop signal")
	pb.sniff.Stop()
	pb.pub.Stop()
}

func (pb *packetbeat) setupSniffer() error {
	config := &pb.config

	withVlans := config.Interfaces.WithVlans
	withICMP := config.Protocols["icmp"].Enabled()

	filter := config.Interfaces.BpfFilter
	if filter == "" && !config.Flows.IsEnabled() {
		filter = protos.Protos.BpfFilter(withVlans, withICMP)
	}

	pb.sniff = &sniffer.SnifferSetup{}
	return pb.sniff.Init(false, filter, pb.createWorker, &config.Interfaces)
}

func (pb *packetbeat) createWorker(dl layers.LinkType) (sniffer.Worker, error) {
	var f *flows.Flows
	var err error
	config := &pb.config

	if config.Flows.IsEnabled() {
		f, err = flows.NewFlows(pb.pub, config.Flows)
		if err != nil {
			return nil, err
		}
	}

	var icmp4 icmp.ICMPv4Processor
	var icmp6 icmp.ICMPv6Processor
	if cfg := config.Protocols["icmp"]; cfg.Enabled() {
		icmp, err := icmp.New(false, pb.pub, cfg)
		if err != nil {
			return nil, err
		}

		icmp4 = icmp
		icmp6 = icmp
	}

	tcp, err := tcp.NewTCP(&protos.Protos)
	if err != nil {
		return nil, err
	}

	udp, err := udp.NewUDP(&protos.Protos)
	if err != nil {
		return nil, err
	}

	worker, err := decoder.New(f, dl, icmp4, icmp6, tcp, udp)
	if err != nil {
		return nil, err
	}

	if f != nil {
		pb.services = append(pb.services, f)
	}
	return worker, nil
}
