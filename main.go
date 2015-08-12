package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/common/droppriv"
	"github.com/elastic/libbeat/filters"
	"github.com/elastic/libbeat/filters/nop"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/libbeat/service"

	"github.com/elastic/packetbeat/config"
	"github.com/elastic/packetbeat/procs"
	"github.com/elastic/packetbeat/protos"
	"github.com/elastic/packetbeat/protos/http"
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
	protos.HttpProtocol:    new(http.Http),
	protos.MysqlProtocol:   new(mysql.Mysql),
	protos.PgsqlProtocol:   new(pgsql.Pgsql),
	protos.RedisProtocol:   new(redis.Redis),
	protos.ThriftProtocol:  new(thrift.Thrift),
	protos.MongodbProtocol: new(mongodb.Mongodb),
}

var EnabledFilterPlugins map[filters.Filter]filters.FilterPlugin = map[filters.Filter]filters.FilterPlugin{
	filters.NopFilter: new(nop.Nop),
}

func main() {

	// Use our own FlagSet, because some libraries pollute the global one
	var cmdLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	cfgfile.CmdLineFlags(cmdLine, Name)
	logp.CmdLineFlags(cmdLine)
	service.CmdLineFlags(cmdLine)

	file := cmdLine.String("I", "", "file")
	loop := cmdLine.Int("l", 1, "Loop file. 0 - loop forever")
	oneAtAtime := cmdLine.Bool("O", false, "Read packets one at a time (press Enter)")
	topSpeed := cmdLine.Bool("t", false, "Read packets as fast as possible, without sleeping")
	publishDisabled := cmdLine.Bool("N", false, "Disable actual publishing for testing")
	printVersion := cmdLine.Bool("version", false, "Print version and exit")
	dumpfile := cmdLine.String("dump", "", "Write all captured packets to this libpcap file.")

	cmdLine.Parse(os.Args[1:])

	sniff := new(sniffer.SnifferSetup)

	if *printVersion {
		fmt.Printf("Packetbeat version %s (%s)\n", Version, runtime.GOARCH)
		return
	}

	err := cfgfile.Read(&config.ConfigSingleton)

	logp.Init(Name, &config.ConfigSingleton.Logging)

	// CLI flags over-riding config
	if *topSpeed {
		config.ConfigSingleton.Interfaces.TopSpeed = true
	}
	if len(*file) > 0 {
		config.ConfigSingleton.Interfaces.File = *file
	}
	config.ConfigSingleton.Interfaces.Loop = *loop
	config.ConfigSingleton.Interfaces.OneAtATime = *oneAtAtime
	if len(*dumpfile) > 0 {
		config.ConfigSingleton.Interfaces.Dumpfile = *dumpfile
	}

	logp.Debug("main", "Initializing output plugins")
	if err = publisher.Publisher.Init(*publishDisabled, config.ConfigSingleton.Output,
		config.ConfigSingleton.Shipper); err != nil {

		logp.Critical(err.Error())
		os.Exit(1)
	}

	if err = procs.ProcWatcher.Init(config.ConfigSingleton.Procs); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	logp.Debug("main", "Initializing protocol plugins")
	for proto, plugin := range EnabledProtocolPlugins {
		err = plugin.Init(false, publisher.Publisher.Queue)
		if err != nil {
			logp.Critical("Initializing plugin %s failed: %v", proto, err)
			os.Exit(1)
		}
		protos.Protos.Register(proto, plugin)
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

	over := make(chan bool)

	stopCb := func() {
		sniff.Stop()
	}

	logp.Debug("main", "Initializing filters")
	afterInputsQueue, err := filters.FiltersRun(
		config.ConfigSingleton.Filter,
		EnabledFilterPlugins,
		publisher.Publisher.Queue,
		stopCb)
	if err != nil {
		logp.Critical("%v", err)
		os.Exit(1)
	}

	logp.Debug("main", "Initializing sniffer")
	err = sniff.Init(false, afterInputsQueue, tcpProc, udpProc)
	if err != nil {
		logp.Critical("Initializing sniffer failed: %v", err)
		os.Exit(1)
	}

	// This needs to be after the sniffer Init but before the sniffer Run.
	if err = droppriv.DropPrivileges(config.ConfigSingleton.RunOptions); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	// Up to here was the initialization, now about running

	if cfgfile.IsTestConfig() {
		// all good, exit with 0
		os.Exit(0)
	}
	service.BeforeRun()

	// run the sniffer in background
	go func() {
		err := sniff.Run()
		if err != nil {
			logp.Critical("Sniffer main loop failed: %v", err)
			os.Exit(1)
		}
		over <- true
	}()

	service.HandleSignals(stopCb)

	// Startup successful, disable stderr logging if requested by
	// cmdline flag
	logp.SetStderr()

	logp.Debug("main", "Waiting for the sniffer to finish")

	// Wait for the goroutines to finish
	for _ = range over {
		if !sniff.IsAlive() {
			break
		}
	}

	logp.Debug("main", "Cleanup")

	if service.WithMemProfile() {
		// wait for all TCP streams to expire
		time.Sleep(tcp.TCP_STREAM_EXPIRY * 1.2)
		tcpProc.PrintTcpMap()
	}
	service.Cleanup()
}
