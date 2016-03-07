package drda

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"time"

	"github.com/Intermernet/ebcdic"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
	"github.com/urso/ucfg"
)

/*

Parse DRDA Protocol used by IBM DB2, Informix or Apache Derby (and possibly other databases)

Limitations:
- Raw message not supported (guess that makes not much sense here)
- No unit tests yet
- Too few system tests
- Check that all codepoints are covered (also "esoteric" like those from IMS)

Dependencies:
- https://github.com/Intermernet/ebcdic

*/

type parseState int

const (
	drdaStateDDM parseState = iota
	drdaStateParameter
)

var stateStrings []string = []string{
	"DDM",
	"Parameter",
}

func drdaAbbrev(codepoint uint16) string {
	abbrev := drda_abbrev[codepoint]

	if abbrev == "" {
		return fmt.Sprint("unknown_", codepoint)
	}

	return abbrev
}

func (state parseState) String() string {
	return stateStrings[state]
}

func (drda *Drda) getTransaction(k common.HashableTcpTuple) *DrdaTransaction {
	v := drda.transactions.Get(k)
	if v != nil {
		return v.(*DrdaTransaction)
	}
	return nil
}

func init() {
	protos.Register("drda", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *ucfg.Config,
) (protos.Plugin, error) {
	p := &Drda{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, &config); err != nil {
		return nil, err
	}
	return p, nil
}

func (drda *Drda) init(results publish.Transactions, config *drdaConfig) error {
	drda.setFromConfig(config)

	drda.transactions = common.NewCache(
		drda.transactionTimeout,
		protos.DefaultTransactionHashSize)
	drda.transactions.StartJanitor(drda.transactionTimeout)
	//drda.handleMysql = handleMysql
	drda.results = results

	return nil
}

func (drda *Drda) setFromConfig(config *drdaConfig) error {

	drda.Ports = config.Ports
	drda.Send_request = config.SendRequest
	drda.Send_response = config.SendResponse
	drda.transactionTimeout = time.Duration(config.TransactionTimeout) * time.Second

	return nil

}

func (drda *Drda) GetPorts() []int {
	return drda.Ports
}

func (stream *DrdaStream) PrepareForNewMessage() {
	stream.data = stream.data[stream.parseOffset:]
	stream.parseState = drdaStateDDM
	stream.parseOffset = 0
	stream.message = nil
}

//main loop
//return: ok, complete
func drdaMessageParser(s *DrdaStream) (bool, bool) {

	m := s.message
	m.parameters = make(map[uint16]Parameter)
	for s.parseOffset < len(s.data) {

		direction := ""

		if m.Direction == 0 {
			direction = "Response"
		} else {
			direction = "Request"
		}

		logp.Debug("drdadetailed", "Direction %s", direction)
		logp.Debug("drdadetailed", "parser round with parseState = %s and offset: %d, len of data is %d", s.parseState, s.parseOffset, len(s.data))

		switch s.parseState {
		case drdaStateDDM:

			m.start = s.parseOffset
			if len(s.data[s.parseOffset:]) < 10 {
				logp.Err("DRDA DDM Message too short. Ignore it.")
				return false, false
			}

			hdr := s.data[s.parseOffset : s.parseOffset+10]
			if hdr[2] != DRDA_MAGIC {
				logp.Err("No DRDA magic byte found (%X) but %X", DRDA_MAGIC, uint8(hdr[2]))
				return false, false
			}

			if m.ddm.Length != 0 {
				logp.Err("DDM already initialized.")
			}

			if m.RemainingLength != 0 {
				logp.Err("Remaining length must be 0.")
			}

			ddm := &Ddm{}

			ddm.Length = uint16(hdr[0])<<8 | uint16(hdr[1])
			ddm.Format = uint8(hdr[3])
			ddm.DSSType = ddm.Format & 0x0F
			ddm.DSSFlags = ddm.Format >> 4
			ddm.Cor = uint16(hdr[4])<<8 | uint16(hdr[5])
			ddm.Length2 = uint16(hdr[6])<<8 | uint16(hdr[7])
			ddm.Codepoint = uint16(hdr[8])<<8 | uint16(hdr[9])
			m.ddm = *ddm

			m.end = int(ddm.Length)
			m.RemainingLength = int(ddm.Length) - 10

			logp.Debug("drdadetailed", ">>>> DRDA DDM: Length %d, codepoint %s", ddm.Length, drdaAbbrev(ddm.Codepoint))
			s.parseOffset += 10

			if ddm.Length > 10 {
				s.parseState = drdaStateParameter
				continue
			} else {
				logp.Debug("drdadetailed", "       - No parameters")
				return true, true
			}

		case drdaStateParameter:

			if len(s.data[s.parseOffset:]) < 4 {
				logp.Err("Parameters message too short. Ignore it.")
				return false, false
			}

			codePoint := uint16(s.data[s.parseOffset+2])<<8 | uint16(s.data[s.parseOffset+3])
			parameterLength := uint16(s.data[s.parseOffset])<<8 | uint16(s.data[s.parseOffset+1])

			if parameterLength == 0 || parameterLength == 1 || int(parameterLength) > m.RemainingLength {
				logp.Debug("drdadetailed", "       - Parameter %s with special length, thats ok but assume last parameter", drdaAbbrev(codePoint))
				parameterLength = uint16(m.RemainingLength)
			} else {
				logp.Debug("drdadetailed", "       - Parameter: Length %d %s (%s)", parameterLength, drdaAbbrev(codePoint), drda_description[codePoint])
			}

			dataLength := int(parameterLength) - 4

			parameter := &Parameter{}
			parameter.Length = parameterLength
			parameter.Codepoint = codePoint

			var data []byte

			if dataLength > 0 {

				data = s.data[s.parseOffset+4 : s.parseOffset+4+dataLength]
				parameter.ASCIIData = string(data)
				parameter.EBCDICData = string(ebcdic.Decode(data))
			}

			m.parameters[codePoint] = *parameter
			m.RemainingLength -= int(parameterLength)
			s.parseOffset += int(parameterLength)

			if m.RemainingLength <= 0 {
				s.parseState = drdaStateDDM
				return true, true
			}

			break

		} //end switch
	} //end for

	return true, false
}

type drdaPrivateData struct {
	Data [2]*DrdaStream
}

func (drda *Drda) ConnectionTimeout() time.Duration {
	return drda.transactionTimeout
}

//entry point
func (drda *Drda) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {

	trans := drda.getTransaction(tcptuple.Hashable())

	if dir == 1 {

		if trans != nil {
			logp.Err("transaction should be nil for request")
		}

		trans = &DrdaTransaction{Type: "drda", tuple: *tcptuple, TsStart: pkt.Ts}
		drda.transactions.Put(tcptuple.Hashable(), trans)
		logp.Debug("drdadetailed", "Initialize transaction")

	} else {
		if trans == nil {
			logp.Err("transaction should be not nil for response")
		}
	}

	//dir == 1 request
	//dir == 0 response

	//relevant tcp packet

	defer logp.Recover("ParseDrda exception")

	priv := drdaPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(drdaPrivateData)
		if !ok {
			priv = drdaPrivateData{}
		}
	}

	if priv.Data[dir] == nil {
		priv.Data[dir] = &DrdaStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &DrdaMessage{},
		}
	} else {
		// concatenate bytes
		priv.Data[dir].data = append(priv.Data[dir].data, pkt.Payload...)
		if len(priv.Data[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			logp.Debug("drda", "Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}

	completed := true

	stream := priv.Data[dir]
	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &DrdaMessage{}
		}

		stream.message.Direction = dir

		ok, complete := drdaMessageParser(priv.Data[dir])
		//logp.Debug("drdadetailed", "drdaMessageParser returned ok=%b complete=%b", ok, complete)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.Data[dir] = nil
			logp.Debug("drdadetailed", "Ignore DRDA message. Drop tcp stream. Try parsing with the next segment")
			return priv
		}

		if complete {

			stream.message.TcpTuple = *tcptuple
			stream.message.Direction = dir
			stream.message.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IpPort())

			if stream.message.Direction == 1 {
				drda.receivedDrdaRequest(stream.message)
			} else {
				drda.receivedDrdaResponse(stream.message)
			}

			// and reset message
			stream.PrepareForNewMessage()
		} else {
			// wait for more data
			completed = false
			break
		}
	}

	if completed {
		logp.Debug("drdadetailed", "Packet with direction %d finished complete", dir)

		if dir == 0 {
			trans.TsEnd = pkt.Ts
			drda.publishTransaction(trans)
			drda.transactions.Delete(trans.tuple.Hashable())

			logp.Debug("drda", "Drda transaction completed: %s", trans.Requests)
		}

	} else {
		logp.Debug("drdadetailed", "Packet with direction %d finished incomplete", dir)
	}

	return priv
}

