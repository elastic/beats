package amqp

import (
	"encoding/binary"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/procs"
)

func (amqp *Amqp) amqpMessageParser(s *AmqpStream) (ok bool, complete bool) {
	for s.parseOffset < len(s.data) {

		if len(s.data[s.parseOffset:]) < 8 {
			logp.Warn("AMQP message smaller than a frame, waiting for more data")
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
			//incorrect header
			return false, false
		} else if f == nil {
			//header not complete
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
			logp.Warn("Received unknown AMQP frame")
			return false, false
		}

		// cast should be safe because f.size should not be bigger than tcp.TCP_MAX_DATA_IN_STREAM
		s.parseOffset += 8 + int(f.size)
		if !ok {
			return false, false
		} else if complete {
			return true, true
		}
	}
	return ok, complete
}

func (s *AmqpStream) PrepareForNewMessage() {
	s.message = nil
}

func isProtocolHeader(data []byte) (isHeader bool, version string) {

	if (string(data[:4]) == "AMQP") && data[4] == 0 {
		return true, string(data[5:8])
	}
	return false, ""
}

//func to read a frame header and check if it is valid and complete
func readFrameHeader(data []byte) (ret *AmqpFrame, err bool) {
	var frame AmqpFrame

	frame.size = binary.BigEndian.Uint32(data[3:7])
	if len(data) < int(frame.size)+8 {
		logp.Warn("Frame shorter than declared size, waiting for more data")
		return nil, false
	}
	if data[frame.size+7] != frameEndOctet {
		logp.Warn("Missing frame end octet in frame, discarding it")
		return nil, true
	}
	frame.Type = data[0]
	frame.channel = binary.BigEndian.Uint16(data[1:3])
	if frame.size == 0 {
		//frame content is nil with hearbeat frames
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

func (amqp *Amqp) decodeMethodFrame(s *AmqpStream, buf []byte) (bool, bool) {
	if len(buf) < 4 {
		logp.Warn("Method frame too small, waiting for more data")
		return true, false
	}
	class := codeClass(binary.BigEndian.Uint16(buf[0:2]))
	method := codeMethod(binary.BigEndian.Uint16(buf[2:4]))
	arguments := buf[4:]
	s.message.ParseArguments = amqp.ParseArguments
	s.message.BodySize = uint64(len(buf[4:]))

	debugf("Received frame of class %d and method %d", class, method)

	fn, exists := amqp.MethodMap[class][method]
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

func (amqp *Amqp) decodeHeaderFrame(s *AmqpStream, buf []byte) bool {
	if len(buf) < 14 {
		logp.Warn("Header frame too small, waiting for mode data")
		return true
	}
	s.message.BodySize = binary.BigEndian.Uint64(buf[4:12])
	debugf("Received Header frame. A message of %d bytes is expected", s.message.BodySize)

	if amqp.ParseHeaders == true {
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

func (s *AmqpStream) decodeBodyFrame(buf []byte) (ok bool, complete bool) {
	s.message.Body = append(s.message.Body, buf...)

	debugf("A body frame of %d bytes long has been transmitted",
		len(buf))
	//is the message complete ? If yes, let's publish it

	complete = uint64(len(s.message.Body)) >= s.message.BodySize
	return true, complete
}

func hasProperty(prop, flag byte) bool {
	return (prop & flag) == flag
}

//function to get message content-type and content-encoding
func getMessageProperties(s *AmqpStream, data []byte) bool {
	m := s.message

	//properties are coded in the two first bytes
	prop1 := data[0]
	prop2 := data[1]
	var offset uint32 = 2

	//while last bit set, we have another property flag field
	for lastbit := 1; data[lastbit]&1 == 1; {
		lastbit += 2
		offset += 2
	}

	if hasProperty(prop1, contentTypeProp) {
		contentType, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get content type in header frame")
			return true
		}
		m.Fields["content-type"] = contentType
		offset = next
	}

	if hasProperty(prop1, contentEncodingProp) {
		contentEncoding, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get content encoding in header frame")
			return true
		}
		m.Fields["content-encoding"] = contentEncoding
		offset = next
	}

	if hasProperty(prop1, headersProp) {
		headers := common.MapStr{}
		next, err, exists := getTable(headers, data, offset)
		if !err && exists {
			m.Fields["headers"] = headers
		} else if err {
			logp.Warn("Failed to get headers")
			return true
		}
		offset = next
	}

	if hasProperty(prop1, deliveryModeProp) {
		if data[offset] == 1 {
			m.Fields["delivery-mode"] = "non-persistent"
		} else if data[offset] == 2 {
			m.Fields["delivery-mode"] = "persistent"
		}
		offset++
	}

	if hasProperty(prop1, priorityProp) {
		m.Fields["priority"] = data[offset]
		offset++
	}

	if hasProperty(prop1, correlationIDProp) {
		correlationID, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get correlation-id in header frame")
			return true
		}
		m.Fields["correlation-id"] = correlationID
		offset = next
	}

	if hasProperty(prop1, replyToProp) {
		replyTo, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get reply-to in header frame")
			return true
		}
		m.Fields["reply-to"] = replyTo
		offset = next
	}

	if hasProperty(prop1, expirationProp) {
		expiration, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get expiration in header frame")
			return true
		}
		m.Fields["expiration"] = expiration
		offset = next
	}

	if hasProperty(prop2, messageIDProp) {
		messageID, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get message id in header frame")
			return true
		}
		m.Fields["message-id"] = messageID
		offset = next
	}

	if hasProperty(prop2, timestampProp) {
		t := time.Unix(int64(binary.BigEndian.Uint64(data[offset:offset+8])), 0)
		m.Fields["timestamp"] = t.Format(amqpTimeLayout)
		offset += 8
	}

	if hasProperty(prop2, typeProp) {
		msgType, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get message type in header frame")
			return true
		}
		m.Fields["type"] = msgType
		offset = next
	}

	if hasProperty(prop2, userIDProp) {
		userID, next, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get user id in header frame")
			return true
		}
		m.Fields["user-id"] = userID
		offset = next
	}

	if hasProperty(prop2, appIDProp) {
		appID, _, err := getShortString(data, offset+1, uint32(data[offset]))
		if err {
			logp.Warn("Failed to get app-id in header frame")
			return true
		}
		m.Fields["app-id"] = appID
	}
	return false
}

func (amqp *Amqp) handleAmqp(m *AmqpMessage, tcptuple *common.TCPTuple, dir uint8) {
	if amqp.mustHideCloseMethod(m) {
		return
	}
	debugf("A message is ready to be handled")
	m.TCPTuple = *tcptuple
	m.Direction = dir
	m.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(tcptuple.IPPort())

	if m.Method == "basic.publish" {
		amqp.handlePublishing(m)
	} else if m.Method == "basic.deliver" || m.Method == "basic.return" ||
		m.Method == "basic.get-ok" {
		amqp.handleDelivering(m)
	} else if m.IsRequest == true {
		amqp.handleAmqpRequest(m)
	} else if m.IsRequest == false {
		amqp.handleAmqpResponse(m)
	}
}

func (amqp *Amqp) mustHideCloseMethod(m *AmqpMessage) bool {
	return amqp.HideConnectionInformation == true &&
		(m.Method == "connection.close" || m.Method == "channel.close") &&
		getReplyCode(m.Fields) < uint16(300)
}
