package main

import (
    "flag"
    "fmt"
    "log/syslog"
    "strconv"
    "strings"
    "syscall"
    "time"

    "github.com/BurntSushi/toml"
    "github.com/akrennmair/gopcap"
	"github.com/nranchev/go-libGeoIP"
)

type Packet struct {
    ts      time.Time
    tuple   IpPortTuple
    payload []byte
}

type protocolType uint16

const (
    UnknownProtocol protocolType = iota
    HttpProtocol
    MysqlProtocol
    RedisProtocol
)

var protocolNames = []string{"unknown", "http", "mysql", "redis"}

type tomlConfig struct {
    Interfaces      tomlInterfaces
    RunOptions      tomlRunOptions
    Protocols       map[string]tomlProtocol
    Procs           tomlProcs
    Elasticsearch   tomlMothership
    Agent           tomlAgent
    Logging         tomlLogging
}

type tomlInterfaces struct {
    Device string
}

type tomlRunOptions struct {
    Uid int
    Gid int
}

type tomlLogging struct {
    Selectors []string
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

func decodePktEth(datalink int, pkt *pcap.Packet) {

    packet := new(Packet)
    var l2hlen int
    var eth_type uint16

    if datalink != pcap.LINKTYPE_LINUX_SLL {
        // we are just assuming ETH
        l2hlen = 14

        // bytes 12 and 13 are the pkt type
        eth_type = Bytes_Ntohs(pkt.Data[12:14])

    } else {
        l2hlen = 16

        // bytes 14 and 15 are the pkt type
        eth_type = Bytes_Ntohs(pkt.Data[14:16])
    }

    if eth_type != 0x800 {
        DEBUG("ip", "Not ipv4 packet. Ignoring")
        return
    }

    if len(pkt.Data) < l2hlen+20 {
        DEBUG("ip", "Packet to short to be ethernet")
        return
    }

    // IP header
    iphl := int((uint16(pkt.Data[l2hlen]) & 0x0f) * 4)
    if len(pkt.Data) < l2hlen+iphl {
        DEBUG("ip", "Packet to short to be IP")
        return
    }
    iphdr := pkt.Data[l2hlen : l2hlen+iphl]

    //DEBUG("Packet timestamp: %s", pkt.Time)
    packet.ts = pkt.Time

    packet.tuple.Src_ip = Bytes_Ntohl(iphdr[12:16])
    packet.tuple.Dst_ip = Bytes_Ntohl(iphdr[16:20])

    mf := (uint8(iphdr[6]&0x20) != 0)
    frag_offset := (uint16(iphdr[7]) << 3) | (uint16(iphdr[6]) & 0x1F)

    if mf || frag_offset > 0 {
        DEBUG("ip", "Fragmented packets not yet supported")
        return
    }

    ip_length := int(Bytes_Ntohs(iphdr[2:4]))

    protocol := uint8(iphdr[9])
    if protocol != 6 {
        DEBUG("ip", "Not TCP packet received. Ignoring")
        return
    }

    if len(pkt.Data) < l2hlen+iphl+20 {
        DEBUG("ip", "Packet too short to be TCP")
        return
    }

    tcphl := int((uint16(pkt.Data[l2hlen+iphl+12]) >> 4) * 4)
    if tcphl > 20 && len(pkt.Data) < l2hlen+iphl+tcphl {
        DEBUG("ip", "Packet too short for the TCP header")
        return
    }

    tcphdr := pkt.Data[l2hlen+iphl : l2hlen+iphl+tcphl]
    packet.tuple.Src_port = Bytes_Ntohs(tcphdr[0:2])
    packet.tuple.Dst_port = Bytes_Ntohs(tcphdr[2:4])

    //DEBUG(" %s:%d -> %s:%d", packet.tuple.src_ip, packet.tuple.src_port,
    //      packet.tuple.dst_ip, packet.tuple.dst_port)

    packet.payload = pkt.Data[l2hlen+iphl+tcphl : l2hlen+ip_length]

    FollowTcp(tcphdr, packet)
}

func DropPrivileges() error {
    var err error

    if !_ConfigMeta.IsDefined("runoptions", "uid") {
        // not found, no dropping privileges but no err
        return nil
    }

    if !_ConfigMeta.IsDefined("runoptions", "gid") {
        return MsgError("GID must be specified for dropping priviledges")
    }

    INFO("Switching to user: %d.%d", _Config.RunOptions.Uid, _Config.RunOptions.Gid)

    if err = syscall.Setgid(_Config.RunOptions.Gid); err != nil {
        return MsgError("setgid: %s", err.Error())
    }

    if err = syscall.Setuid(_Config.RunOptions.Uid); err != nil {
        return MsgError("setuid: %s", err.Error())
    }

    return nil
}

func main() {

    configfile := flag.String("c", "agent.conf", "Configuration file")
    file := flag.String("I", "", "file")
    loop := flag.Int("l", 0, "Loop file. 0 - loop forever")
    debugSelectorsStr := flag.String("d", "", "Enable certain debug selectors")
    oneAtAtime := flag.Bool("O", false, "Read packets one at a time (press Enter)")
    toStdout := flag.Bool("e", false, "Output to stdout instead of syslog")
    panicAfter := flag.Int("P", 0, "Panic after a given number of seconds, only for testing")

    flag.Parse()

    debugSelectors := []string{}
    logLevel := syslog.LOG_DEBUG
    if len(*debugSelectorsStr) > 0 {
        debugSelectors = strings.Split(*debugSelectorsStr, ",")
    }

    var h *pcap.Pcap
    var err error

    if _ConfigMeta, err = toml.DecodeFile(*configfile, &_Config); err != nil {
        fmt.Printf("TOML config parsing failed on %s: %s. We will use defaults.\n", *configfile, err)
    }

    if len(debugSelectors) == 0 {
        debugSelectors = _Config.Logging.Selectors
    }
    LogInit(logLevel, "", !*toStdout, debugSelectors)

    if *panicAfter > 0 {
        go func() {
            time.Sleep(time.Duration(*panicAfter) * time.Second)
            panic("Panicking as requested")
        }()
    }

    if *file != "" {
        h, err = pcap.Openoffline(*file)
        if h == nil {
            ERR("Openoffline(%s) failed: %s", *file, err)
            return
        }
    } else {
        device := _Config.Interfaces.Device

        h, err = pcap.Openlive(device, 65535, true, 0)
        if h == nil {
            ERR("pcap.Openlive failed(%s): %s", device, err)
            return
        }
    }
    defer h.Close()

    if err = DropPrivileges(); err != nil {
        CRIT(err.Error())
        return
    }

    filter := configToFilter(&_Config)
    if filter != "" {
        DEBUG("pcapfilter", "Installing filter '%s'", filter)
        err := h.Setfilter(filter)
        if err != nil {
            ERR("pcap.Setfilter failed: %s", err)
            return
        }
    }

    if err = Publisher.Init(); err != nil {
        CRIT(err.Error())
        return
    }

    if err = procWatcher.Init(&_Config.Procs); err != nil {
        CRIT(err.Error())
        return
    }

    if err = TcpInit(); err != nil {
        CRIT(err.Error())
        return
    }

    datalink := h.Datalink()
    if datalink != pcap.LINKTYPE_ETHERNET && datalink != pcap.LINKTYPE_LINUX_SLL {
        WARN("Unsuported link type: %d", datalink)
    }

	_GeoLite, err = libgeo.Load("/usr/share/GeoIP/GeoIP.dat")
	if err != nil {
        CRIT(err.Error())
		return
	}

    counter := 0
    live := true
    loopCount := 1
    var lastPktTime *time.Time = nil
    for live {
        var pkt *pcap.Packet
        var res int32

        if *oneAtAtime {
            fmt.Println("Press enter to read packet")
            fmt.Scanln()
        }

        pkt, res = h.NextEx()
        switch res {
        case -1:
            ERR("pcap.NextEx() error: %s", h.Geterror())
            live = false
            continue
        case -2:
            DEBUG("pcapread", "End of file")
            loopCount += 1
            if *loop > 0 && loopCount > *loop {
                // give a bit of time to the publish goroutine
                // to flush
                time.Sleep(300 * time.Millisecond)
                live = false
            }
            // reopen the file
            h, err = pcap.Openoffline(*file)
            if h == nil {
                ERR("Openoffline(%s) failed: %s", *file, err)
                return
            }
            lastPktTime = nil
            continue
        case 0:
            // timeout
            continue
        }
        if res != 1 {
            panic(fmt.Sprintf("Unexpected return code from pcap.NextEx: %d", res))
        }

        if pkt == nil {
            panic("Nil packet despite res=1")
        }

        if *file != "" {
            if lastPktTime != nil {
                sleep := pkt.Time.Sub(*lastPktTime)
                if sleep > 0 {
                    time.Sleep(sleep)
                } else {
                    WARN("Time in pcap went back in time: %d", sleep)
                }
            }
            _lastPktTime := pkt.Time
            lastPktTime = &_lastPktTime
            pkt.Time = time.Now() // overwrite what we get from the pcap
        }
        counter++

        //DEBUG("pkt received")

        decodePktEth(datalink, pkt)
    }
    INFO("Input finish. Processed %d packets. Have a nice day!", counter)
}
