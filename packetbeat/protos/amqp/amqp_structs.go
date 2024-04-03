// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package amqp

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type amqpMethod func(*amqpMessage, []byte) (bool, bool)

const (
	transactionsHashSize = 1 << 16
	transactionTimeout   = 10 * 1e9
)

// layout used when a timestamp must be parsed
const (
	amqpTimeLayout = "January _2 15:04:05 2006"
)

// Frame types and codes

type frameType byte

const (
	methodType    frameType = 1
	headerType    frameType = 2
	bodyType      frameType = 3
	heartbeatType frameType = 8
)

const (
	frameEndOctet byte = 206
)

// Codes for MethodMap
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

// Message properties codes for byte prop1 in getMessageProperties
const (
	expirationProp      byte = 1
	replyToProp         byte = 2
	correlationIDProp   byte = 4
	priorityProp        byte = 8
	deliveryModeProp    byte = 16
	headersProp         byte = 32
	contentEncodingProp byte = 64
	contentTypeProp     byte = 128
)

// Message properties codes for byte prop2 in getMessageProperties

const (
	appIDProp     byte = 8
	userIDProp    byte = 16
	typeProp      byte = 32
	timestampProp byte = 64
	messageIDProp byte = 128
)

// table types
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
	byteArray      = 'x' // rabbitMQ specific field
)

type amqpPrivateData struct {
	data [2]*amqpStream
}

type amqpFrame struct {
	Type frameType
	// channel uint16  (frame channel is currently ignored)
	size    uint32
	content []byte
}

type amqpMessage struct {
	ts             time.Time
	tcpTuple       common.TCPTuple
	cmdlineTuple   *common.ProcessTuple
	method         string
	isRequest      bool
	request        string
	direction      uint8
	parseArguments bool

	// mapstr containing all the options for the methods and header fields
	fields mapstr.M

	body     []byte
	bodySize uint64

	notes []string
}

// represent a stream of data to be parsed
type amqpStream struct {
	data        []byte
	parseOffset int
	message     *amqpMessage
}

// contains the result of parsing
type amqpTransaction struct {
	tuple common.TCPTuple
	src   common.Endpoint
	dst   common.Endpoint
	ts    time.Time

	method   string
	request  string
	response string
	endTime  time.Time
	body     []byte
	bytesOut uint64
	bytesIn  uint64
	toString bool
	notes    []string

	amqp mapstr.M

	timer *time.Timer
}
