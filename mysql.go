package main

import (
    "bytes"
    "encoding/csv"
    "labix.org/v2/mgo/bson"
    "strings"
    "time"
)

// Packet types
const (
    MYSQL_CMD_QUERY = 3
)

type TcpTuple struct {
    Src_ip, Dst_ip     uint32
    Src_port, Dst_port uint16
    stream_id          uint32
}

type MysqlMessage struct {
    start int
    end   int

    Ts             time.Time
    Number         int
    IsRequest      bool
    FieldLength    uint32
    Seq            uint8
    Typ            uint8
    NumberOfRows   int
    NumberOfFields int
    Size           uint64
    Fields         []string
    Rows           [][]string
    Tables         string
    IsOK           bool
    AffectedRows   int
    InsertId       int
    IsError        bool
    ErrorCode      int
    ErrorInfo      string
    Query          string

    Stream_id    uint32
    Direction    uint8
    IsTruncated  bool
    Tuple        *IpPortTuple
    CmdlineTuple *CmdlineTuple
    Raw          []byte
}

type DbEndpoint struct {
    Ip      string
    Port    uint16
    Name    string
    Cmdline string
    Proc  string
}

type MysqlTransaction struct {
    Type         string
    tuple        TcpTuple
    Src          DbEndpoint
    Dst          DbEndpoint
    ResponseTime int32
    Ts           int64
    JsTs         time.Time
    ts           time.Time

    Mysql bson.M

    Request_raw  string
    Response_raw string

    timer *time.Timer
}

type MysqlStream struct {
    tcpStream *TcpStream

    data []byte

    parseOffset   int
    parseState    int
    bytesReceived int
    isClient      bool

    message *MysqlMessage
}

const (
    TransactionsHashSize = 2 ^ 16
    TransactionTimeout   = 10 * 1e9
)

const (
    MysqlStateStart = iota
    MysqlStateEatMessage
    MysqlStateEatFields
    MysqlStateEatRows
)

var mysqlTransactionsMap = make(map[TcpTuple]*MysqlTransaction, TransactionsHashSize)

func (stream *MysqlStream) PrepareForNewMessage() {
    stream.data = stream.data[stream.message.end:]
    stream.parseState = MysqlStateStart
    stream.parseOffset = 0
    stream.bytesReceived = 0
    stream.isClient = false
    stream.message = nil
}

