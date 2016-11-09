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

type amqpPlugin struct {
	ports                     []int
	sendRequest               bool
	sendResponse              bool
	maxBodyLength             int
	parseHeaders              bool
	parseArguments            bool
	hideConnectionInformation bool
	transactions              *common.Cache
	transactionTimeout        time.Duration
	results                   publish.Transactions

	//map containing functions associated with different method numbers
	methodMap map[codeClass]map[codeMethod]amqpMethod
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
	p := &amqpPlugin{}
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

func (amqp *amqpPlugin) init(results publish.Transactions, config *amqpConfig) error {
	amqp.initMethodMap()
	amqp.setFromConfig(config)

	if amqp.hideConnectionInformation == false {
		amqp.addConnectionMethods()
	}
	amqp.transactions = common.NewCache(
		amqp.transactionTimeout,
		protos.DefaultTransactionHashSize)
	amqp.transactions.StartJanitor(amqp.transactionTimeout)
	amqp.results = results
	return nil
}

func (amqp *amqpPlugin) initMethodMap() {
	amqp.methodMap = map[codeClass]map[codeMethod]amqpMethod{
		connectionCode: {
			connectionClose:   connectionCloseMethod,
			connectionCloseOk: okMethod,
		},
		channelCode: {
			channelClose:   channelCloseMethod,
			channelCloseOk: okMethod,
		},
		exchangeCode: {
			exchangeDeclare:   exchangeDeclareMethod,
			exchangeDeclareOk: okMethod,
			exchangeDelete:    exchangeDeleteMethod,
			exchangeDeleteOk:  okMethod,
			exchangeBind:      exchangeBindMethod,
			exchangeBindOk:    okMethod,
			exchangeUnbind:    exchangeUnbindMethod,
			exchangeUnbindOk:  okMethod,
		},
		queueCode: {
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
		basicCode: {
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
		txCode: {
			txSelect:     txSelectMethod,
			txSelectOk:   okMethod,
			txCommit:     txCommitMethod,
			txCommitOk:   okMethod,
			txRollback:   txRollbackMethod,
			txRollbackOk: okMethod,
		},
	}
}

func (amqp *amqpPlugin) GetPorts() []int {
	return amqp.ports
}

func (amqp *amqpPlugin) setFromConfig(config *amqpConfig) {
	amqp.ports = config.Ports
	amqp.sendRequest = config.SendRequest
	amqp.sendResponse = config.SendResponse
	amqp.maxBodyLength = config.MaxBodyLength
	amqp.parseHeaders = config.ParseHeaders
	amqp.parseArguments = config.ParseArguments
	amqp.hideConnectionInformation = config.HideConnectionInformation
	amqp.transactionTimeout = config.TransactionTimeout
}

func (amqp *amqpPlugin) addConnectionMethods() {
	amqp.methodMap[connectionCode][connectionStart] = connectionStartMethod
	amqp.methodMap[connectionCode][connectionStartOk] = connectionStartOkMethod
	amqp.methodMap[connectionCode][connectionTune] = connectionTuneMethod
	amqp.methodMap[connectionCode][connectionTuneOk] = connectionTuneOkMethod
	amqp.methodMap[connectionCode][connectionOpen] = connectionOpenMethod
	amqp.methodMap[connectionCode][connectionOpenOk] = okMethod
	amqp.methodMap[channelCode][channelOpen] = channelOpenMethod
	amqp.methodMap[channelCode][channelOpenOk] = okMethod
	amqp.methodMap[channelCode][channelFlow] = channelFlowMethod
	amqp.methodMap[channelCode][channelFlowOk] = channelFlowOkMethod
	amqp.methodMap[basicCode][basicQos] = basicQosMethod
	amqp.methodMap[basicCode][basicQosOk] = okMethod
}

func (amqp *amqpPlugin) ConnectionTimeout() time.Duration {
	return amqp.transactionTimeout
}

func (amqp *amqpPlugin) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple,
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

	if priv.data[dir] == nil {
		priv.data[dir] = &amqpStream{
			data:    pkt.Payload,
			message: &amqpMessage{ts: pkt.Ts},
		}
	} else {
		// concatenate databytes
		priv.data[dir].data = append(priv.data[dir].data, pkt.Payload...)
		if len(priv.data[dir].data) > tcp.TCPMaxDataInStream {
			debugf("Stream data too large, dropping TCP stream")
			priv.data[dir] = nil
			return priv
		}
	}

	stream := priv.data[dir]

	for len(stream.data) > 0 {
		if stream.message == nil {
			stream.message = &amqpMessage{ts: pkt.Ts}
		}

		ok, complete := amqp.amqpMessageParser(stream)
		if !ok {
			// drop this tcp stream. Will retry parsing with the next
			// segment in it
			priv.data[dir] = nil
			return priv
		}
		if !complete {
			break
		}
		amqp.handleAmqp(stream.message, tcptuple, dir)
	}
	return priv
}

func (amqp *amqpPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
	nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
	detailedf("GapInStream called")
	return private, true
}

func (amqp *amqpPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
	private protos.ProtocolData) protos.ProtocolData {
	return private
}

func (amqp *amqpPlugin) handleAmqpRequest(msg *amqpMessage) {
	// Add it to the HT
	tuple := msg.tcpTuple

	trans := amqp.getTransaction(tuple.Hashable())
	if trans != nil {
		if trans.amqp != nil {
			debugf("Two requests without a Response. Dropping old request: %s", trans.amqp)
			unmatchedRequests.Add(1)
		}
	} else {
		trans = &amqpTransaction{tuple: tuple}
		amqp.transactions.Put(tuple.Hashable(), trans)
	}

	trans.ts = msg.ts
	trans.src = common.Endpoint{
		IP:   msg.tcpTuple.SrcIP.String(),
		Port: msg.tcpTuple.SrcPort,
		Proc: string(msg.cmdlineTuple.Src),
	}
	trans.dst = common.Endpoint{
		IP:   msg.tcpTuple.DstIP.String(),
		Port: msg.tcpTuple.DstPort,
		Proc: string(msg.cmdlineTuple.Dst),
	}
	if msg.direction == tcp.TCPDirectionReverse {
		trans.src, trans.dst = trans.dst, trans.src
	}

	trans.method = msg.method
	// get the righ request
	if len(msg.request) > 0 {
		trans.request = strings.Join([]string{msg.method, msg.request}, " ")
	} else {
		trans.request = msg.method
	}
	//length = message + 4 bytes header + frame end octet
	trans.bytesIn = msg.bodySize + 12
	if msg.fields != nil {
		trans.amqp = msg.fields
	} else {
		trans.amqp = common.MapStr{}
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
	trans.timer = time.AfterFunc(transactionTimeout, func() { amqp.expireTransaction(trans) })
}

func (amqp *amqpPlugin) handleAmqpResponse(msg *amqpMessage) {
	tuple := msg.tcpTuple
	trans := amqp.getTransaction(tuple.Hashable())
	if trans == nil || trans.amqp == nil {
		debugf("Response from unknown transaction. Ignoring.")
		unmatchedResponses.Add(1)
		return
	}

	//length = message + 4 bytes class/method + frame end octet + header
	trans.bytesOut = msg.bodySize + 12
	//merge the both fields from request and response
	trans.amqp.Update(msg.fields)
	trans.response = common.OK_STATUS

	if msg.method == "basic.get-empty" {
		trans.method = "basic.get-empty"
	}

	trans.responseTime = int32(msg.ts.Sub(trans.ts).Nanoseconds() / 1e6)
	trans.notes = msg.notes

	amqp.publishTransaction(trans)

	debugf("Amqp transaction completed")

	// remove from map
	amqp.transactions.Delete(trans.tuple.Hashable())
	if trans.timer != nil {
		trans.timer.Stop()
	}
}

func (amqp *amqpPlugin) expireTransaction(trans *amqpTransaction) {
	debugf("Transaction expired")

	//possibility of a connection.close or channel.close method that didn't get an
	//ok answer. Let's publish it.
	if isCloseError(trans) {
		trans.notes = append(trans.notes, "Close-ok method not received by sender")
		amqp.publishTransaction(trans)
	}
	// remove from map
	amqp.transactions.Delete(trans.tuple.Hashable())
}

//This method handles published messages from clients. Being an async
//process, the method, header and body frames are regrouped in one transaction
func (amqp *amqpPlugin) handlePublishing(client *amqpMessage) {

	tuple := client.tcpTuple
	trans := amqp.getTransaction(tuple.Hashable())

	if trans == nil {
		trans = &amqpTransaction{tuple: tuple}
		amqp.transactions.Put(client.tcpTuple.Hashable(), trans)
	}

	trans.ts = client.ts
	trans.src = common.Endpoint{
		IP:   client.tcpTuple.SrcIP.String(),
		Port: client.tcpTuple.SrcPort,
		Proc: string(client.cmdlineTuple.Src),
	}
	trans.dst = common.Endpoint{
		IP:   client.tcpTuple.DstIP.String(),
		Port: client.tcpTuple.DstPort,
		Proc: string(client.cmdlineTuple.Dst),
	}

	trans.method = client.method
	//for publishing and delivering, bytes in and out represent the length of the
	//message itself
	trans.bytesIn = client.bodySize

	if client.bodySize > uint64(amqp.maxBodyLength) {
		trans.body = client.body[:amqp.maxBodyLength]
	} else {
		trans.body = client.body
	}

	trans.toString = isStringable(client)

	trans.amqp = client.fields
	amqp.publishTransaction(trans)
	debugf("Amqp transaction completed")
	//delete trans from map
	amqp.transactions.Delete(trans.tuple.Hashable())
}

//This method handles delivered messages via basic.deliver and basic.get-ok AND
//returned messages to clients. Being an async process, the method, header and
//body frames are regrouped in one transaction
func (amqp *amqpPlugin) handleDelivering(server *amqpMessage) {

	tuple := server.tcpTuple
	trans := amqp.getTransaction(tuple.Hashable())

	if trans == nil {
		trans = &amqpTransaction{tuple: tuple}
		amqp.transactions.Put(server.tcpTuple.Hashable(), trans)
	}

	trans.ts = server.ts
	trans.src = common.Endpoint{
		IP:   server.tcpTuple.SrcIP.String(),
		Port: server.tcpTuple.SrcPort,
		Proc: string(server.cmdlineTuple.Src),
	}
	trans.dst = common.Endpoint{
		IP:   server.tcpTuple.DstIP.String(),
		Port: server.tcpTuple.DstPort,
		Proc: string(server.cmdlineTuple.Dst),
	}

	//for publishing and delivering, bytes in and out represent the length of the
	//message itself
	trans.bytesOut = server.bodySize

	if server.bodySize > uint64(amqp.maxBodyLength) {
		trans.body = server.body[:amqp.maxBodyLength]
	} else {
		trans.body = server.body
	}
	trans.toString = isStringable(server)
	if server.method == "basic.get-ok" {
		trans.method = "basic.get"
	} else {
		trans.method = server.method
	}
	trans.amqp = server.fields

	amqp.publishTransaction(trans)
	debugf("Amqp transaction completed")
	//delete trans from map
	amqp.transactions.Delete(trans.tuple.Hashable())
}

func (amqp *amqpPlugin) publishTransaction(t *amqpTransaction) {

	if amqp.results == nil {
		return
	}

	event := common.MapStr{}
	event["type"] = "amqp"

	event["method"] = t.method
	if isError(t) {
		event["status"] = common.ERROR_STATUS
	} else {
		event["status"] = common.OK_STATUS
	}
	event["responsetime"] = t.responseTime
	event["amqp"] = t.amqp
	event["bytes_out"] = t.bytesOut
	event["bytes_in"] = t.bytesIn
	event["@timestamp"] = common.Time(t.ts)
	event["src"] = &t.src
	event["dst"] = &t.dst

	//let's try to convert request/response to a readable format
	if amqp.sendRequest {
		if t.method == "basic.publish" {
			if t.toString {
				if uint64(len(t.body)) < t.bytesIn {
					event["request"] = string(t.body) + " [...]"
				} else {
					event["request"] = string(t.body)
				}
			} else {
				if uint64(len(t.body)) < t.bytesIn {
					event["request"] = bodyToString(t.body) + " [...]"
				} else {
					event["request"] = bodyToString(t.body)
				}
			}
		} else {
			event["request"] = t.request
		}
	}
	if amqp.sendResponse {
		if t.method == "basic.deliver" || t.method == "basic.return" ||
			t.method == "basic.get" {
			if t.toString {
				if uint64(len(t.body)) < t.bytesOut {
					event["response"] = string(t.body) + " [...]"
				} else {
					event["response"] = string(t.body)
				}
			} else {
				if uint64(len(t.body)) < t.bytesOut {
					event["response"] = bodyToString(t.body) + " [...]"
				} else {
					event["response"] = bodyToString(t.body)
				}
			}
		} else {
			event["response"] = t.response
		}
	}
	if len(t.notes) > 0 {
		event["notes"] = t.notes
	}

	amqp.results.PublishTransaction(event)
}

//function to check if method is async or not
func isAsynchronous(trans *amqpTransaction) bool {
	if val, ok := trans.amqp["no-wait"]; ok && val == true {
		return true
	}

	return trans.method == "basic.reject" ||
		trans.method == "basic.ack" ||
		trans.method == "basic.nack"
}

//function to convert a body slice into a readable format
func bodyToString(data []byte) string {
	ret := make([]string, len(data))
	for i, c := range data {
		ret[i] = strconv.Itoa(int(c))
	}
	return strings.Join(ret, " ")
}

//function used to check if a body message can be converted to readable string
func isStringable(m *amqpMessage) bool {
	stringable := false

	if contentEncoding, ok := m.fields["content-encoding"].(string); ok &&
		contentEncoding != "" {
		return false
	}
	if contentType, ok := m.fields["content-type"].(string); ok {
		stringable = strings.Contains(contentType, "text") ||
			strings.Contains(contentType, "json")
	}
	return stringable
}

func (amqp *amqpPlugin) getTransaction(k common.HashableTCPTuple) *amqpTransaction {
	v := amqp.transactions.Get(k)
	if v != nil {
		return v.(*amqpTransaction)
	}
	return nil
}

func isError(t *amqpTransaction) bool {
	return t.method == "basic.return" || t.method == "basic.reject" ||
		isCloseError(t)
}

func isCloseError(t *amqpTransaction) bool {
	return (t.method == "connection.close" || t.method == "channel.close") &&
		getReplyCode(t.amqp) >= 300
}

func getReplyCode(m common.MapStr) uint16 {
	code, _ := m["reply-code"].(uint16)
	return code
}
