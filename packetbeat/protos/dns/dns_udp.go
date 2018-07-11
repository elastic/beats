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

package dns

import (
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos"
)

// Only EDNS packets should have their size beyond this value
const maxDNSPacketSize = (1 << 9) // 512 (bytes)

func (dns *dnsPlugin) ParseUDP(pkt *protos.Packet) {
	defer logp.Recover("Dns ParseUdp")
	packetSize := len(pkt.Payload)

	debugf("Parsing packet addressed with %s of length %d.",
		pkt.Tuple.String(), packetSize)

	dnsPkt, err := decodeDNSData(transportUDP, pkt.Payload)
	if err != nil {
		// This means that malformed requests or responses are being sent or
		// that someone is attempting to the DNS port for non-DNS traffic. Both
		// are issues that a monitoring system should report.
		debugf("%s", err.Error())
		return
	}

	dnsTuple := dnsTupleFromIPPort(&pkt.Tuple, transportUDP, dnsPkt.Id)
	dnsMsg := &dnsMessage{
		ts:           pkt.Ts,
		tuple:        pkt.Tuple,
		cmdlineTuple: procs.ProcWatcher.FindProcessesTupleUDP(&pkt.Tuple),
		data:         dnsPkt,
		length:       packetSize,
	}

	if dnsMsg.data.Response {
		dns.receivedDNSResponse(&dnsTuple, dnsMsg)
	} else /* Query */ {
		dns.receivedDNSRequest(&dnsTuple, dnsMsg)
	}
}
