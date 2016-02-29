package amqp

import (
	"github.com/elastic/beats/libbeat/common"
	"time"
)

type AmqpMethod func(*AmqpMessage, []byte) (bool, bool)

const (
	TransactionsHashSize = 2 ^ 16
	TransactionTimeout   = 10 * 1e9
)

//layout used when a timestamp must be parsed
const (
	amqpTimeLayout = "January _2 15:04:05 2006"
)

//Frame types and codes
const (
	methodType    = 1
	headerType    = 2
	bodyType      = 3
	heartbeatType = 8
)

const (
	frameEndOctet byte = 206
)

//Codes for MethodMap
type codeClass uint16

const (
	connectionCode codeClass = 10
	channelCode    codeClass = 20
	exchangeCode   codeClass = 40
	queueCode      codeClass = 50
	basicCode      codeClass = 60
	txCode         codeClass = 90
)

type codeMethod uint16

const (
	connectionStart   codeMethod = 10
	connectionStartOk codeMethod = 11
	connectionTune    codeMethod = 30
	connectionTuneOk  codeMethod = 31
	connectionOpen    codeMethod = 40
	connectionOpenOk  codeMethod = 41
	connectionClose   codeMethod = 50
	connectionCloseOk codeMethod = 51
)

const (
	channelOpen    codeMethod = 10
	channelOpenOk  codeMethod = 11
	channelFlow    codeMethod = 20
	channelFlowOk  codeMethod = 21
	channelClose   codeMethod = 40
	channelCloseOk codeMethod = 41
)

const (
	exchangeDeclare   codeMethod = 10
	exchangeDeclareOk codeMethod = 11
	exchangeDelete    codeMethod = 20
	exchangeDeleteOk  codeMethod = 21
	exchangeBind      codeMethod = 30
	exchangeBindOk    codeMethod = 31
	exchangeUnbind    codeMethod = 40
	exchangeUnbindOk  codeMethod = 51
)

const (
	queueDeclare   codeMethod = 10
	queueDeclareOk codeMethod = 11
	queueBind      codeMethod = 20
	queueBindOk    codeMethod = 21
	queuePurge     codeMethod = 30
	queuePurgeOk   codeMethod = 31
	queueDelete    codeMethod = 40
	queueDeleteOk  codeMethod = 41
	queueUnbind    codeMethod = 50
	queueUnbindOk  codeMethod = 51
)

const (
	basicQos       codeMethod = 10
	basicQosOk     codeMethod = 11
	basicConsume   codeMethod = 20
	basicConsumeOk codeMethod = 21
	basicCancel    codeMethod = 30
	basicCancelOk  codeMethod = 31
	basicPublish   codeMethod = 40
	basicReturn    codeMethod = 50
	basicDeliver   codeMethod = 60
	basicGet       codeMethod = 70
	basicGetOk     codeMethod = 71
	basicGetEmpty  codeMethod = 72
	basicAck       codeMethod = 80
	basicReject    codeMethod = 90
	basicRecover   codeMethod = 110
	basicRecoverOk codeMethod = 111
	basicNack      codeMethod = 120
)

const (
	txSelect     codeMethod = 10
	txSelectOk   codeMethod = 11
	txCommit     codeMethod = 20
	txCommitOk   codeMethod = 21
	txRollback   codeMethod = 30
	txRollbackOk codeMethod = 31
)

//Message properties codes for byte prop1 in getMessageProperties
const (
	expirationProp      byte = 1
	replyToProp         byte = 2
	correlationIdProp   byte = 4
	priorityProp        byte = 8
	deliveryModeProp    byte = 16
	headersProp         byte = 32
	contentEncodingProp byte = 64
	contentTypeProp     byte = 128
)

//Message properties codes for byte prop2 in getMessageProperties

const (
	appIdProp     byte = 8
	userIdProp    byte = 16
	typeProp      byte = 32
	timestampProp byte = 64
	messageIdProp byte = 128
)

//table types
const (
	boolean        = 't'
	shortShortInt  = 'b'
	shortShortUint = 'B'
	shortInt       = 'U'
	shortUint      = 'u'
	longInt        = 'I'
	longUint       = 'i'
	longLongInt    = 'L'
	longLongUint   = 'l'
	float          = 'f'
	double         = 'd'
	decimal        = 'D'
	shortString    = 's'
	longString     = 'S'
	fieldArray     = 'A'
	timestamp      = 'T'
	fieldTable     = 'F'
	noField        = 'V'
	byteArray      = 'x' //rabbitMQ specific field
)

type amqpPrivateData struct {
	Data [2]*AmqpStream
}

type AmqpFrame struct {
	Type    byte
	channel uint16
	size    uint32
	content []byte
}

type AmqpMessage struct {
	Ts             time.Time
	TcpTuple       common.TcpTuple
	CmdlineTuple   *common.CmdlineTuple
	Method         string
	IsRequest      bool
	Request        string
	Direction      uint8
	ParseArguments bool

	//mapstr containing all the options for the methods and header fields
	Fields common.MapStr

	Body      []byte
	Body_size uint64

	Notes []string
}

// represent a stream of data to be parsed
type AmqpStream struct {
	tcptuple *common.TcpTuple

	data        []byte
	parseOffset int

	message *AmqpMessage
}

// contains the result of parsing
type AmqpTransaction struct {
	Type  string
	tuple common.TcpTuple
	Src   common.Endpoint
	Dst   common.Endpoint
	Ts    int64
	JsTs  time.Time
	ts    time.Time

	Method       string
	Request      string
	Response     string
	ResponseTime int32
	Body         []byte
	BytesOut     uint64
	BytesIn      uint64
	ToString     bool
	Notes        []string

	Amqp common.MapStr

	timer *time.Timer
}
