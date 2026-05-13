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
	"encoding/binary"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (amqp *amqpPlugin) amqpMessageParser(s *amqpStream) (ok bool, complete bool) {
	for s.parseOffset < len(s.data) {

		if len(s.data[s.parseOffset:]) < 8 {
			logp.Debug("amqp", "AMQP message smaller than a frame, waiting for more data")
			return true, false
		}

		yes, version := isProtocolHeader(s.data[s.parseOffset:])
		if yes {
			debugf("Client header detected, version %d.%d.%d",
				version[0], version[1], version[2])
			s.parseOffset += 8
		}

		f, err := readFrameHeader(s.data[s.parseOffset:])
		if err {
			// incorrect header
			return false, false
		} else if f == nil {
			// header not complete
			return true, false
		}

		switch f.Type {
		case methodType:
			ok, complete = amqp.decodeMethodFrame(s, f.content)
		case headerType:
			ok = amqp.decodeHeaderFrame(s, f.content)
		case bodyType:
			ok, complete = s.decodeBodyFrame(f.content)
		case heartbeatType:
			detailedf("Heartbeat frame received")
		default:
			logp.Debug("amqp", "Received unknown AMQP frame")
			return false, false
		}

		// cast should be safe because f.size should not be bigger than tcp.TCP_MAX_DATA_IN_STREAM
		s.parseOffset += 8 + int(f.size)
		if !ok {
			return false, false
		}
		if complete {
			return true, true
		}
	}
	return ok, complete
}

func isProtocolHeader(data []byte) (isHeader bool, version string) {
	if (string(data[:4]) == "AMQP") && data[4] == 0 {
		return true, string(data[5:8])
	}
	return false, ""
}

// func to read a frame header and check if it is valid and complete
func readFrameHeader(data []byte) (ret *amqpFrame, err bool) {
	var frame amqpFrame
	if len(data) < 8 {
		logp.Debug("amqp", "Partial frame header, waiting for more data")
		return nil, false
	}
	frame.size = binary.BigEndian.Uint32(data[3:7])
	if len(data) < int(frame.size)+8 {
		logp.Debug("amqp", "Frame shorter than declared size, waiting for more data")
		return nil, false
	}
	if data[frame.size+7] != frameEndOctet {
		logp.Debug("amqp", "Missing frame end octet in frame, discarding it")
		return nil, true
	}
	frame.Type = frameType(data[0])
	if frame.size == 0 {
		// frame content is nil with heartbeat frames
		frame.content = nil
	} else {
		frame.content = data[7 : frame.size+7]
	}
	return &frame, false
}

/*
The Method Payload, according to official doc :
0           2           4
+----------+-----------+-------------- - -
| class-id | method-id | arguments...
+----------+-----------+-------------- - -
  short       short       ...
*/

func (amqp *amqpPlugin) decodeMethodFrame(s *amqpStream, buf []byte) (bool, bool) {
	if len(buf) < 4 {
		logp.Debug("amqp", "Method frame too small, waiting for more data")
		return true, false
	}
	class := codeClass(binary.BigEndian.Uint16(buf[0:2]))
	method := codeMethod(binary.BigEndian.Uint16(buf[2:4]))
	arguments := buf[4:]
	s.message.parseArguments = amqp.parseArguments
	s.message.bodySize = uint64(len(buf[4:]))

	debugf("Received frame of class %d and method %d", class, method)

	fn, exists := amqp.methodMap[class][method]
	if !exists {
		logp.Debug("amqpdetailed", "Received unknown or not supported method")
		return false, false
	}

	return fn(s.message, arguments)
}

/*
Structure of a content header, according to official doc :
0           2        4          12               14
+----------+--------+-----------+----------------+------------- - -
| class-id | weight | body size | property flags | property list...
+----------+--------+-----------+----------------+------------- - -
  short      short   long long        short         remainder...
*/

func (amqp *amqpPlugin) decodeHeaderFrame(s *amqpStream, buf []byte) bool {
	if len(buf) < 14 {
		logp.Debug("amqp", "Header frame too small, waiting for mode data")
		return true
	}
	s.message.bodySize = binary.BigEndian.Uint64(buf[4:12])
	debugf("Received Header frame. A message of %d bytes is expected", s.message.bodySize)

	if amqp.parseHeaders {
		err := getMessageProperties(s, buf[12:])
		if err {
			return false
		}
	}
	return true
}

/*
Structure of a body frame, according to official doc :
+-----------------------+ +-----------+
| Opaque binary payload | | frame-end |
+-----------------------+ +-----------+
*/

func (s *amqpStream) decodeBodyFrame(buf []byte) (ok bool, complete bool) {
	s.message.body = append(s.message.body, buf...)

	debugf("A body frame of %d bytes long has been transmitted",
		len(buf))
	// is the message complete ? If yes, let's publish it

	complete = uint64(len(s.message.body)) >= s.message.bodySize
	return true, complete
}

func hasProperty(prop, flag byte) bool {
	return (prop & flag) == flag
}

