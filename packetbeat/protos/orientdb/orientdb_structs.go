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

// +build !integration

package orientdb

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type orientdbMessage struct {
	ts time.Time

	tcpTuple     common.TCPTuple
	cmdlineTuple *common.ProcessTuple
	direction    uint8

	// Standard message header fields from orientdb binary protocol
	messageLength int
	sessionID     int
	token         string
	opCode        opCode

	// deduced from content. Either an operation from the original binary protocol or the name of a command (passed through a query)
	method   string
	error    string
	resource string
	params   map[string]interface{}

	// Other fields vary very much depending on operation type
	// lets just put them in a map
	event common.MapStr
}

// Represent a stream being parsed that contains a orientdb message
type stream struct {
	tcpTuple *common.TCPTuple

	data    []byte
	message *orientdbMessage
}

// Parser moves to next message in stream
func (st *stream) PrepareForNewMessage() {
	st.data = st.data[st.message.messageLength:]
	st.message = nil
}

// The private data of a parser instance
// is composed of 2 potentially active streams: incoming, outgoing
type orientdbConnectionData struct {
	streams [2]*stream
}

// Represent a full orientdb transaction (request/reply)
// These transactions are the end product of this parser
type transaction struct {
	cmdline      *common.ProcessTuple
	src          common.Endpoint
	dst          common.Endpoint
	responseTime int32
	ts           time.Time
	bytesOut     int
	bytesIn      int

	orientdb common.MapStr

	event    common.MapStr
	method   string
	error    string
	resource string
	params   map[string]interface{}
}

type opCode int32

const (
	opRequestShutdown         opCode = 1
	opRequestConnect          opCode = 2
	opRequestDbOpen           opCode = 3
	opRequestDbList           opCode = 74
	opRequestDbClose          opCode = 5
	opRequestDataClusterAdd   opCode = 10
	opRequestDataClusterCount opCode = 12
	opRequestRecordCreate     opCode = 31
	opRequestRecordRead       opCode = 30
	opRequestRecordUpdate     opCode = 32
	opRequestRecordDelete     opCode = 33
	opRequestCommand          opCode = 41
)

var opCodeNames = map[opCode]string{
	1:  "REQUEST_SHUTDOWN",
	2:  "REQUEST_CONNECT",
	3:  "REQUEST_DB_OPEN",
	74: "REQUEST_DB_LIST",
	5:  "REQUEST_DB_CLOSE",
	10: "REQUEST_DATA_CLUSTER_ADD",
	12: "REQUEST_DATA_CLUSTER_COUNT",
	31: "REQUEST_RECORD_CREATE",
	30: "REQUEST_RECORD_READ",
	32: "REQUEST_RECORD_UPDATE",
	33: "REQUEST_RECORD_DELETE",
	41: "REQUEST_COMMAND",
}

func validOpcode(o opCode) bool {
	_, found := opCodeNames[o]
	return found
}

func (o opCode) String() string {
	if name, found := opCodeNames[o]; found {
		return name
	}
	return fmt.Sprintf("(value=%d)", int32(o))
}

func awaitsReply(c opCode) bool {
	return false
}