func mysqlMessageParser(s *MysqlStream) (bool, bool) {

    DEBUG("mysqldetailed", "extract mysql message")
    m := s.message
    for s.parseOffset < len(s.data) {
        switch s.parseState {
        case MysqlStateStart:
            m.start = s.parseOffset
            if len(s.data[s.parseOffset:]) < 5 {
                DEBUG("mysql", "Message too short.")
                return false, false
            }
            hdr := s.data[s.parseOffset : s.parseOffset+5]
            m.FieldLength = uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16
            m.Seq = uint8(hdr[3])
            m.Typ = uint8(hdr[4])

            DEBUG("mysqldetailed", "Seq=%d, type=%d, start=%d", m.Seq, m.Typ, m.start)

            if m.Seq == 0 && m.Typ == MYSQL_CMD_QUERY {
                // parse request
                m.IsRequest = true
                m.start = s.parseOffset
                s.parseState = MysqlStateEatMessage
                s.parseOffset += 4
                s.bytesReceived = 0

                if !s.isClient {
                    s.isClient = true
                }
                m.Number = int(uint8(s.data[m.start+4]))

            } else if m.Seq == 1 && !s.isClient {
                // parse response
                m.IsRequest = false
                m.Number = int(uint8(s.data[m.start+3]))

                if uint8(hdr[4]) == 0x00 {
                    DEBUG("mysqldetailed", "OK response")
                    m.start = s.parseOffset
                    s.parseOffset += 4
                    s.parseState = MysqlStateEatMessage
                    s.bytesReceived = 0
                    m.IsOK = true
                } else if uint8(hdr[4]) == 0xff {
                    DEBUG("mysqldetailed", "Error response")
                    m.start = s.parseOffset
                    s.parseOffset += 4
                    s.parseState = MysqlStateEatMessage
                    m.IsError = true
                } else if m.FieldLength == 1 {
                    DEBUG("mysqldetailed", "Query response. Number of fields", uint8(hdr[4]))
                    m.NumberOfFields = int(hdr[4])
                    m.start = s.parseOffset
                    s.parseOffset += 5
                    s.parseState = MysqlStateEatFields
                    s.bytesReceived = 0
                } else {
                    // something else. ignore
                    DEBUG("mysqldetailed", "Unexpected message 1")
                    return false, false
                }

            } else {
                // something else. ignore
                DEBUG("mysqldetailed", "Unexpected message")
                return false, false
            }
            break

        case MysqlStateEatMessage:
            if len(s.data[s.parseOffset:]) >= int(m.FieldLength)-s.bytesReceived {
                s.parseOffset += (int(m.FieldLength) - s.bytesReceived)
                m.end = s.parseOffset
                if m.IsRequest {
                    m.Query = string(s.data[m.start+5 : m.end])
                } else if m.IsOK {
                    m.AffectedRows = int(s.data[m.start+5])
                    m.InsertId = int(s.data[m.start+6])
                } else if m.IsError {
                    m.ErrorCode = int(uint16(s.data[m.start+6])<<8 | uint16(s.data[m.start+7]))
                    m.ErrorInfo = string(s.data[m.start+9:m.start+14]) + ": " + string(s.data[m.start+15:])
                }
                return true, true
            } else {
                s.bytesReceived += (len(s.data) - s.parseOffset)
                s.parseOffset = len(s.data)
                return true, false
            }
            break

        case MysqlStateEatFields:
            if len(s.data[s.parseOffset:]) < 3 {
                return true, false
            }
            lensl := s.data[s.parseOffset : s.parseOffset+3]
            m.FieldLength = uint32(lensl[0]) | uint32(lensl[1])<<8 | uint32(lensl[2])<<16
            m.FieldLength += 4 // header

            if len(s.data[s.parseOffset:]) >= int(m.FieldLength)-s.bytesReceived {
                if uint8(s.data[s.parseOffset+4]) == 0xfe {
                    // EOF marker
                    s.parseOffset += (int(m.FieldLength) - s.bytesReceived)

                    s.parseState = MysqlStateEatRows
                    s.bytesReceived = 0
                } else {
                    _ /* catalog */, off := read_lstring(s.data, s.parseOffset+4)
                    db /*schema */, off := read_lstring(s.data, off)
                    table /* table */, off := read_lstring(s.data, off)

                    db_table := string(db) + "." + string(table)

                    if len(m.Tables) == 0 {
                        m.Tables = db_table
                    } else if !strings.Contains(m.Tables, db_table) {
                        m.Tables = m.Tables + ", " + db_table
                    }

                    s.parseOffset += (int(m.FieldLength) - s.bytesReceived)
                    // go to next field
                }
            } else {
                // wait for more
                return true, false
            }
            break

        case MysqlStateEatRows:
            if len(s.data[s.parseOffset:]) < 3 {
                return true, false
            }
            lensl := s.data[s.parseOffset : s.parseOffset+3]
            m.FieldLength = uint32(lensl[0]) | uint32(lensl[1])<<8 | uint32(lensl[2])<<16

            m.FieldLength += 4 //header
            if len(s.data[s.parseOffset:]) >= int(m.FieldLength)-s.bytesReceived {
                if uint8(s.data[s.parseOffset+4]) == 0xfe {
                    // EOF marker
                    s.parseOffset += (int(m.FieldLength) - s.bytesReceived)

                    if m.end == 0 {
                        m.end = s.parseOffset
                    } else {
                        m.IsTruncated = true
                    }
                    m.Size = uint64(s.parseOffset - m.start)
                    if !m.IsError {
                        // in case the reponse was sent successfully
                        m.IsOK = true
                    }
                    return true, true
                } else {
                    s.parseOffset += (int(m.FieldLength) - s.bytesReceived)
                    if m.end == 0 && s.parseOffset > MAX_PAYLOAD_SIZE {
                        // only send up to here, but read until the end
                        m.end = s.parseOffset
                    }
                    m.NumberOfRows += 1
                    // go to next row
                }
            } else {
                // wait for more
                return true, false
            }

            break
        }
    }

    return true, false
}