// function to get message content-type and content-encoding
func getMessageProperties(s *amqpStream, data []byte) bool {
	m := s.message

	if len(data) < 2 {
		logp.Debug("amqp", "Malformed packet: unexpected end of data")
		return true
	}

	// properties are coded in the two first bytes
	prop1 := data[0]
	prop2 := data[1]
	var offset uint32 = 2

	// while last bit set, we have another property flag field
	lastbit := 1
	for {
		if data[lastbit]&1 == 1 {
			lastbit += 2
			offset += 2
		} else {
			break
		}
		if lastbit >= len(data) {
			logp.Debug("amqp", "Malformed packet: unexpected end of data")
			return true
		}
	}

	logp.Debug("amqp", "offset:%d, lastbit:%d", offset, lastbit)

	if hasProperty(prop1, contentTypeProp) {
		contentType, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get content type in header frame")
			return true
		}
		m.fields["content-type"] = contentType
		offset += consumed
	}

	if hasProperty(prop1, contentEncodingProp) {
		contentEncoding, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get content encoding in header frame")
			return true
		}
		m.fields["content-encoding"] = contentEncoding
		offset += consumed
	}

	if hasProperty(prop1, headersProp) {
		if int(offset) >= len(data) {
			logp.Debug("amqp", "Malformed packet: unexpected end of data")
			return true
		}
		headers := mapstr.M{}
		next, err, exists := getTable(headers, data, offset)
		if !err && exists {
			m.fields["headers"] = headers
		} else if err {
			logp.Debug("amqp", "Failed to get headers")
			return true
		}
		offset = next
	}

	if hasProperty(prop1, deliveryModeProp) {
		if int(offset) >= len(data) {
			logp.Debug("amqp", "Malformed packet: unexpected end of data")
			return true
		}
		switch data[offset] {
		case 1:
			m.fields["delivery-mode"] = "non-persistent"
		case 2:
			m.fields["delivery-mode"] = "persistent"
		}
		offset++
	}

	if hasProperty(prop1, priorityProp) {
		if int(offset) >= len(data) {
			logp.Debug("amqp", "Malformed packet: unexpected end of data")
			return true
		}
		m.fields["priority"] = data[offset]
		offset++
	}

	if hasProperty(prop1, correlationIDProp) {
		correlationID, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get correlation-id in header frame")
			return true
		}
		m.fields["correlation-id"] = correlationID
		offset += consumed
	}

	if hasProperty(prop1, replyToProp) {
		replyTo, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get reply-to in header frame")
			return true
		}
		m.fields["reply-to"] = replyTo
		offset += consumed
	}

	if hasProperty(prop1, expirationProp) {
		expiration, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get expiration in header frame")
			return true
		}
		m.fields["expiration"] = expiration
		offset += consumed
	}

	if hasProperty(prop2, messageIDProp) {
		messageID, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get message id in header frame")
			return true
		}
		m.fields["message-id"] = messageID
		offset += consumed
	}

	if hasProperty(prop2, timestampProp) {
		timeInt, err := getIntegerAt[int64](data, offset)
		if err {
			logp.Debug("amqp", "Malformed packet: unexpected end of data")
			return true
		}
		t := time.Unix(timeInt, 0)
		m.fields["timestamp"] = t.Format(amqpTimeLayout)
		offset += 8
	}

	if hasProperty(prop2, typeProp) {
		msgType, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get message type in header frame")
			return true
		}
		m.fields["type"] = msgType
		offset += consumed
	}

	if hasProperty(prop2, userIDProp) {
		userID, consumed, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get user id in header frame")
			return true
		}
		m.fields["user-id"] = userID
		offset += consumed
	}

	if hasProperty(prop2, appIDProp) {
		appID, _, err := getLVString[uint8](data, offset)
		if err {
			logp.Debug("amqp", "Failed to get app-id in header frame")
			return true
		}
		m.fields["app-id"] = appID
	}
	return false
}

func (amqp *amqpPlugin) handleAmqp(m *amqpMessage, tcptuple *common.TCPTuple, dir uint8) {
	if amqp.mustHideCloseMethod(m) {
		return
	}
	debugf("A message is ready to be handled")
	m.tcpTuple = *tcptuple
	m.direction = dir
	m.cmdlineTuple = amqp.watcher.FindProcessesTupleTCP(tcptuple.IPPort())

	switch {
	case m.method == "basic.publish":
		amqp.handlePublishing(m)
	case m.method == "basic.deliver" || m.method == "basic.return" || m.method == "basic.get-ok":
		amqp.handleDelivering(m)
	case m.isRequest:
		amqp.handleAmqpRequest(m)
	default: // !m.isRequest
		amqp.handleAmqpResponse(m)
	}
}

func (amqp *amqpPlugin) mustHideCloseMethod(m *amqpMessage) bool {
	return amqp.hideConnectionInformation &&
		(m.method == "connection.close" || m.method == "channel.close") &&
		getReplyCode(m.fields) < 300
}
