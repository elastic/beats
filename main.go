package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"packetbeat/common"
	"packetbeat/common/droppriv"
	"packetbeat/config"
	"packetbeat/filters"
	"packetbeat/filters/nop"
	"packetbeat/inputs"
	"packetbeat/inputs/gobeacon"
	"packetbeat/inputs/sniffer"
	"packetbeat/inputs/udpjson"
	"packetbeat/logp"
	"packetbeat/outputs"
	"packetbeat/procs"
	"packetbeat/protos"
	"packetbeat/protos/http"
	"packetbeat/protos/mysql"
	"packetbeat/protos/pgsql"
	"packetbeat/protos/redis"
	"packetbeat/protos/tcp"
	"packetbeat/protos/thrift"

	"github.com/BurntSushi/toml"
)

const Version = "0.5.0"

var EnabledProtocolPlugins map[protos.Protocol]protos.ProtocolPlugin = map[protos.Protocol]protos.ProtocolPlugin{
	protos.HttpProtocol:   new(http.Http),
	protos.MysqlProtocol:  new(mysql.Mysql),
	protos.PgsqlProtocol:  new(pgsql.Pgsql),
	protos.RedisProtocol:  new(redis.Redis),
	protos.ThriftProtocol: new(thrift.Thrift),
}

var EnabledInputPlugins map[inputs.Input]inputs.InputPlugin = map[inputs.Input]inputs.InputPlugin{
	inputs.SnifferInput:  new(sniffer.SnifferSetup),
	inputs.UdpjsonInput:  new(udpjson.Udpjson),
	inputs.GoBeaconInput: new(gobeacon.GoBeacon),
}

var EnabledFilterPlugins map[filters.Filter]filters.FilterPlugin = map[filters.Filter]filters.FilterPlugin{
	filters.NopFilter: new(nop.Nop),
}

func writeHeapProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		logp.Err("Failed creating file %s: %s", filename, err)
		return
	}
	pprof.WriteHeapProfile(f)
	f.Close()

	logp.Info("Created memory profile file %s.", filename)
}

func debugMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logp.Debug("mem", "Memory stats: In use: %d Total (even if freed): %d System: %d",
		m.Alloc, m.TotalAlloc, m.Sys)
}

