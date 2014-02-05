package main

import (
    "fmt"
    "strings"
    "time"
)

const TCP_STREAM_EXPIRY = 10 * 1e9
const TCP_STREAM_HASH_SIZE = 2 ^ 16

type TcpStream struct {
    id       uint32
    tuple    *IpPortTuple
    timer    *time.Timer
    protocol protocolType

    httpData   [2]*HttpStream
    mysqlData  [2]*MysqlStream
    parserData [2]ParserStream
    redisData  [2]*RedisStream
}

type ParserStream interface {
    Parse() (bool, bool)
    AddData([]byte)
    Message() interface{}
    Reset()
}

var __id uint32 = 0

func GetId() uint32 {
    __id += 1
    return __id
}

const (
    TCP_FLAG_FIN = 0x01
    TCP_FLAG_SYN = 0x02
    TCP_FLAG_RST = 0x04
    TCP_FLAG_PSH = 0x08
    TCP_FLAG_ACK = 0x10
    TCP_FLAG_URG = 0x20
)

// Config
type tomlProtocol struct {
    Ports         []int
    Send_request  bool
    Send_response bool
}

var tcpStreamsMap = make(map[IpPortTuple]*TcpStream, TCP_STREAM_HASH_SIZE)
var tcpPortMap map[uint16]protocolType

func decideProtocol(tuple *IpPortTuple) protocolType {
    protocol, exists := tcpPortMap[tuple.Src_port]
    if exists {
        return protocol
    }

    protocol, exists = tcpPortMap[tuple.Dst_port]
    if exists {
        return protocol
    }

    return UnknownProtocol
}

func (stream *TcpStream) AddPacket(pkt *Packet, flags uint8, original_dir uint8) {
    //DEBUG(" (tcp stream %d[%d])", stream.id, original_dir)

    // create/reset timer
    if stream.timer != nil {
        stream.timer.Stop()
    }
    stream.timer = time.AfterFunc(TCP_STREAM_EXPIRY, func() { stream.Expire() })

    // call upper layer
    if len(pkt.payload) == 0 && stream.protocol == HttpProtocol {
        if flags&TCP_FLAG_FIN != 0 {
            HttpReceivedFin(stream, original_dir)
        }
        return
    }
    switch stream.protocol {
    case HttpProtocol:
        ParseHttp(pkt, stream, original_dir)

        if flags&TCP_FLAG_FIN != 0 {
            HttpReceivedFin(stream, original_dir)
        }
        break
    case MysqlProtocol:
        ParseMysql(pkt, stream, original_dir)
        break

    case RedisProtocol:
        ParseRedis(pkt, stream, original_dir)
        break
    }
}

func (stream *TcpStream) Expire() {

    // de-register from dict
    delete(tcpStreamsMap, *stream.tuple)
}

func FollowTcp(tcphdr []byte, pkt *Packet) {
    stream, exists := tcpStreamsMap[pkt.tuple]
    var original_dir uint8 = 1
    if !exists {
        // search also the other direction
        rev_tuple := IpPortTuple{Src_ip: pkt.tuple.Dst_ip, Dst_ip: pkt.tuple.Src_ip,
            Src_port: pkt.tuple.Dst_port, Dst_port: pkt.tuple.Src_port}

        stream, exists = tcpStreamsMap[rev_tuple]
        if !exists {
            protocol := decideProtocol(&pkt.tuple)
            if protocol == UnknownProtocol {
                // don't follow
                return
            }

            // create
            stream = &TcpStream{id: GetId(), tuple: &pkt.tuple, protocol: protocol}
            tcpStreamsMap[pkt.tuple] = stream
        } else {
            original_dir = 0
        }
    }
    flags := uint8(tcphdr[13])
    stream.AddPacket(pkt, flags, original_dir)
}

func PrintTcpMap() {
    fmt.Printf("Streams in memory:")
    for _, stream := range tcpStreamsMap {
        fmt.Printf(" %d", stream.id)
    }
    fmt.Printf("\n")
}

func configToPortsMap(config *tomlConfig) map[uint16]protocolType {
    var res = map[uint16]protocolType{}

    var proto protocolType
    for proto = UnknownProtocol + 1; int(proto) < len(protocolNames); proto++ {

        protoConfig, exists := config.Protocols[protocolNames[proto]]
        if !exists {
            // skip
            continue
        }

        for _, port := range protoConfig.Ports {
            res[uint16(port)] = proto
        }
    }

    return res
}

func configToFilter(config *tomlConfig) string {

    res := []string{}

    for _, protoConfig := range config.Protocols {
        for _, port := range protoConfig.Ports {
            res = append(res, fmt.Sprintf("port %d", port))
        }
    }

    return strings.Join(res, " or ")
}

func TcpInit() error {
    tcpPortMap = configToPortsMap(&_Config)

    return nil
}