func ParseMysql(pkt *Packet, tcp *TcpStream, dir uint8) {
    if tcp.mysqlData[dir] == nil {
        tcp.mysqlData[dir] = &MysqlStream{
            tcpStream: tcp,
            data:      pkt.payload,
            message:   &MysqlMessage{Ts: pkt.ts},
        }
    } else {
        // concatenate bytes
        tcp.mysqlData[dir].data = append(tcp.mysqlData[dir].data, pkt.payload...)
    }

    stream := tcp.mysqlData[dir]
    if stream.message == nil {
        stream.message = &MysqlMessage{Ts: pkt.ts}
    }

    ok, complete := mysqlMessageParser(tcp.mysqlData[dir])
    if !ok {
        // drop this tcp stream. Will retry parsing with the next
        // segment in it
        tcp.mysqlData[dir] = nil
        WARN("Fail parsing MySQL message. Drop tcp stream. Try parsing with the next segment")
        return
    }

    if complete {
        // all ok, ship it
        msg := stream.data[stream.message.start:stream.message.end]

        // Publisher.PublishMysql(stream.message, tcp, dir, msg)
        handleMysql(stream.message, tcp, dir, msg)

        // and reset message
        stream.PrepareForNewMessage()
    }
}

func handleMysql(m *MysqlMessage, tcp *TcpStream,
    dir uint8, raw_msg []byte) {

    m.Stream_id = tcp.id
    m.Tuple = tcp.tuple
    m.Direction = dir
    m.CmdlineTuple = procWatcher.FindProcessesTuple(tcp.tuple)
    m.Raw = raw_msg

    if m.IsRequest {
        receivedMysqlRequest(m)
    } else {
        receivedMysqlResponse(m)
    }
}

func receivedMysqlRequest(msg *MysqlMessage) {

    // Add it to the HT
    tuple := TcpTuple{
        Src_ip:    msg.Tuple.Src_ip,
        Dst_ip:    msg.Tuple.Dst_ip,
        Src_port:  msg.Tuple.Src_port,
        Dst_port:  msg.Tuple.Dst_port,
        stream_id: msg.Stream_id,
    }

    trans := mysqlTransactionsMap[tuple]
    if trans != nil {
        if len(trans.Mysql) != 0 {
            WARN("Two requests without a Response. Dropping old request")
        }
    } else {
        trans = &MysqlTransaction{Type: "mysql", tuple: tuple}
        mysqlTransactionsMap[tuple] = trans
    }

    DEBUG("mysql", "Received request with tuple: %s", tuple)

    trans.ts = msg.Ts
    trans.Ts = int64(trans.ts.UnixNano() / 1000) // transactions have microseconds resolution
    trans.JsTs = msg.Ts
    trans.Src = DbEndpoint{
        Ip:     Ipv4_Ntoa(tuple.Src_ip),
        Port:   tuple.Src_port,
        Proc: string(msg.CmdlineTuple.Src),
    }
    trans.Dst = DbEndpoint{
        Ip:     Ipv4_Ntoa(tuple.Dst_ip),
        Port:   tuple.Dst_port,
        Proc: string(msg.CmdlineTuple.Dst),
    }

    index := strings.Index(msg.Query, " ")
    var method string
    if index > 0 {
        method = strings.ToUpper(msg.Query[:index])
    } else {
        method = strings.ToUpper(msg.Query)
    }

    trans.Mysql = bson.M{
        "query": msg.Query,
        "method": method,
    }

    // save Raw message
    trans.Request_raw = msg.Query

    if trans.timer != nil {
        trans.timer.Stop()
    }
    trans.timer = time.AfterFunc(TransactionTimeout, func() { trans.Expire() })
}

