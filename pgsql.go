package main

import (
    "labix.org/v2/mgo/bson"
    "time"
)

// Packet types
const (
)

type PgsqlMessage struct {
    start int
    end   int

    Ts             time.Time
    IsRequest      bool
    Query          string
    Size           uint64
    Fields         []string
    Rows           [][]string
    NumberOfRows   int
    NumberOfFields int
    IsOK           bool
    IsError        bool


    Stream_id    uint32
    Direction    uint8
    Tuple        *IpPortTuple
    CmdlineTuple *CmdlineTuple
    Raw          []byte
}

type PgsqlTransaction struct {
    Type         string
    tuple        TcpTuple
    Src          Endpoint
    Dst          Endpoint
    ResponseTime int32
    Ts           int64
    JsTs         time.Time
    ts           time.Time


    Mysql bson.M

    Request_raw  string
    Response_raw string

    timer *time.Timer
}

type PgsqlStream struct {
    tcpStream *TcpStream

    data []byte

    parseOffset   int
    parseState    int
    bytesReceived int
    isClient      bool

    message       *PgsqlMessage
}

const (
    ConnectionCycle = iota
    SimpleQueryCycle
)

var PgsqlCommands = map[byte]string {
    'R': "Authentication response",
    'K': "BackendKeyData",
    'B': "Bind",
    '2': "BindComplete",
    'C': "Close or CommandComplete",
    '3': "CloseComplete",
    'd': "CopyData",
    'c': "CopyDone",
    'f': "CopyFail",
    'G': "CopyInResponse",
    'H': "CopyOutResponse or Flush",
    'W': "CopyBothResponse",
    'D': "DataRow or Descrive",
    'I': "EmptyQueryResponse",
    'E': "ErrorResponse or Execute",
    'F': "FunctionCall",
    'V': "FunctionCallResponse",
    'n': "NoData",
    'N': "NoticeResponse",
    'A': "NotificationResponse",
    't' :"ParameterDescription",
    'S': "ParameterStatus or Sync",
    'P': "Parse",
    '1': "ParseComplete",
    'p': "PasswordMessage",
    's': "PortalSuspended",
    'Q': "Query",
    'Z': "ReadyForQuery",
    'T': "RowDescription",
    'X': "Terminate",
}
var pgsqlTransactionsMap = make(map[TcpTuple]*PgsqlTransaction, TransactionsHashSize)

func isPgsqlCommand(c byte) bool {
    _, exists := PgsqlCommands[c]
    return exists
}

func (stream *PgsqlStream) PrepareForNewMessage() {
    stream.data = stream.data[stream.message.end:]
    stream.parseState = ConnectionCycle
    stream.parseOffset = 0
    stream.bytesReceived = 0
    stream.isClient = false
    stream.message = nil
}

func pgsqlFieldsParser(s *PgsqlStream) {
    // read field count (int16)
    field_count := Bytes_Ntohs(s.data[s.parseOffset:s.parseOffset+2])
    s.parseOffset += 2
    DEBUG("pgsql", "Row Description field count=%d", field_count)

    for i := 0; i < int(field_count); i++ {
        // read field name (null terminated string)
        field_name, err := readString(s.data[s.parseOffset:])
        if err != nil {
            ERR("Fail to read the column field")
        }
        s.parseOffset += len(field_name) + 1

        // read Table OID (int32)
        table_oid := int32(Bytes_Ntohl(s.data[s.parseOffset:s.parseOffset+4]))
        s.parseOffset += 4

        // read Column Index (int16)
        column_index := int16(Bytes_Ntohs(s.data[s.parseOffset:s.parseOffset+2]))
        s.parseOffset += 2

        // read Type OID (int32)
        type_oid := int32(Bytes_Ntohl(s.data[s.parseOffset:s.parseOffset+4]))
        s.parseOffset += 4

        // read column length (int16)
        column_length := int16(Bytes_Ntohs(s.data[s.parseOffset:s.parseOffset+2]))
        s.parseOffset += 2

        // read type modifier (int32)
        type_modif := int32(Bytes_Ntohl(s.data[s.parseOffset:s.parseOffset+4]))
        s.parseOffset += 4

        // read format (int16)
        format := Bytes_Ntohs(s.data[s.parseOffset:s.parseOffset+2])
        s.parseOffset += 2

        DEBUG("pgsql", "Field name=%s table oid=%d column_index=%d, type_oid=%d, column_length=%d, type_modif=%d, format=%d", field_name, table_oid, column_index, type_oid, column_length, type_modif, format)
    }
}

