package beater

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common/droppriv"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/service"
	"github.com/tsg/gopacket/layers"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/decoder"
	"github.com/elastic/beats/packetbeat/flows"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/dns"
	"github.com/elastic/beats/packetbeat/protos/http"
	"github.com/elastic/beats/packetbeat/protos/icmp"
	"github.com/elastic/beats/packetbeat/protos/memcache"
	"github.com/elastic/beats/packetbeat/protos/mongodb"
	"github.com/elastic/beats/packetbeat/protos/mysql"
	"github.com/elastic/beats/packetbeat/protos/pgsql"
	"github.com/elastic/beats/packetbeat/protos/redis"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/protos/thrift"
	"github.com/elastic/beats/packetbeat/protos/udp"
	"github.com/elastic/beats/packetbeat/publish"
	"github.com/elastic/beats/packetbeat/sniffer"
)

var EnabledProtocolPlugins map[protos.Protocol]protos.ProtocolPlugin = map[protos.Protocol]protos.ProtocolPlugin{
	protos.HttpProtocol:     new(http.HTTP),
	protos.MemcacheProtocol: new(memcache.Memcache),
	protos.MysqlProtocol:    new(mysql.Mysql),
	protos.PgsqlProtocol:    new(pgsql.Pgsql),
	protos.RedisProtocol:    new(redis.Redis),
	protos.ThriftProtocol:   new(thrift.Thrift),
	protos.MongodbProtocol:  new(mongodb.Mongodb),
	protos.DnsProtocol:      new(dns.Dns),
}

// Beater object. Contains all objects needed to run the beat
type Packetbeat struct {
	PbConfig    config.Config
	CmdLineArgs CmdLineArgs
	Pub         *publish.PacketbeatPublisher
	Sniff       *sniffer.SnifferSetup
	over        chan bool

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
		File:         flag.String("I", "", "file"),
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
func (pb *Packetbeat) HandleFlags(b *beat.Beat) {
	// -devices CLI flag
	if *pb.CmdLineArgs.PrintDevices {
		devs, err := sniffer.ListDeviceNames(true)
		if err != nil {
			fmt.Printf("Error getting devices list: %v\n", err)
			os.Exit(1)
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
		os.Exit(0)
	}
}

// Loads the beat specific config and overwrites params based on cmd line
func (pb *Packetbeat) Config(b *beat.Beat) error {

	// Read beat implementation config as needed for setup
	err := cfgfile.Read(&pb.PbConfig, "")

	// CLI flags over-riding config
	if *pb.CmdLineArgs.TopSpeed {
		pb.PbConfig.Interfaces.TopSpeed = true
	}

	if len(*pb.CmdLineArgs.File) > 0 {
		pb.PbConfig.Interfaces.File = *pb.CmdLineArgs.File
	}

	pb.PbConfig.Interfaces.Loop = *pb.CmdLineArgs.Loop
	pb.PbConfig.Interfaces.OneAtATime = *pb.CmdLineArgs.OneAtAtime

	if len(*pb.CmdLineArgs.Dumpfile) > 0 {
		pb.PbConfig.Interfaces.Dumpfile = *pb.CmdLineArgs.Dumpfile
	}

	// assign global singleton as it is used in protocols
	// TODO: Refactor
	config.ConfigSingleton = pb.PbConfig

	return err
}

// Setup packetbeat
func (pb *Packetbeat) Setup(b *beat.Beat) error {

	if err := procs.ProcWatcher.Init(pb.PbConfig.Procs); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	queueSize := defaultQueueSize
	if pb.PbConfig.Shipper.QueueSize != nil {
		queueSize = *pb.PbConfig.Shipper.QueueSize
	}
	bulkQueueSize := defaultBulkQueueSize
	if pb.PbConfig.Shipper.BulkQueueSize != nil {
		bulkQueueSize = *pb.PbConfig.Shipper.BulkQueueSize
	}
	pb.Pub = publish.NewPublisher(b.Publisher, queueSize, bulkQueueSize)
	pb.Pub.Start()

	logp.Debug("main", "Initializing protocol plugins")
	for proto, plugin := range EnabledProtocolPlugins {
		err := plugin.Init(false, pb.Pub)
		if err != nil {
			logp.Critical("Initializing plugin %s failed: %v", proto, err)
			os.Exit(1)
		}
		protos.Protos.Register(proto, plugin)
	}

	pb.over = make(chan bool)

	logp.Debug("main", "Initializing sniffer")
	if err := pb.setupSniffer(); err != nil {
		logp.Critical("Initializing sniffer failed: %v", err)
		os.Exit(1)
	}

	// This needs to be after the sniffer Init but before the sniffer Run.
	if err := droppriv.DropPrivileges(config.ConfigSingleton.RunOptions); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	return nil
}

func (pb *Packetbeat) setupSniffer() error {
	cfg := &pb.PbConfig

	withVlans := cfg.Interfaces.With_vlans
	withICMP := cfg.Protocols.Icmp.Enabled
	filter := cfg.Interfaces.Bpf_filter
	if filter == "" && cfg.Flows == nil {
		filter = protos.Protos.BpfFilter(withVlans, withICMP)
	}

	pb.Sniff = &sniffer.SnifferSetup{}
	return pb.Sniff.Init(false, pb.makeWorkerFactory(filter))
}

func (pb *Packetbeat) makeWorkerFactory(filter string) sniffer.WorkerFactory {
	return func(dl layers.LinkType) (sniffer.Worker, string, error) {
		var f *flows.Flows
		var err error

		if pb.PbConfig.Flows != nil {
			f, err = flows.NewFlows(pb.Pub, pb.PbConfig.Flows)
			if err != nil {
				return nil, "", err
			}
		}

		icmp, err := icmp.NewIcmp(false, pb.Pub)
		if err != nil {
			return nil, "", err
		}

		tcp, err := tcp.NewTcp(&protos.Protos)
		if err != nil {
			return nil, "", err
		}

		udp, err := udp.NewUdp(&protos.Protos)
		if err != nil {
			return nil, "", err
		}

		worker, err := decoder.NewDecoder(f, dl, icmp, icmp, tcp, udp)
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

	// run the sniffer in background
	go func() {
		err := pb.Sniff.Run()
		if err != nil {
			logp.Critical("Sniffer main loop failed: %v", err)
			os.Exit(1)
		}
		pb.over <- true
	}()

	// Startup successful, disable stderr logging if requested by
	// cmdline flag
	logp.SetStderr()

	logp.Debug("main", "Waiting for the sniffer to finish")

	// Wait for the goroutines to finish
	for range pb.over {
		if !pb.Sniff.IsAlive() {
			break
		}
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
	pb.Sniff.Stop()
}