func receivedMysqlResponse(msg *MysqlMessage) {
    tuple := TcpTuple{
        Src_ip:    msg.Tuple.Src_ip,
        Dst_ip:    msg.Tuple.Dst_ip,
        Src_port:  msg.Tuple.Src_port,
        Dst_port:  msg.Tuple.Dst_port,
        stream_id: msg.Stream_id,
    }
    trans := mysqlTransactionsMap[tuple]
    if trans == nil {
        WARN("Response from unknown transaction. Ignoring.")
        return
    }
    // check if the request was received
    if len(trans.Mysql) == 0 {
        WARN("Response from unknown transaction. Ignoring.")
        return

    }
    // save json details
    trans.Mysql = bson_concat(trans.Mysql, bson.M{
        "isok":          msg.IsOK,
        "affected_rows": msg.AffectedRows,
        "insert_id":     msg.InsertId,
        "tables":        msg.Tables,
        "num_rows":      msg.NumberOfRows,
        "size":          msg.Size,
        "num_fields":    msg.NumberOfFields,
        "iserror":       msg.IsError,
        "error_code":    msg.ErrorCode,
        "error_message": msg.ErrorInfo,
    })

    trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6) // resp_time in milliseconds

    // save Raw message
    if len(msg.Raw) > 0 {
        fields, rows := parseMysqlResponse(msg.Raw)
        DEBUG("mysql", "fields: %d", len(fields))
        if len(rows) > 0 {
            DEBUG("mysql", "rows: %d", len(rows[0]))
        }

        trans.Response_raw = dumpInCSVFormat(fields, rows)
    }

    DEBUG("mysql", "response raw: {%s}", trans.Response_raw)

    err := Publisher.PublishMysqlTransaction(trans)
    if err != nil {
        WARN("Publish failure: %s", err)
    }

    DEBUG("mysql", "Mysql transaction completed: %s", trans.Mysql)

    // remove from map
    delete(mysqlTransactionsMap, trans.tuple)
    if trans.timer != nil {
        trans.timer.Stop()
    }
}

func (trans *MysqlTransaction) Expire() {
    // TODO: Here we need to PUBLISH an incomplete/timeout transaction
    // remove from map
    delete(mysqlTransactionsMap, trans.tuple)
}

func dumpInCSVFormat(fields []string, rows [][]string) string {

    var buf bytes.Buffer
    writer := csv.NewWriter(&buf)

    for i, field := range fields {
        fields[i] = strings.Replace(field, "\n", "\\n", -1)
    }
    if len(fields) > 0 {
        writer.Write(fields)
    }

    for _, row := range rows {
        for i, field := range row {
            field = strings.Replace(field, "\n", "\\n", -1)
            field = strings.Replace(field, "\r", "\\r", -1)
            row[i] = field
        }
        writer.Write(row)
    }
    writer.Flush()

    csv := buf.String()
    return csv
}

func parseMysqlResponse(data []byte) ([]string, [][]string) {

    length := read_length(data, 0)
    if length < 1 {
        WARN("Warning: Skipping empty Response")
        return []string{}, [][]string{}
    }

    fields := []string{}
    rows := [][]string{}

    if uint8(data[4]) == 0x00 {
        // OK response
    } else if uint8(data[4]) == 0xff {
        // Error response
    } else {
        offset := 5

        // Read fields
        for {
            length = read_length(data, offset)

            if uint8(data[offset+4]) == 0xfe {
                // EOF
                offset += length + 4
                break
            }

            _ /* catalog */, off := read_lstring(data, offset+4)
            _ /*database*/, off = read_lstring(data, off)
            _ /*table*/, off = read_lstring(data, off)
            _ /*org table*/, off = read_lstring(data, off)
            name, off := read_lstring(data, off)
            _ /* org name */, off = read_lstring(data, off)

            fields = append(fields, string(name))

            offset += length + 4
        }

        // Read rows
        for offset < len(data) {
            var row []string

            if uint8(data[offset+4]) == 0xfe {
                // EOF
                offset += length + 4
                break
            }

            length = read_length(data, offset)
            off := offset + 4 // skip length + packet number
            start := off
            for off < start+length {
                var text []byte

                if uint8(data[off]) == 0xfb {
                    text = []byte("NULL")
                    off++
                } else {
                    text, off = read_lstring(data, off)
                }

                row = append(row, string(text))
            }

            rows = append(rows, row)

            offset += length + 4
        }
    }
    return fields, rows
}