func main() {

	// Use our own FlagSet, because some libraries pollute the global one
	var cmdLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	configfile := cmdLine.String("c", "packetbeat.conf", "Configuration file")
	file := cmdLine.String("I", "", "file")
	loop := cmdLine.Int("l", 1, "Loop file. 0 - loop forever")
	debugSelectorsStr := cmdLine.String("d", "", "Enable certain debug selectors")
	oneAtAtime := cmdLine.Bool("O", false, "Read packets one at a time (press Enter)")
	toStderr := cmdLine.Bool("e", false, "Output to stdout instead of syslog")
	topSpeed := cmdLine.Bool("t", false, "Read packets as fast as possible, without sleeping")
	publishDisabled := cmdLine.Bool("N", false, "Disable actual publishing for testing")
	verbose := cmdLine.Bool("v", false, "Log at INFO level")
	printVersion := cmdLine.Bool("version", false, "Print version and exit")
	memprofile := cmdLine.String("memprofile", "", "Write memory profile to this file")
	cpuprofile := cmdLine.String("cpuprofile", "", "Write cpu profile to file")
	dumpfile := cmdLine.String("dump", "", "Write all captured packets to this libpcap file.")

	cmdLine.Parse(os.Args[1:])

	if *printVersion {
		fmt.Printf("Packetbeat version %s (%s)\n", Version, runtime.GOARCH)
		return
	}

	logLevel := logp.LOG_ERR
	if *verbose {
		logLevel = logp.LOG_INFO
	}

	debugSelectors := []string{}
	if len(*debugSelectorsStr) > 0 {
		debugSelectors = strings.Split(*debugSelectorsStr, ",")
		logLevel = logp.LOG_DEBUG
	}

	var err error

	if config.ConfigMeta, err = toml.DecodeFile(*configfile, &config.ConfigSingleton); err != nil {
		fmt.Printf("TOML config parsing failed on %s: %s. Exiting.\n", *configfile, err)
		return
	}
	if len(debugSelectors) == 0 {
		debugSelectors = config.ConfigSingleton.Logging.Selectors
	}
	logp.LogInit(logp.Priority(logLevel), "", !*toStderr, true, debugSelectors)

	if !logp.IsDebug("stdlog") {
		// disable standard logging by default
		log.SetOutput(ioutil.Discard)
	}

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
	if err = outputs.Publisher.Init(*publishDisabled); err != nil {
		logp.Critical(err.Error())
		return
	}

	if err = procs.ProcWatcher.Init(&config.ConfigSingleton.Procs); err != nil {
		logp.Critical(err.Error())
		return
	}

	outputs.LoadGeoIPData()

	logp.Debug("main", "Initializing protocol plugins")
	for proto, plugin := range EnabledProtocolPlugins {
		err = plugin.Init(false, outputs.Publisher.Queue)
		if err != nil {
			logp.Critical("Initializing plugin %s failed: %v", proto, err)
			return
		}
		protos.Protos.Register(proto, plugin)
	}

	if err = tcp.TcpInit(config.ConfigSingleton.Protocols); err != nil {
		logp.Critical(err.Error())
		return
	}

	over := make(chan bool)

	if !config.ConfigMeta.IsDefined("input", "inputs") {
		config.ConfigSingleton.Input.Inputs = []string{"sniffer"}
	}
	if len(config.ConfigSingleton.Input.Inputs) == 0 {
		logp.Critical("At least one input needs to be enabled")
		return
	}

	logp.Debug("main", "Initializing filters plugins")
	for filter, plugin := range EnabledFilterPlugins {
		filters.Filters.Register(filter, plugin)
	}
	filters_plugins, err :=
		LoadConfiguredFilters(config.ConfigSingleton.Filter)
	if err != nil {
		logp.Critical("Error loading filters plugins: %v", err)
	}
	logp.Debug("main", "Filters plugins order: %v", filters_plugins)
	var afterInputsQueue chan common.MapStr
	if len(filters_plugins) > 0 {
		runner := NewFilterRunner(outputs.Publisher.Queue, filters_plugins)
		go func() {
			err := runner.Run()
			if err != nil {
				logp.Critical("Filters runner failed: %v", err)
				// shutting doen
				inputs.Inputs.StopAll()
			}
		}()
		afterInputsQueue = runner.FiltersQueue
	} else {
		// short-circuit the runner
		afterInputsQueue = outputs.Publisher.Queue
	}

	logp.Debug("main", "Initializing input plugins")
	for input, plugin := range EnabledInputPlugins {
		configured_inputs := config.ConfigSingleton.Input.Inputs
		if input.IsInList(configured_inputs) {
			logp.Debug("main", "Input plugin %s is enabled", input)
			err = plugin.Init(false, afterInputsQueue)
			if err != nil {
				logp.Critical("Ininitializing plugin %s failed: %v", input, err)
				return
			}
			inputs.Inputs.Register(input, plugin)
		}
	}

	// This needs to be after the sniffer Init but before the sniffer Run.
	if err = droppriv.DropPrivileges(); err != nil {
		logp.Critical(err.Error())
		return
	}

	// Up to here was the initialization, now about running

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	for input, plugin := range inputs.Inputs.Registered() {
		// run the plugin in background
		go func(input inputs.Input, plugin inputs.InputPlugin) {
			err := plugin.Run()
			if err != nil {
				logp.Critical("Plugin %s main loop failed: %v", input, err)
				return
			}
			over <- true
		}(input, plugin)
	}

	// On ^C or SIGTERM, gracefully stop inputs
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		logp.Debug("signal", "Received sigterm/sigint, stopping")
		inputs.Inputs.StopAll()
	}()

	if !*toStderr {
		logp.Info("Startup successful, sending output only to syslog from now on")
		logp.SetToStderr(false)
	}

	logp.Debug("main", "Waiting for inputs to finish")

	// Wait for one of the inputs goroutines to finish
	for _ = range over {
		if !inputs.Inputs.AreAllAlive() {
			break
		}
	}

	logp.Debug("main", "Cleanup")

	// stop and close all other input plugin servers
	inputs.Inputs.StopAll()
	inputs.Inputs.CloseAll()

	if *memprofile != "" {
		// wait for all TCP streams to expire
		time.Sleep(tcp.TCP_STREAM_EXPIRY * 1.2)
		tcp.PrintTcpMap()
		runtime.GC()

		writeHeapProfile(*memprofile)

		debugMemStats()
	}
}
