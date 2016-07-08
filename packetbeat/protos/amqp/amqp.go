package amqp

import (
	"expvar"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

var (
	debugf    = logp.MakeDebug("amqp")
	detailedf = logp.MakeDebug("amqpdetailed")
)

type Amqp struct {
	Ports                     []int
	SendRequest               bool
	SendResponse              bool
	MaxBodyLength             int
	ParseHeaders              bool
	ParseArguments            bool
	HideConnectionInformation bool
	transactions              *common.Cache
	transactionTimeout        time.Duration
	results                   publish.Transactions

	//map containing functions associated with different method numbers
	MethodMap map[codeClass]map[codeMethod]AmqpMethod
}

var (
	unmatchedRequests  = expvar.NewInt("amqp.unmatched_requests")
	unmatchedResponses = expvar.NewInt("amqp.unmatched_responses")
)

func init() {
	protos.Register("amqp", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &Amqp{}
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

func (amqp *Amqp) init(results publish.Transactions, config *amqpConfig) error {
	amqp.initMethodMap()
	amqp.setFromConfig(config)

	if amqp.HideConnectionInformation == false {
		amqp.addConnectionMethods()
	}
	amqp.transactions = common.NewCache(
		amqp.transactionTimeout,
		protos.DefaultTransactionHashSize)
	amqp.transactions.StartJanitor(amqp.transactionTimeout)
	amqp.results = results
	return nil
}

func (amqp *Amqp) initMethodMap() {
	amqp.MethodMap = map[codeClass]map[codeMethod]AmqpMethod{
		connectionCode: map[codeMethod]AmqpMethod{
			connectionClose:   connectionCloseMethod,
			connectionCloseOk: okMethod,
		},
		channelCode: map[codeMethod]AmqpMethod{
			channelClose:   channelCloseMethod,
			channelCloseOk: okMethod,
		},
		exchangeCode: map[codeMethod]AmqpMethod{
			exchangeDeclare:   exchangeDeclareMethod,
			exchangeDeclareOk: okMethod,
			exchangeDelete:    exchangeDeleteMethod,
			exchangeDeleteOk:  okMethod,
			exchangeBind:      exchangeBindMethod,
			exchangeBindOk:    okMethod,
			exchangeUnbind:    exchangeUnbindMethod,
			exchangeUnbindOk:  okMethod,
		},
		queueCode: map[codeMethod]AmqpMethod{
			queueDeclare:   queueDeclareMethod,
			queueDeclareOk: queueDeclareOkMethod,
			queueBind:      queueBindMethod,
			queueBindOk:    okMethod,
			queueUnbind:    queueUnbindMethod,
			queueUnbindOk:  okMethod,
			queuePurge:     queuePurgeMethod,
			queuePurgeOk:   queuePurgeOkMethod,
			queueDelete:    queueDeleteMethod,
			queueDeleteOk:  queueDeleteOkMethod,
		},
		basicCode: map[codeMethod]AmqpMethod{
			basicConsume:   basicConsumeMethod,
			basicConsumeOk: basicConsumeOkMethod,
			basicCancel:    basicCancelMethod,
			basicCancelOk:  basicCancelOkMethod,
			basicPublish:   basicPublishMethod,
			basicReturn:    basicReturnMethod,
			basicDeliver:   basicDeliverMethod,
			basicGet:       basicGetMethod,
			basicGetOk:     basicGetOkMethod,
			basicGetEmpty:  basicGetEmptyMethod,
			basicAck:       basicAckMethod,
			basicReject:    basicRejectMethod,
			basicRecover:   basicRecoverMethod,
			basicRecoverOk: okMethod,
			basicNack:      basicNackMethod,
		},
		txCode: map[codeMethod]AmqpMethod{
			txSelect:     txSelectMethod,
			txSelectOk:   okMethod,
			txCommit:     txCommitMethod,
			txCommitOk:   okMethod,
			txRollback:   txRollbackMethod,
			txRollbackOk: okMethod,
		},
	}
}

func (amqp *Amqp) GetPorts() []int {
	return amqp.Ports
}

func (amqp *Amqp) setFromConfig(config *amqpConfig) {
	amqp.Ports = config.Ports
	amqp.SendRequest = config.SendRequest
	amqp.SendResponse = config.SendResponse
	amqp.MaxBodyLength = config.MaxBodyLength
	amqp.ParseHeaders = config.ParseHeaders
	amqp.ParseArguments = config.ParseArguments
	amqp.HideConnectionInformation = config.HideConnectionInformation
	amqp.transactionTimeout = config.TransactionTimeout
}

func (amqp *Amqp) addConnectionMethods() {
	amqp.MethodMap[connectionCode][connectionStart] = connectionStartMethod
	amqp.MethodMap[connectionCode][connectionStartOk] = connectionStartOkMethod
	amqp.MethodMap[connectionCode][connectionTune] = connectionTuneMethod
	amqp.MethodMap[connectionCode][connectionTuneOk] = connectionTuneOkMethod
	amqp.MethodMap[connectionCode][connectionOpen] = connectionOpenMethod
	amqp.MethodMap[connectionCode][connectionOpenOk] = okMethod
	amqp.MethodMap[channelCode][channelOpen] = channelOpenMethod
	amqp.MethodMap[channelCode][channelOpenOk] = okMethod
	amqp.MethodMap[channelCode][channelFlow] = channelFlowMethod
	amqp.MethodMap[channelCode][channelFlowOk] = channelFlowOkMethod
	amqp.MethodMap[basicCode][basicQos] = basicQosMethod
	amqp.MethodMap[basicCode][basicQosOk] = okMethod
}

func (amqp *Amqp) ConnectionTimeout() time.Duration {
	return amqp.transactionTimeout
}

func (amqp *Amqp) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple,
	dir uint8, private protos.ProtocolData) protos.ProtocolData {

	defer logp.Recover("ParseAmqp exception")
	detailedf("Parse method triggered")

	priv := amqpPrivateData{}
	if private != nil {
		var ok bool
		priv, ok = private.(amqpPrivateData)
		if !ok {
			priv = amqpPrivateData{}
		}
	}

	if priv.Data[dir] == nil {
		priv.Data[dir] = &AmqpStream{
			tcptuple: tcptuple,
			data:     pkt.Payload,
			message:  &AmqpMessage{Ts: pkt.Ts},
		}
	} else {
		// concatenate databytes
		priv.Data[dir].data = append(priv.Data[dir].data, pkt.Payload...)
		if len(priv.Data[dir].data) > tcp.TCP_MAX_DATA_IN_STREAM {
			debugf("Stream data too large, dropping TCP stream")
			priv.Data[dir] = nil
			return priv
		}
	}

	stream := priv.Data[dir]

	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &AmqpMessage{Ts: pkt.Ts}
		}

		ok, complete := amqp.amqpMessageParser(stream)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.Data[dir] = nil
			return priv
		}
		if !complete {
			break
		}
		amqp.handleAmqp(stream.message, tcptuple, dir)
	}
	return priv
}

