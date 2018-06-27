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

package cassandra

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/packetbeat/protos"
)

// Transaction Publisher.
type transPub struct {
	sendRequest        bool
	sendResponse       bool
	sendRequestHeader  bool
	sendResponseHeader bool
	ignoredOps         string

	results protos.Reporter
}

func (pub *transPub) onTransaction(requ, resp *message) error {
	if pub.results == nil {
		return nil
	}

	event := pub.createEvent(requ, resp)
	if event.Fields != nil {
		pub.results(event)
	}
	return nil
}

func (pub *transPub) createEvent(requ, resp *message) beat.Event {
	status := common.OK_STATUS

	if resp.failed {
		status = common.ERROR_STATUS
	}

	//ignore
	if (resp != nil && resp.ignored) || (requ != nil && requ.ignored) {
		return beat.Event{}
	}

	var timestamp time.Time
	fields := common.MapStr{
		"type":      "cassandra",
		"status":    status,
		"cassandra": common.MapStr{},
	}

	//requ can be null, if the message is a PUSHed message
	if requ != nil {
		// resp_time in milliseconds
		responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

		source, dest := common.MakeEndpointPair(requ.Tuple.BaseTuple, requ.CmdlineTuple)
		src, dst := &source, &dest

		timestamp = requ.Ts
		fields["responsetime"] = responseTime
		fields["bytes_in"] = requ.Size
		fields["src"] = src

		// add processing notes/errors to fields
		if len(requ.Notes)+len(resp.Notes) > 0 {
			fields["notes"] = append(requ.Notes, resp.Notes...)
		}

		if pub.sendRequest {
			if pub.sendRequestHeader {
				if requ.data == nil {
					requ.data = map[string]interface{}{}
				}
				requ.data["headers"] = requ.header
			}

			if len(requ.data) > 0 {
				fields["cassandra"].(common.MapStr)["request"] = requ.data
			}
		}

		fields["dst"] = dst

	} else {
		//dealing with PUSH message
		fields["no_request"] = true
		timestamp = resp.Ts

		dst := &common.Endpoint{
			IP:   resp.Tuple.DstIP.String(),
			Port: resp.Tuple.DstPort,
			Proc: string(resp.CmdlineTuple.Dst),
		}
		fields["dst"] = dst
	}

	fields["bytes_out"] = resp.Size

	if pub.sendResponse {

		if pub.sendResponseHeader {
			if resp.data == nil {
				resp.data = map[string]interface{}{}
			}

			resp.data["headers"] = resp.header
		}

		if len(resp.data) > 0 {
			fields["cassandra"].(common.MapStr)["response"] = resp.data
		}

	}

	return beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	}
}