func pgsqlRowsParser(s *PgsqlStream) {
    // read field count (int16)
    field_count := Bytes_Ntohs(s.data[s.parseOffset:s.parseOffset+2])
    s.parseOffset += 2
    DEBUG("pgsql", "DataRow field count=%d", field_count)

    for i := 0; i < int(field_count); i++ {

        // read column length (int32)
        column_length := int32(Bytes_Ntohl(s.data[s.parseOffset:s.parseOffset+4]))
        s.parseOffset += 4

        // read column value (byten)
        column_value := []byte{}

        if column_length > 0 {
            column_value = s.data[s.parseOffset:s.parseOffset+ int(column_length)]
            s.parseOffset += int(column_length)
        } else if column_length == -1 {
            column_value = nil
        }

        DEBUG("pgsql", "Field length=%d, value=%s", column_length, string(column_value))
    }
}

func pgsqlErrorParser(s *PgsqlStream) {

    // read field type(byte1)
    field_type := s.data[s.parseOffset]
    s.parseOffset += 1

    // read field value(string)
    field_value, err := readString(s.data[s.parseOffset:])
    if err != nil {
        ERR("Fail to read the column field")
    }

    DEBUG("pgsql", "Error type=%c, message=%s", field_type, field_value)
}

func pgsqlMessageParser(s *PgsqlStream) (bool, bool) {

    m := s.message
    for s.parseOffset < len(s.data) {
        switch s.parseState {
            case ConnectionCycle:
                if len(s.data[s.parseOffset:]) < 5 {
                    WARN("Postgresql Message too short (length=%d)", len(s.data[s.parseOffset:]))
                    return true, false
                }
                // read type
                typ := byte(s.data[s.parseOffset])

                // check command type
                if isPgsqlCommand(typ) {
                    DEBUG("pgsql", "Pgsql type %c", typ)
                    s.parseOffset += 1

                    if typ == 'Z' {
                        // ReadyForQuery
                        s.parseState = SimpleQueryCycle
                    }
                } else {
                    // Startup Message, SSL Message, Cancel Request
                    DEBUG("pgsql", "Startup command")
                }
                // read message length
                length := Bytes_Ntohl(s.data[s.parseOffset:s.parseOffset+4])
                if len(s.data[s.parseOffset:]) >= int(length) {
                    s.parseOffset += int(length)
                } else {
                    // wait for more
                    return true, false
                }

                break
            case SimpleQueryCycle:
                if len(s.data[s.parseOffset:]) < 5 {
                    WARN("Postgresql Message too short (length=%d)", len(s.data[s.parseOffset:]))
                    return true, false
                }

                // read type
                typ := byte(s.data[s.parseOffset])
                s.parseOffset += 1

                // read message length
                length := Bytes_Ntohl(s.data[s.parseOffset:s.parseOffset+4])
                DEBUG("pgsql", "SimpleQueryCycle type=%c, length=%d", typ, length)

                if typ == 'Q' {
                    // Simple Query
                    if !s.isClient {
                        s.isClient = true
                    }
                    m.start = s.parseOffset
                    m.IsRequest = true
                    if len(s.data[s.parseOffset:]) >= int(length) {
                        s.parseOffset += int(length)
                        m.end = s.parseOffset
                        m.Query = string(s.data[m.start+5 : m.end])
                        DEBUG("pgsql", "Simple Query", "%s", m.Query)
                    }
                    return true, true
                } else if typ == 'T' {
                    // RowDescription

                    m.start = s.parseOffset
                    m.IsRequest = false
                    if len(s.data[s.parseOffset:]) > int(length) {
                        s.parseOffset += 4
                        pgsqlFieldsParser(s)
                    } else {
                        // wait for more
                        return true, false
                    }
                } else if typ == 'D' {
                    // DataRow
                    if len(s.data[s.parseOffset:]) > int(length) {
                        s.parseOffset += 4
                        pgsqlRowsParser(s)

                        m.end = s.parseOffset
                        m.Size = uint64(m.end - m.start)

                        return true, true
                    } else {
                        // wait for more
                        return true, false
                    }

                } else if typ == 'I' {
                    // EmptyQueryResponse
                    DEBUG("pgsql", "EmptyQueryResponse")
                    m.start = s.parseOffset
                    m.IsOK = true
                    m.IsRequest = false
                    s.parseOffset += 4 // length

                    return true, true

                } else if typ == 'E' {
                    // ErrorResponse
                    DEBUG("pgsql", "ErrorResponse")
                    m.start = s.parseOffset
                    m.IsRequest = false
                    m.IsError = true

                    if len(s.data[s.parseOffset:]) > int(length) {
                        s.parseOffset += 4 //length
                        pgsqlErrorParser(s)

                        m.end = s.parseOffset

                        return true, true
                    } else {
                        // wait for more
                        return true, false
                    }
                } else if typ == 'C' {
                    // CommandComplete
                    DEBUG("pgsql", "CommandComplete")
                    if len(s.data[s.parseOffset:]) > int(length) {
                        s.parseOffset += int(length)
                    } else {
                        // wait for more
                        return true, false
                    }
                    s.parseState =  ConnectionCycle
                } else {
                    // skip command
                    if len(s.data[s.parseOffset:]) > int(length) {
                        s.parseOffset += int(length)
                    } else {
                        // wait for more
                        return true, false
                    }
                }
                break
        }
    }

    return true, false
}