func (amqp *Amqp) GapInStream(tcptuple *common.TcpTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	detailedf("GapInStream called")
	return private, true
}

func (amqp *Amqp) ReceivedFin(tcptuple *common.TcpTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

func (amqp *Amqp) handleAmqpRequest(msg *AmqpMessage) {
	// Add it to the HT
	tuple := msg.TcpTuple

	trans := amqp.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.Amqp != nil {
			debugf("Two requests without a Response. Dropping old request: %s", trans.Amqp)
			unmatchedRequests.Add(1)
		}
	} else {
		trans = &AmqpTransaction{Type: "amqp", tuple: tuple}
		amqp.transactions.Put(tuple.Hashable(), trans)
	}

	trans.ts = msg.Ts
	trans.Ts = int64(trans.ts.UnixNano() / 1000)
	trans.JsTs = msg.Ts
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

	trans.Method = msg.Method
	// get the righ request
	if len(msg.Request) > 0 {
		trans.Request = strings.Join([]string{msg.Method, msg.Request}, " ")
	} else {
		trans.Request = msg.Method
	}
	//length = message + 4 bytes header + frame end octet
	trans.BytesIn = msg.Body_size + 12
	if msg.Fields != nil {
		trans.Amqp = msg.Fields
	} else {
		trans.Amqp = common.MapStr{}
	}

	//if error or exception, publish it now. sometimes client or server never send
	//an ack message and the error is lost. Also, if nowait flag set, don't expect
	//any response and publish
	if isAsynchronous(trans) {
		amqp.publishTransaction(trans)
		debugf("Amqp transaction completed")
		amqp.transactions.Delete(trans.tuple.Hashable())
		return
	}

	if trans.timer != nil {
		trans.timer.Stop()
	}
	trans.timer = time.AfterFunc(TransactionTimeout, func() { amqp.expireTransaction(trans) })
}

