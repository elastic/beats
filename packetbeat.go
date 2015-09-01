package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common/droppriv"
	"github.com/elastic/libbeat/filters"
	"github.com/elastic/libbeat/filters/nop"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/libbeat/service"

	"github.com/elastic/packetbeat/beat"
	"github.com/elastic/packetbeat/config"
	"github.com/elastic/packetbeat/procs"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/dns"
	"github.com/elastic/packetbeat/protos/http"
	"github.com/elastic/packetbeat/protos/memcache"
	"github.com/elastic/packetbeat/protos/mongodb"
	"github.com/elastic/packetbeat/protos/mysql"
	"github.com/elastic/packetbeat/protos/pgsql"
	"github.com/elastic/packetbeat/protos/redis"
	"github.com/elastic/packetbeat/protos/tcp"
	"github.com/elastic/packetbeat/protos/thrift"
	"github.com/elastic/packetbeat/protos/udp"
	"github.com/elastic/packetbeat/sniffer"
)

// You can overwrite these, e.g.: go build -ldflags "-X main.Version 1.0.0-beta3"
var Version = "1.0.0-beta2"
var Name = "packetbeat"

var EnabledProtocolPlugins map[protos.Protocol]protos.ProtocolPlugin = map[protos.Protocol]protos.ProtocolPlugin{
	protos.HttpProtocol:     new(http.Http),
	protos.MemcacheProtocol: new(memcache.Memcache),
	protos.MysqlProtocol:    new(mysql.Mysql),
	protos.PgsqlProtocol:    new(pgsql.Pgsql),
	protos.RedisProtocol:    new(redis.Redis),
	protos.ThriftProtocol:   new(thrift.Thrift),
	protos.MongodbProtocol:  new(mongodb.Mongodb),
	protos.DnsProtocol:      new(dns.Dns),
}

var EnabledFilterPlugins map[filters.Filter]filters.FilterPlugin = map[filters.Filter]filters.FilterPlugin{
	filters.NopFilter: new(nop.Nop),
}

// Beater object. Contains all objects needed to run the beat
type Packetbeat struct {
	PbConfig    config.Config
	CmdLineArgs CmdLineArgs
	Sniff       *sniffer.SnifferSetup
	over        chan bool
	tcpProc     *tcp.Tcp
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

func fetchAdditionalCmdLineArgs(cmdLine *flag.FlagSet) CmdLineArgs {

	args := CmdLineArgs{
		File:         cmdLine.String("I", "", "file"),
		Loop:         cmdLine.Int("l", 1, "Loop file. 0 - loop forever"),
		OneAtAtime:   cmdLine.Bool("O", false, "Read packets one at a time (press Enter)"),
		TopSpeed:     cmdLine.Bool("t", false, "Read packets as fast as possible, without sleeping"),
		Dumpfile:     cmdLine.String("dump", "", "Write all captured packets to this libpcap file"),
		PrintDevices: cmdLine.Bool("devices", false, "Print the list of devices and exit"),
		WaitShutdown: cmdLine.Int("waitstop", 0, "Additional seconds to wait befor shutting down"),
	}

	return args
}

// Handle custom command line flags
func (pb *Packetbeat) CliFlags(b *beat.Beat) {
	// -devices CLI flag
	if *pb.CmdLineArgs.PrintDevices {
		devs, err := sniffer.ListDeviceNames()
		if err != nil {
			fmt.Printf("Error getting devices list: %v\n", err)
			os.Exit(1)
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
	err := cfgfile.Read(&pb.PbConfig)

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
		err := plugin.Init(false, publisher.Publisher.Queue)
		if err != nil {
			logp.Critical("Initializing plugin %s failed: %v", proto, err)
			os.Exit(1)
		}
		protos.Protos.Register(proto, plugin)
	}

	var err error

	pb.tcpProc, err = tcp.NewTcp(&protos.Protos)
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

	logp.Debug("main", "Initializing filters")
	afterInputsQueue, err := filters.FiltersRun(
		config.ConfigSingleton.Filter,
		EnabledFilterPlugins,
		publisher.Publisher.Queue,
		b.Stop)

	if err != nil {
		logp.Critical("%v", err)
		os.Exit(1)
	}

	logp.Debug("main", "Initializing sniffer")
	err = pb.Sniff.Init(false, afterInputsQueue, pb.tcpProc, udpProc)
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
	for _ = range pb.over {
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
		// wait for all TCP streams to expire
		time.Sleep(tcp.TCP_STREAM_EXPIRY * 1.2)
		pb.tcpProc.PrintTcpMap()
	}
	return nil
}

// Called by the Beat stop function
func (pb *Packetbeat) Stop() {
	pb.Sniff.Stop()
}