func ParsePgsql(pkt *Packet, tcp *TcpStream, dir uint8) {

    defer RECOVER("ParsePgsql exception")

    if tcp.pgsqlData[dir] == nil {
        tcp.pgsqlData[dir] = &PgsqlStream{
            tcpStream: tcp,
            data:      pkt.payload,
            message:   &PgsqlMessage{Ts: pkt.ts},
        }
    } else {
        // concatenate bytes
        tcp.pgsqlData[dir].data = append(tcp.pgsqlData[dir].data, pkt.payload...)
    }

    stream := tcp.pgsqlData[dir]
    if stream.message == nil {
        stream.message = &PgsqlMessage{Ts: pkt.ts}
    }

    ok, complete := pgsqlMessageParser(tcp.pgsqlData[dir])
    if !ok {
        // drop this tcp stream. Will retry parsing with the next
        // segment in it
        tcp.pgsqlData[dir] = nil
        DEBUG("pgsql", "Ignore Postgresql message. Drop tcp stream. Try parsing with the next segment")
        return
    }

    if complete {
        // all ok, ship it
        msg := stream.data[stream.message.start:stream.message.end]

        handlePgsql(stream.message, tcp, dir, msg)

        // and reset message
        stream.PrepareForNewMessage()
    }
}

func handlePgsql(m *PgsqlMessage, tcp *TcpStream,
    dir uint8, raw_msg []byte) {

    m.Stream_id = tcp.id
    m.Tuple = tcp.tuple
    m.Direction = dir
    m.CmdlineTuple = procWatcher.FindProcessesTuple(tcp.tuple)
    m.Raw = raw_msg

    if m.IsRequest {
        receivedPgsqlRequest(m)
    } else {
        receivedPgsqlResponse(m)
    }
}

func receivedPgsqlRequest(msg *PgsqlMessage) {

}

func receivedPgsqlResponse(msg *PgsqlMessage) {
}

func (trans *PgsqlTransaction) Expire() {
    // TODO: Here we need to PUBLISH an incomplete/timeout transaction
    // remove from map
    delete(pgsqlTransactionsMap, trans.tuple)
}
