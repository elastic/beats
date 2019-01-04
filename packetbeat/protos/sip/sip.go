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

package sip

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"

	"github.com/elastic/beats/packetbeat/protos"
)

var (
	debugf = logp.MakeDebug("sip")
)

// Packetbeats monitoring metrics
var (
	messageIgnored = monitoring.NewInt(nil, "sip.message_ignored")
)

const (
	transportTCP = iota
	transportUDP
)

// MessegeStatus
const (
	SipStatusReceived        = iota
	SipStatusHeaderReceiving
	SipStatusBodyReceiving
	SipStatusRejected
)

// Detail parse mode
const (
	SipDetailURI          = iota // ex. sip:bob@example.com
	SipDetailNameAddr            // ex. "Bob"<sip:bob@example.com>
	SipDetailInt                 // ex. 123
	SipDetailIntMethod           // ex. 123 INVITE
	SipDetailIntIntMethod        // ex. 123 123 INVITE
	SipDetailIntString           // ex. 123 INVITE
	SipDetailIntInt              // ex. 123 123
	SipDetailIntIntString        // ex. 123 123 INVITE
)

func init() {
	// Memo: Secound argment*New* is below New function.
	protos.Register("sip", New)
}

// New create a sip plugin
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &sipPlugin{}
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

func getLastElementStrArray(array []common.NetString) common.NetString {
	return array[len(array)-1]
}

/**
 ******************************************************************
 * transport
 *******************************************************************
 **/

// Transport protocol.
// transport=0 tcp, transport=1, udp
type transport uint8

func (t transport) String() string {

	transportNames := []string{
		"tcp",
		"udp",
	}

	if int(t) >= len(transportNames) {
		return "impossible"
	}
	return transportNames[t]
}
