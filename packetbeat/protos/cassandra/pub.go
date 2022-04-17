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

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"

	"github.com/menderesk/beats/v7/packetbeat/pb"
	"github.com/menderesk/beats/v7/packetbeat/protos"
)

// Transaction Publisher.
type transPub struct {
	sendRequest        bool
	sendResponse       bool
	sendRequestHeader  bool
	sendResponseHeader bool

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
	// Ignore.
	if (requ == nil && resp == nil) || (resp != nil && resp.ignored) || (requ != nil && requ.ignored) {
		return beat.Event{}
	}

	var ts time.Time
	var src, dst common.Endpoint
	for _, m := range []*message{requ, resp} {
		if m == nil {
			continue
		}
		ts = m.Ts
		src, dst = common.MakeEndpointPair(m.Tuple.BaseTuple, m.CmdlineTuple)
		break
	}

	evt, pbf := pb.NewBeatEvent(ts)
	pbf.SetSource(&src)
	pbf.AddIP(src.IP)
	pbf.SetDestination(&dst)
	pbf.AddIP(dst.IP)
	pbf.Event.Dataset = "cassandra"
	pbf.Network.Transport = "tcp"
	pbf.Network.Protocol = pbf.Event.Dataset

	fields := evt.Fields
	fields["type"] = pbf.Event.Dataset

	cassandra := common.MapStr{}
	status := common.OK_STATUS

	// requ can be null, if the message is a PUSHed message
	if requ != nil {
		pbf.Source.Bytes = int64(requ.Size)
		pbf.Event.Start = requ.Ts
		pbf.Error.Message = requ.Notes

		if pub.sendRequest {
			if pub.sendRequestHeader {
				if requ.data == nil {
					requ.data = common.MapStr{}
				}
				requ.data["headers"] = requ.header
			}

			if len(requ.data) > 0 {
				cassandra["request"] = requ.data
			}
		}
	} else {
		// dealing with PUSH message
		cassandra["no_request"] = true
	}

	if resp != nil {
		pbf.Destination.Bytes = int64(resp.Size)
		pbf.Event.End = resp.Ts
		pbf.Error.Message = append(pbf.Error.Message, resp.Notes...)

		if resp.failed {
			status = common.ERROR_STATUS
		}

		if pub.sendResponse {
			if pub.sendResponseHeader {
				if resp.data == nil {
					resp.data = common.MapStr{}
				}
				resp.data["headers"] = resp.header
			}

			if len(resp.data) > 0 {
				cassandra["response"] = resp.data
			}
		}
	}

	fields["status"] = status

	if len(cassandra) > 0 {
		fields["cassandra"] = cassandra
	}

	return evt
}
