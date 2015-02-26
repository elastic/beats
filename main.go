package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"packetbeat/common/droppriv"
	"packetbeat/config"
	"packetbeat/inputs/sniffer"
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
	"github.com/packetbeat/gopacket/pcap"
)

const Version = "0.4.3"

var EnabledProtocolPlugins map[protos.Protocol]protos.ProtocolPlugin = map[protos.Protocol]protos.ProtocolPlugin{
	protos.HttpProtocol:   new(http.Http),
	protos.MysqlProtocol:  new(mysql.Mysql),
	protos.PgsqlProtocol:  new(pgsql.Pgsql),
	protos.RedisProtocol:  new(redis.Redis),
	protos.ThriftProtocol: new(thrift.Thrift),
}

// Structure grouping main components/modules
type PacketbeatStruct struct {
	Sniffer *sniffer.SnifferSetup
	Decoder *tcp.DecoderStruct
}

// Global variable containing the main values
var Packetbeat PacketbeatStruct

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
	toStdout := cmdLine.Bool("e", false, "Output to stdout instead of syslog")
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
	logp.LogInit(logp.Priority(logLevel), "", !*toStdout, debugSelectors)

	if !logp.IsDebug("stdlog") {
		// disable standard logging by default
		log.SetOutput(ioutil.Discard)
	}

	config.ConfigSingleton.Interfaces.Bpf_filter =
		tcp.ConfigToFilter(config.ConfigSingleton.Protocols)
	Packetbeat.Sniffer, err = sniffer.CreateSniffer(&config.ConfigSingleton.Interfaces, file)
	if err != nil {
		logp.Critical("Error creating sniffer: %s", err)
		return
	}
	sniffer := Packetbeat.Sniffer
	Packetbeat.Decoder, err = tcp.CreateDecoder(sniffer.Datalink())
	if err != nil {
		logp.Critical("Error creating decoder: %s", err)
		return
	}

	if err = droppriv.DropPrivileges(); err != nil {
		logp.Critical(err.Error())
		return
	}

	if err = outputs.Publisher.Init(*publishDisabled); err != nil {
		logp.Critical(err.Error())
		return
	}

	if err = procs.ProcWatcher.Init(&config.ConfigSingleton.Procs); err != nil {
		logp.Critical(err.Error())
		return
	}

	outputs.LoadGeoIPData()

	// Initializing protocol plugins
	for proto, plugin := range EnabledProtocolPlugins {
		err = plugin.Init(false, outputs.Publisher.Queue)
		if err != nil {
			logp.Critical("Initializing plugin %s failed: %v", proto, plugin)
			return
		}
		protos.Protos.Register(proto, plugin)
	}

	if err = tcp.TcpInit(config.ConfigSingleton.Protocols); err != nil {
		logp.Critical(err.Error())
		return
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var dumper *pcap.Dumper = nil
	if *dumpfile != "" {
		p, err := pcap.OpenDead(sniffer.Datalink(), 65535)
		if err != nil {
			logp.Critical(err.Error())
			return
		}
		dumper, err = p.NewDumper(*dumpfile)
		if err != nil {
			logp.Critical(err.Error())
			return
		}
	}

	live := true

	// On ^C or SIGTERM, gracefully set live to false
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		live = false
		logp.Debug("signal", "Received term singal, set live to false")
	}()

	counter := 0
	loopCount := 1
	var lastPktTime *time.Time = nil
	for live {
		if *oneAtAtime {
			fmt.Println("Press enter to read packet")
			fmt.Scanln()
		}

		data, ci, err := sniffer.DataSource.ReadPacketData()

		if err == pcap.NextErrorTimeoutExpired || err == syscall.EINTR {
			logp.Debug("pcapread", "Interrupted")
			continue
		}

		if err == io.EOF {
			logp.Debug("pcapread", "End of file")
			loopCount += 1
			if *loop > 0 && loopCount > *loop {
				// give a bit of time to the publish goroutine
				// to flush
				time.Sleep(300 * time.Millisecond)
				live = false
				continue
			}

			logp.Debug("pcapread", "Reopening the file")
			err = sniffer.Reopen()
			if err != nil {
				logp.Critical("Error reopening file: %s", err)
				live = false
				continue
			}
			lastPktTime = nil
			continue
		}

		if err != nil {
			logp.Critical("Sniffing error: %s", err)
			live = false
			continue
		}

		if len(data) == 0 {
			// Empty packet, probably timeout from afpacket
			continue
		}

		if *file != "" {
			if lastPktTime != nil && !*topSpeed {
				sleep := ci.Timestamp.Sub(*lastPktTime)
				if sleep > 0 {
					time.Sleep(sleep)
				} else {
					logp.Warn("Time in pcap went backwards: %d", sleep)
				}
			}
			_lastPktTime := ci.Timestamp
			lastPktTime = &_lastPktTime
			ci.Timestamp = time.Now() // overwrite what we get from the pcap
		}
		counter++

		if dumper != nil {
			dumper.WritePacketData(data, ci)
		}
		logp.Debug("pcapread", "Packet number: %d", counter)
		Packetbeat.Decoder.DecodePacketData(data, &ci)
	}
	logp.Info("Input finish. Processed %d packets. Have a nice day!", counter)

	if *memprofile != "" {
		// wait for all TCP streams to expire
		time.Sleep(tcp.TCP_STREAM_EXPIRY * 1.2)
		tcp.PrintTcpMap()
		runtime.GC()

		writeHeapProfile(*memprofile)

		debugMemStats()
	}

	if dumper != nil {
		dumper.Close()
	}
}