func (amqp *Amqp) handleAmqpResponse(msg *AmqpMessage) {
	tuple := msg.TcpTuple
	trans := amqp.getTransaction(tuple.Hashable())
	if trans == nil || trans.Amqp == nil {
		debugf("Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}

	//length = message + 4 bytes class/method + frame end octet + header
	trans.BytesOut = msg.Body_size + 12
	//merge the both fields from request and response
	trans.Amqp.Update(msg.Fields)
	trans.Response = common.OK_STATUS

	if msg.Method == "basic.get-empty" {
		trans.Method = "basic.get-empty"
	}

	trans.ResponseTime = int32(msg.Ts.Sub(trans.ts).Nanoseconds() / 1e6)
	trans.Notes = msg.Notes

	amqp.publishTransaction(trans)

	debugf("Amqp transaction completed")

	// remove from map
	amqp.transactions.Delete(trans.tuple.Hashable())
	if trans.timer != nil {
		trans.timer.Stop()
	}
}

func (amqp *Amqp) expireTransaction(trans *AmqpTransaction) {
	debugf("Transaction expired")

	//possibility of a connection.close or channel.close method that didn't get an
	//ok answer. Let's publish it.
	if isCloseError(trans) {
		trans.Notes = append(trans.Notes, "Close-ok method not received by sender")
		amqp.publishTransaction(trans)
	}
	// remove from map
	amqp.transactions.Delete(trans.tuple.Hashable())
}

//This method handles published messages from clients. Being an async
//process, the method, header and body frames are regrouped in one transaction
func (amqp *Amqp) handlePublishing(client *AmqpMessage) {

	tuple := client.TcpTuple
	trans := amqp.getTransaction(tuple.Hashable())

	if trans == nil {
		trans = &AmqpTransaction{Type: "amqp", tuple: tuple}
		amqp.transactions.Put(client.TcpTuple.Hashable(), trans)
	}

	trans.ts = client.Ts
	trans.Ts = int64(client.Ts.UnixNano() / 1000)
	trans.JsTs = client.Ts
	trans.Src = common.Endpoint{
		Ip:   client.TcpTuple.Src_ip.String(),
		Port: client.TcpTuple.Src_port,
		Proc: string(client.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   client.TcpTuple.Dst_ip.String(),
		Port: client.TcpTuple.Dst_port,
		Proc: string(client.CmdlineTuple.Dst),
	}

	trans.Method = client.Method
	//for publishing and delivering, bytes in and out represent the length of the
	//message itself
	trans.BytesIn = client.Body_size

	if client.Body_size > uint64(amqp.MaxBodyLength) {
		trans.Body = client.Body[:amqp.MaxBodyLength]
	} else {
		trans.Body = client.Body
	}

	trans.ToString = isStringable(client)

	trans.Amqp = client.Fields
	amqp.publishTransaction(trans)
	debugf("Amqp transaction completed")
	//delete trans from map
	amqp.transactions.Delete(trans.tuple.Hashable())
}

//This method handles delivered messages via basic.deliver and basic.get-ok AND
//returned messages to clients. Being an async process, the method, header and
//body frames are regrouped in one transaction
func (amqp *Amqp) handleDelivering(server *AmqpMessage) {

	tuple := server.TcpTuple
	trans := amqp.getTransaction(tuple.Hashable())

	if trans == nil {
		trans = &AmqpTransaction{Type: "amqp", tuple: tuple}
		amqp.transactions.Put(server.TcpTuple.Hashable(), trans)
	}

	trans.ts = server.Ts
	trans.Ts = int64(server.Ts.UnixNano() / 1000)
	trans.JsTs = server.Ts
	trans.Src = common.Endpoint{
		Ip:   server.TcpTuple.Src_ip.String(),
		Port: server.TcpTuple.Src_port,
		Proc: string(server.CmdlineTuple.Src),
	}
	trans.Dst = common.Endpoint{
		Ip:   server.TcpTuple.Dst_ip.String(),
		Port: server.TcpTuple.Dst_port,
		Proc: string(server.CmdlineTuple.Dst),
	}

	//for publishing and delivering, bytes in and out represent the length of the
	//message itself
	trans.BytesOut = server.Body_size

	if server.Body_size > uint64(amqp.MaxBodyLength) {
		trans.Body = server.Body[:amqp.MaxBodyLength]
	} else {
		trans.Body = server.Body
	}
	trans.ToString = isStringable(server)
	if server.Method == "basic.get-ok" {
		trans.Method = "basic.get"
	} else {
		trans.Method = server.Method
	}
	trans.Amqp = server.Fields

	amqp.publishTransaction(trans)
	debugf("Amqp transaction completed")
	//delete trans from map
	amqp.transactions.Delete(trans.tuple.Hashable())
}

func (amqp *Amqp) publishTransaction(t *AmqpTransaction) {

	if amqp.results == nil {
		return
	}

	event := common.MapStr{}
	event["type"] = "amqp"

	event["method"] = t.Method
	if isError(t) {
		event["status"] = common.ERROR_STATUS
	} else {
		event["status"] = common.OK_STATUS
	}
	event["responsetime"] = t.ResponseTime
	event["amqp"] = t.Amqp
	event["bytes_out"] = t.BytesOut
	event["bytes_in"] = t.BytesIn
	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.Src
	event["dst"] = &t.Dst

	//let's try to convert request/response to a readable format
	if amqp.SendRequest {
		if t.Method == "basic.publish" {
			if t.ToString {
				if uint64(len(t.Body)) < t.BytesIn {
					event["request"] = string(t.Body) + " [...]"
				} else {
					event["request"] = string(t.Body)
				}
			} else {
				if uint64(len(t.Body)) < t.BytesIn {
					event["request"] = bodyToString(t.Body) + " [...]"
				} else {
					event["request"] = bodyToString(t.Body)
				}
			}
		} else {
			event["request"] = t.Request
		}
	}
	if amqp.SendResponse {
		if t.Method == "basic.deliver" || t.Method == "basic.return" ||
			t.Method == "basic.get" {
			if t.ToString {
				if uint64(len(t.Body)) < t.BytesOut {
					event["response"] = string(t.Body) + " [...]"
				} else {
					event["response"] = string(t.Body)
				}
			} else {
				if uint64(len(t.Body)) < t.BytesOut {
					event["response"] = bodyToString(t.Body) + " [...]"
				} else {
					event["response"] = bodyToString(t.Body)
				}
			}
		} else {
			event["response"] = t.Response
		}
	}
	if len(t.Notes) > 0 {
		event["notes"] = t.Notes
	}

	amqp.results.PublishTransaction(event)
}

//function to check if method is async or not
func isAsynchronous(trans *AmqpTransaction) bool {
	if val, ok := trans.Amqp["no-wait"]; ok && val == true {
		return true
	} else {
		return trans.Method == "basic.reject" ||
			trans.Method == "basic.ack" ||
			trans.Method == "basic.nack"
	}
}

//function to convert a body slice into a readable format
func bodyToString(data []byte) string {
	var ret []string = make([]string, len(data))

	for i, c := range data {
		ret[i] = strconv.Itoa(int(c))
	}
	return strings.Join(ret, " ")
}

//function used to check if a body message can be converted to readable string
func isStringable(m *AmqpMessage) bool {
	stringable := false

	if contentEncoding, ok := m.Fields["content-encoding"].(string); ok &&
		contentEncoding != "" {
		return false
	}
	if contentType, ok := m.Fields["content-type"].(string); ok {
		stringable = strings.Contains(contentType, "text") ||
			strings.Contains(contentType, "json")
	}
	return stringable
}

func (amqp *Amqp) getTransaction(k common.HashableTcpTuple) *AmqpTransaction {
	v := amqp.transactions.Get(k)
	if v != nil {
		return v.(*AmqpTransaction)
	}
	return nil
}

func isError(t *AmqpTransaction) bool {
	return t.Method == "basic.return" || t.Method == "basic.reject" ||
		isCloseError(t)
}

func isCloseError(t *AmqpTransaction) bool {
	return (t.Method == "connection.close" || t.Method == "channel.close") &&
		getReplyCode(t.Amqp) >= 300
}

func getReplyCode(m common.MapStr) uint16 {
	code, ok := m["reply-code"].(uint16)
	if !ok {
		return 0
	} else {
		return code
	}
}
