package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/nranchev/go-libGeoIP"
	_ "github.com/packetbeat/gopacket/layers"
	"github.com/packetbeat/gopacket/pcap"
)

const Version = "0.4.0"

// Structure grouping main components/modules
type PacketbeatStruct struct {
	Sniffer *SnifferSetup
	Decoder *DecoderStruct
}

// Global variable containing the main values
var Packetbeat PacketbeatStruct

type protocolType uint16

const (
	UnknownProtocol protocolType = iota
	HttpProtocol
	MysqlProtocol
	RedisProtocol
	PgsqlProtocol
	ThriftProtocol
)

var protocolNames = []string{"unknown", "http", "mysql", "redis", "pgsql", "thrift"}

type tomlConfig struct {
	Interfaces tomlInterfaces
	RunOptions tomlRunOptions
	Protocols  map[string]tomlProtocol
	Procs      tomlProcs
	Output     map[string]tomlMothership
	Agent      tomlAgent
	Logging    tomlLogging
	Passwords  tomlPasswords
	Thrift     tomlThrift
}

type tomlRunOptions struct {
	Uid int
	Gid int
}

type tomlLogging struct {
	Selectors []string
}

type tomlPasswords struct {
	Hide_keywords []string
}

var _Config tomlConfig
var _ConfigMeta toml.MetaData
var _GeoLite *libgeo.GeoIP

func Bytes_Ipv4_Ntoa(bytes []byte) string {
	var strarr []string = make([]string, 4)
	for i, b := range bytes {
		strarr[i] = strconv.Itoa(int(b))
	}
	return strings.Join(strarr, ".")
}

func writeHeapProfile(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		ERR("Failed creating file %s: %s", filename, err)
		return
	}
	pprof.WriteHeapProfile(f)
	f.Close()

	INFO("Created memory profile file %s.", filename)
}

func debugMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	DEBUG("mem", "Memory stats: In use: %d Total (even if freed): %d System: %d",
		m.Alloc, m.TotalAlloc, m.Sys)
}

func loadGeoIPData() {
	geoip_path := "/usr/share/GeoIP/GeoIP.dat"
	fi, err := os.Lstat(geoip_path)
	if err != nil {
		WARN("Could not load GeoIP data: %s", err.Error())
		return
	}

	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		// follow symlink
		geoip_path, err = filepath.EvalSymlinks(geoip_path)
		if err != nil {
			WARN("Could not load GeoIP data: %s", err.Error())
			return
		}
	}

	_GeoLite, err = libgeo.Load(geoip_path)
	if err != nil {
		WARN("Could not load GeoIP data: %s", err.Error())
	}
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
	memprofile := cmdLine.String("memprofile", "", "write memory profile to this file")
	cpuprofile := cmdLine.String("cpuprofile", "", "write cpu profile to file")

	cmdLine.Parse(os.Args[1:])

	if *printVersion {
		fmt.Printf("Packetbeat version %s (%s)\n", Version, runtime.GOARCH)
		return
	}

	logLevel := LOG_ERR
	if *verbose {
		logLevel = LOG_INFO
	}

	debugSelectors := []string{}
	if len(*debugSelectorsStr) > 0 {
		debugSelectors = strings.Split(*debugSelectorsStr, ",")
		logLevel = LOG_DEBUG
	}

	var err error

	if _ConfigMeta, err = toml.DecodeFile(*configfile, &_Config); err != nil {
		fmt.Printf("TOML config parsing failed on %s: %s. Exiting.\n", *configfile, err)
		return
	}

	if len(debugSelectors) == 0 {
		debugSelectors = _Config.Logging.Selectors
	}
	LogInit(logLevel, "", !*toStdout, debugSelectors)

	if !IS_DEBUG("stdlog") {
		// disable standard logging by default
		log.SetOutput(ioutil.Discard)
	}

	_Config.Interfaces.Bpf_filter = configToFilter(&_Config)
	Packetbeat.Sniffer, err = CreateSniffer(&_Config.Interfaces, file)
	if err != nil {
		CRIT("Error creating sniffer: %s", err)
		return
	}
	sniffer := Packetbeat.Sniffer
	Packetbeat.Decoder, err = CreateDecoder(sniffer.Datalink())
	if err != nil {
		CRIT("Error creating decoder: %s", err)
		return
	}

	if err = DropPrivileges(); err != nil {
		CRIT(err.Error())
		return
	}

	if err = Publisher.Init(*publishDisabled); err != nil {
		CRIT(err.Error())
		return
	}

	if err = procWatcher.Init(&_Config.Procs); err != nil {
		CRIT(err.Error())
		return
	}

	if err = ThriftMod.Init(false); err != nil {
		CRIT(err.Error())
		return
	}

	if err = TcpInit(); err != nil {
		CRIT(err.Error())
		return
	}

	loadGeoIPData()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	live := true

	// On ^C or SIGTERM, gracefully set live to false
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		live = false
		DEBUG("signal", "Received term singal, set live to false")
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
			DEBUG("pcapread", "Interrupted")
			continue
		}

		if err == io.EOF {
			DEBUG("pcapread", "End of file")
			loopCount += 1
			if *loop > 0 && loopCount > *loop {
				// give a bit of time to the publish goroutine
				// to flush
				time.Sleep(300 * time.Millisecond)
				live = false
				continue
			}

			DEBUG("pcapread", "Reopening the file")
			err = sniffer.Reopen()
			if err != nil {
				CRIT("Error reopening file: %s", err)
				live = false
				continue
			}
			lastPktTime = nil
			continue
		}

		if err != nil {
			CRIT("Sniffing error: %s", err)
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
					WARN("Time in pcap went backwards: %d", sleep)
				}
			}
			_lastPktTime := ci.Timestamp
			lastPktTime = &_lastPktTime
			ci.Timestamp = time.Now() // overwrite what we get from the pcap
		}
		counter++

		DEBUG("pcapread", "Packet number: %d", counter)
		Packetbeat.Decoder.DecodePacketData(data, &ci)
	}
	INFO("Input finish. Processed %d packets. Have a nice day!", counter)

	if *memprofile != "" {
		// wait for all TCP streams to expire
		time.Sleep(TCP_STREAM_EXPIRY * 1.2)
		PrintTcpMap()
		runtime.GC()

		writeHeapProfile(*memprofile)

		debugMemStats()
	}
}
