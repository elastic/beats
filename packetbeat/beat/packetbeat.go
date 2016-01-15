package beat

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

	"github.com/elastic/beats/packetbeat/config"
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
	Sniff       *sniffer.SnifferSetup
	over        chan bool
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

	pb.Sniff = new(sniffer.SnifferSetup)

	logp.Debug("main", "Initializing protocol plugins")
	for proto, plugin := range EnabledProtocolPlugins {
		err := plugin.Init(false, b.Events)
		if err != nil {
			logp.Critical("Initializing plugin %s failed: %v", proto, err)
			os.Exit(1)
		}
		protos.Protos.Register(proto, plugin)
	}

	var err error

	icmpProc, err := icmp.NewIcmp(false, b.Events)
	if err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	tcpProc, err := tcp.NewTcp(&protos.Protos)
	if err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	udpProc, err := udp.NewUdp(&protos.Protos)
	if err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	pb.over = make(chan bool)

	logp.Debug("main", "Initializing sniffer")
	err = pb.Sniff.Init(false, icmpProc, icmpProc, tcpProc, udpProc)
	if err != nil {
		logp.Critical("Initializing sniffer failed: %v", err)
		os.Exit(1)
	}

	// This needs to be after the sniffer Init but before the sniffer Run.
	if err = droppriv.DropPrivileges(config.ConfigSingleton.RunOptions); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	return err
}

func (pb *Packetbeat) Run(b *beat.Beat) error {

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
	return nil
}

// Called by the Beat stop function
func (pb *Packetbeat) Stop() {
	pb.Sniff.Stop()
}