func (drda *Drda) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

	/*defer logp.Recover("GapInStream(drda) exception")

	if private == nil {
		return private, false
	}
	drdaData, ok := private.(drdaPrivateData)
	if !ok {
		return private, false
	}
	stream := drdaData.Data[dir]
	if stream == nil || stream.message == nil {
		// nothing to do
		return private, false
	}

	if drda.messageGap(stream, nbytes) {
		// we need to publish from here
		drda.messageComplete(tcptuple, dir, stream)
	}

	// we always drop the TCP stream. Because it's binary and len based,
	// there are too few cases in which we could recover the stream (maybe
	// for very large blobs, leaving that as TODO)
	*/

	//TODO: handle GapInStream()

	logp.Err("Unhandled gap of %d bytes in TCP stream", nbytes)

	return private, true
}

func (drda *Drda) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {

	// TODO: check if we have data pending and either drop it to free
	// memory or send it up the stack.
	return private
}

func (drda *Drda) receivedDrdaRequest(msg *DrdaMessage) {
	tuple := msg.TcpTuple
	trans := drda.getTransaction(tuple.Hashable())

	if trans == nil {
		logp.Err("No transaction for this request")
	}

	trans.Src = common.Endpoint{
		Ip:   msg.TcpTuple.Src_ip.String(),
		Port: msg.TcpTuple.Src_port,
		Proc: string(msg.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   msg.TcpTuple.Dst_ip.String(),
		Port: msg.TcpTuple.Dst_port,
		Proc: string(msg.CmdlineTuple.Dst),
	}
	if msg.Direction == tcp.TcpDirectionReverse {
		trans.Src, trans.Dst = trans.Dst, trans.Src
	}

	if trans.Requests == nil {
		trans.Requests = common.MapStr{}
	}

	tmp := common.MapStr{}

	for key, value := range msg.parameters {

		p := common.MapStr{}
		p["desc"] = drda_description[key]
		p["codepoint"] = value.Codepoint
		p["length"] = value.Length
		p["data_ascii"] = value.ASCIIData
		p["data_ebcdic"] = value.EBCDICData
		tmp[drdaAbbrev(key)] = p
	}

	if val, ok := msg.parameters[DRDA_CP_SQLSTT]; ok {
		trans.Query = val.ASCIIData
	}

	trans.Requests[fmt.Sprint(drdaAbbrev(msg.ddm.Codepoint), "_", msg.ddm.Cor)] = common.MapStr{

		"description":  drda_description[msg.ddm.Codepoint],
		"codepoint":    msg.ddm.Codepoint,
		"length2":      msg.ddm.Length2,
		"format":       msg.ddm.Format,
		"format_flags": msg.ddm.DSSFlags,
		//"format_reserved":        msg.ddm.DSSFlags >> DRDA_DSSFMT_RESERVED,
		//"format_chained":         msg.ddm.DSSFlags >> DRDA_DSSFMT_CHAINED,
		//"format_continue":        msg.ddm.DSSFlags >> DRDA_DSSFMT_CONTINUE,
		//"format_samecor":         msg.ddm.DSSFlags >> DRDA_DSSFMT_SAME_CORR,
		"format_dss_type":        msg.ddm.DSSType,
		"format_dss_type_abbrev": dss_abbrev[uint16(msg.ddm.DSSType)],
		"correlation_id":         msg.ddm.Cor,
		"length":                 msg.ddm.Length,
		"direction":              msg.Direction,
		"parameters":             tmp,
	}

	trans.Notes = msg.Notes
	trans.BytesIn += uint64(msg.ddm.Length)
}

func (drda *Drda) receivedDrdaResponse(msg *DrdaMessage) {
	trans := drda.getTransaction(msg.TcpTuple.Hashable())

	if trans == nil {
		logp.Err("No transaction for this response")
	}

	if trans.Responses == nil {
		trans.Responses = common.MapStr{}
	}

	tmp := common.MapStr{}

	for key, value := range msg.parameters {

		p := common.MapStr{}
		p["desc"] = drda_description[key]
		p["codepoint"] = value.Codepoint
		p["length"] = value.Length
		p["data_ascii"] = value.ASCIIData
		p["data_ebcdic"] = value.EBCDICData

		tmp[drdaAbbrev(key)] = p
	}

	trans.Responses[fmt.Sprint(drdaAbbrev(msg.ddm.Codepoint), "_", msg.ddm.Cor)] = common.MapStr{

		"description":  drda_description[msg.ddm.Codepoint],
		"codepoint":    msg.ddm.Codepoint,
		"length2":      msg.ddm.Length2,
		"format":       msg.ddm.Format,
		"format_flags": msg.ddm.DSSFlags,
		//"format_reserved":        msg.ddm.DSSFlags >> DRDA_DSSFMT_RESERVED,
		//"format_chained":         msg.ddm.DSSFlags >> DRDA_DSSFMT_CHAINED,
		//"format_continue":        msg.ddm.DSSFlags >> DRDA_DSSFMT_CONTINUE,
		//"format_samecor":         msg.ddm.DSSFlags >> DRDA_DSSFMT_SAME_CORR,
		"format_dss_type":        msg.ddm.DSSType,
		"format_dss_type_abbrev": dss_abbrev[uint16(msg.ddm.DSSType)],
		"correlation_id":         msg.ddm.Cor,
		"length":                 msg.ddm.Length,
		//"direction": msg.Direction,
		"parameters": tmp,
	}

	trans.BytesOut += uint64(msg.ddm.Length)
	trans.Notes = append(trans.Notes, msg.Notes...)
}

func (drda *Drda) publishTransaction(t *DrdaTransaction) {

	if drda.results == nil {
		logp.Err("Nothing to publish")
		return
	}

	t.ResponseTime = int32(t.TsEnd.Sub(t.TsStart).Nanoseconds() / 1e6) // resp_time in milliseconds

	event := common.MapStr{}
	event["type"] = "drda"

	event["responsetime"] = t.ResponseTime
	if drda.Send_request {
		event["request"] = "n.a." //t.Request_raw
	}
	if drda.Send_response {
		event["response"] = "n.a." //t.Response_raw
	}

	event["query"] = t.Query

	event["status"] = common.OK_STATUS

	drdaMap := common.MapStr{}

	drdaMap["requests"] = t.Requests
	drdaMap["responses"] = t.Responses
	event["drda"] = drdaMap

	event["bytes_out"] = t.BytesOut
	event["bytes_in"] = t.BytesIn

	if len(t.Notes) > 0 {
		event["notes"] = t.Notes
	}

	event["@timestamp"] = common.Time(t.TsStart)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	logp.Debug("drda", "Transaction published")

	drda.results.PublishTransaction(event)
}
