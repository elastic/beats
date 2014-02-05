package main

import (
    "time"
)

const MAX_PAYLOAD_SIZE = 100 * 1024

// replacement for time.Time when gobbing
type MsgTime struct {
    Sec  int64
    Nsec int32
}

type MsgHeader struct {
    Ts  MsgTime
    Typ uint16
}

const VERSION = 1

const (
    MSG_TYPE_NOP = iota
    MSG_TYPE_HELLO
    MSG_TYPE_HELLO_RESP
    MSG_TYPE_HTTP
    MSG_TYPE_MYSQL // Now deprecated
    MSG_TYPE_REPORT
    MSG_TYPE_REDIS
    MSG_TYPE_MYSQL_TRANSACTION
    MSG_TYPE_HTTP_TRANSACTION
    MSG_TYPE_REDIS_TRANSACTION
)

type IpPortTuple struct {
    Src_ip, Dst_ip     uint32
    Src_port, Dst_port uint16
}

type CmdlineTuple struct {
    Src, Dst []byte
}

type HelloMessage struct {
    Version uint16
    Flags   uint16
    Authstr []byte // must be 40 bytes
    Name    []byte
}

// Agent connection flags
const (
    AGENT_USE_TLS = 1 << iota
)

type HelloRespMessage struct {
    Flags         uint16
    Response_code uint8
    Msg           []byte
}

const HTTP_VERSION = 1

const (
    HTTP_FLAGS_DIR_INITIAL = 1 << iota
    HTTP_FLAGS_IS_REQUEST
)

const MYSQL_VERSION = 1

const (
    MYSQL_FLAGS_DIR_INITIAL = 1 << iota
    MYSQL_FLAGS_IS_REQUEST
    MYSQL_FLAGS_IS_TRUNCATED
)

const REDIS_VERSION = 1

const (
    REDIS_FLAGS_DIR_INITIAL = 1 << iota
    REDIS_FLAGS_IS_REQUEST
)

type PublishMessage struct {
    Version      uint16
    Stream_id    uint32
    Tuple        *IpPortTuple
    CmdlineTuple *CmdlineTuple
    Flags        uint16
    Data         []byte
    Raw          []byte
}

type PublishTransaction struct {
    Data []byte
}

type ReportMessage struct {
    Msg []byte
}

func NewMsgTime(ts time.Time) MsgTime {
    return MsgTime{
        Sec:  ts.Unix(),
        Nsec: int32(ts.Nanosecond()),
    }
}

func (msgtime *MsgTime) Time() time.Time {
    return time.Unix(msgtime.Sec, int64(msgtime.Nsec)).UTC()
}

func read_lstring(data []byte, offset int) ([]byte, int) {
    length, off := read_linteger(data, offset)
    return data[off : off+int(length)], off + int(length)
}
func read_linteger(data []byte, offset int) (uint64, int) {
    switch uint8(data[offset]) {
    case 0xfe:
        return uint64(data[offset+1]) | uint64(data[offset+2])<<8 |
                uint64(data[offset+2])<<16 | uint64(data[offset+3])<<24 |
                uint64(data[offset+4])<<32 | uint64(data[offset+5])<<40 |
                uint64(data[offset+6])<<48 | uint64(data[offset+7])<<56,
            offset + 9
    case 0xfd:
        return uint64(data[offset+1]) | uint64(data[offset+2])<<8 |
            uint64(data[offset+2])<<16, offset + 4
    case 0xfc:
        return uint64(data[offset+1]) | uint64(data[offset+2])<<8, offset + 3
    }

    if uint64(data[offset]) >= 0xfb {
        panic("Unexpected value in read_linteger")
    }

    return uint64(data[offset]), offset + 1
}

func read_length(data []byte, offset int) int {
    length := uint32(data[offset]) |
        uint32(data[offset+1])<<8 |
        uint32(data[offset+2])<<16
    return int(length)
}
