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

package smtp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/mail"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/packetbeat/pb"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
)

// Transaction Publisher.
type transPub struct {
	sendRequest     bool
	sendResponse    bool
	sendDataHeaders bool
	sendDataBody    bool

	results protos.Reporter
}

func (pub *transPub) onTransaction(
	trans transaction,
	sessionID uuid.UUID,
) error {
	if pub.results == nil {
		return nil
	}

	pub.results(pub.createEvent(trans, sessionID))

	return nil
}

func (pub *transPub) createEvent(
	trans transaction,
	sessionID uuid.UUID,
) beat.Event {
	d := common.MapStr{"session_id": sessionID}
	drequ := common.MapStr{}
	dresp := common.MapStr{}

	var fields common.MapStr
	var evt beat.Event
	var pbf *pb.Fields
	var src, dst *common.Endpoint

	switch t := trans.(type) {
	case *transPrompt:
		evt, pbf = pb.NewBeatEvent(t.resp.Ts)
		fields = evt.Fields

		d["type"] = "PROMPT"
		if pub.sendResponse {
			fields["response"] =
				common.NetString(t.resp.raw.BufferedBytes())
		}
		if t.resp.statusCode >= 400 {
			fields["status"] = common.SERVER_ERROR_STATUS
		}
		fields["bytes_out"] = t.resp.Size
		dresp["code"] = t.resp.statusCode
		if len(t.resp.statusPhrases) > 0 {
			dresp["phrases"] = t.resp.statusPhrases
		}
		src, dst = pub.getEndpoints(t.resp)
		pbf.Network.Transport = t.resp.Transport.String()

	case *transCommand:
		evt, pbf = pb.NewBeatEvent(t.resp.Ts)
		fields = evt.Fields

		d["type"] = "COMMAND"
		if pub.sendRequest {
			fields["request"] =
				common.NetString(t.requ.raw.BufferedBytes())
		}
		if pub.sendResponse {
			fields["response"] =
				common.NetString(t.resp.raw.BufferedBytes())
		}
		fields["bytes_in"] = t.requ.Size
		fields["bytes_out"] = t.resp.Size
		if t.resp.statusCode >= 400 {
			fields["status"] = common.SERVER_ERROR_STATUS
		}
		// Response time in milliseconds
		pbf.Event.Duration = t.resp.Ts.Sub(t.requ.Ts)

		drequ["command"] = t.requ.command
		drequ["param"] = t.requ.param
		dresp["code"] = t.resp.statusCode
		dresp["phrases"] = t.resp.statusPhrases
		src, dst = pub.getEndpoints(t.requ)
		pbf.Network.Transport = t.resp.Transport.String()

	case *transMail:
		evt, pbf = pb.NewBeatEvent(t.resp.Ts)
		fields = evt.Fields

		d["type"] = "MAIL"
		if t.reversePath != nil {
			d["envelope_sender"] = t.reversePath
		}
		if t.forwardPaths != nil {
			d["envelope_recipients"] = t.forwardPaths
		}
		if bytes.Equal(t.requ.command, constEOD) {
			headers, body, err := pub.parsePayload(t)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse data payload: %s", err)
				t.Notes = append(t.Notes, msg)
				debugf(msg)
			} else {
				if pub.sendDataHeaders {
					d["headers"] = headers
				}
				if pub.sendDataBody {
					d["body"] = body
				}
			}
		}
		fields["bytes_in"] = t.BytesIn
		fields["bytes_out"] = t.BytesOut
		fields["status"] = t.Status
		if len(t.Notes) > 0 {
			fields["notes"] = t.Notes
		}
		src, dst = pub.getEndpoints(t.requ)
		pbf.Network.Transport = t.resp.Transport.String()
	}

	pbf.SetSource(src)
	pbf.SetDestination(dst)

	pbf.Event.Start = evt.Timestamp
	pbf.Event.Dataset = "smtp"
	pbf.Network.Protocol = pbf.Event.Dataset

	fields["type"] = "smtp"
	if fields["status"] == nil {
		fields["status"] = common.OK_STATUS
	}

	if len(drequ) > 0 {
		d["request"] = drequ
	}
	if len(dresp) > 0 {
		d["response"] = dresp
	}

	fields["smtp"] = d

	return evt
}

func (pub *transPub) parsePayload(t *transMail) (
	map[string]common.NetString,
	common.NetString,
	error,
) {
	if !pub.sendDataHeaders && !pub.sendDataBody {
		return nil, nil, nil
	}

	var headers map[string]common.NetString
	var body []byte
	var err error

	payload, err := mail.ReadMessage(&t.requ.raw)
	if err != nil {
		return nil, nil, err
	}

	headers = make(map[string]common.NetString)

	for k := range payload.Header {
		headers[k] = common.NetString(payload.Header.Get(k))
	}

	if body, err = ioutil.ReadAll(payload.Body); err != nil {
		return nil, nil, err
	}

	return headers, body, nil
}

func (pub *transPub) getEndpoints(m *message) (
	*common.Endpoint,
	*common.Endpoint,
) {
	src := &common.Endpoint{
		IP:      m.Tuple.SrcIP.String(),
		Port:    m.Tuple.SrcPort,
		Process: m.CmdlineTuple.Src,
	}
	dst := &common.Endpoint{
		IP:      m.Tuple.DstIP.String(),
		Port:    m.Tuple.DstPort,
		Process: m.CmdlineTuple.Dst,
	}
	if m.Direction == tcp.TCPDirectionReverse {
		src, dst = dst, src
	}

	return src, dst
}
