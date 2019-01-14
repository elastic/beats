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

package orientdb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func orientdbMessageParser(s *stream) (bool, bool) {
	d := newDecoder(s.data)
	fmt.Println(s.data)

	if len(s.data) == 0 {
		// Not even enough data to parse length of message
		return true, false
	}

	code, _ := d.readByte()

	opCode := opCode(code)

	// fill up the header common to all messages
	sessionID, _ := d.readInt32()
	s.message.sessionID = int(sessionID)

	if !validOpcode(opCode) {
		logp.Err("Unknown operation code: %v", opCode)
		return false, false
	}

	s.message.opCode = opCode
	debugf("opCode = %v", s.message.opCode)

	s.message.event = common.MapStr{}

	switch s.message.opCode {
	case opRequestConnect:
		s.message.method = "connect"
		ok, completed := opRequestConnectParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestDbOpen:
		s.message.method = "dbOpen"
		ok, completed := opRequestDbOpenParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestDbList:
		s.message.method = "dbList"
		ok, completed := opRequestDbListParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestDbClose:
		s.message.method = "dbClose"
		ok, completed := opRequestDbCloseParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestShutdown:
		s.message.method = "shutdown"
		ok, completed := opRequestShutdownParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestDataClusterAdd:
		s.message.method = "addCluster"
		ok, completed := opRequestDataClusterAddParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestDataClusterCount:
		s.message.method = "clusterCount"
		ok, completed := opRequestDataClusterCountParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestRecordCreate:
		s.message.method = "recordCreate"
		ok, completed := opRequestRecordCreateParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestRecordRead:
		s.message.method = "recordLoad"
		ok, completed := opRequestRecordReadParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestRecordUpdate:
		s.message.method = "recordUpdate"
		ok, completed := opRequestRecordUpdateParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestRecordDelete:
		s.message.method = "recordDelete"
		ok, completed := opRequestRecordDeleteParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	case opRequestCommand:
		s.message.method = "commandLoad"
		ok, completed := opRequestCommandParse(d, s.message)
		s.message.messageLength = d.i
		return ok, completed
	}
	return false, false
}

func opRequestConnectParse(d *decoder, o *orientdbMessage) (bool, bool) {
	clientName, _ := d.readString()
	clientVersion, _ := d.readString()

	protocolVersion, _ := d.readShort()

	if protocolVersion > 21 {
		clientID, _ := d.readString()
		serializationType, _ := d.readString()
		if protocolVersion > 26 {
			_, _ = d.readBoolean()
			if protocolVersion >= 36 {
				_, _ = d.readBoolean()
				_, _ = d.readBoolean()
			}
		}
		_, _ = d.readString()
		_, _ = d.readString()
		o.event["clientName"] = clientName
		o.event["clientVersion"] = clientVersion
		o.event["clientID"] = clientID
		o.event["serializationType"] = serializationType
	} else {
		clientID, _ := d.readString()
		_, _ = d.readString()
		_, _ = d.readString()
		o.event["clientID"] = clientID
	}
	return true, true
}

func opRequestDbOpenParse(d *decoder, o *orientdbMessage) (bool, bool) {
	clientName, _ := d.readString()
	clientVersion, _ := d.readString()

	protocolVersion, _ := d.readShort()

	if protocolVersion > 21 {
		clientID, _ := d.readString()
		serializationType, _ := d.readString()
		if protocolVersion > 26 {
			_, _ = d.readBoolean()
			if protocolVersion >= 36 {
				_, _ = d.readBoolean()
				_, _ = d.readBoolean()
			}
		}
		database, _ := d.readString()

		if protocolVersion < 33 {
			_, _ = d.readString()
		}
		_, _ = d.readString()
		_, _ = d.readString()
		o.event["clientName"] = clientName
		o.event["clientVersion"] = clientVersion
		o.event["clientID"] = clientID
		o.event["serializationType"] = serializationType
		o.event["database"] = database
	}
	return true, true
}

func opRequestDbListParse(d *decoder, o *orientdbMessage) (bool, bool) {
	return true, true
}

func opRequestDbCloseParse(d *decoder, o *orientdbMessage) (bool, bool) {
	return true, true
}

func opRequestShutdownParse(d *decoder, o *orientdbMessage) (bool, bool) {
	_, _ = d.readString()
	_, _ = d.readString()
	return true, true
}

func opRequestDataClusterAddParse(d *decoder, o *orientdbMessage) (bool, bool) {
	clusterName, _ := d.readString()
	clusterID, _ := d.readShort()
	o.event["clusterName"] = clusterName
	o.event["clusterID"] = clusterID
	return true, true
}

func opRequestDataClusterCountParse(d *decoder, o *orientdbMessage) (bool, bool) {
	length, _ := d.readShort()
	clusterIds := []int16{}
	for count := int16(1); count <= length; count++ {
		clusterID, _ := d.readShort()
		clusterIds = append(clusterIds, clusterID)
	}
	_, _ = d.readBoolean()
	o.event["clusterIds"] = strings.Trim(strings.Replace(fmt.Sprint(clusterIds),
		" ", ", ", -1), "[]")
	return true, true
}

func opRequestRecordCreateParse(d *decoder, o *orientdbMessage) (bool, bool) {
	clusterID, _ := d.readShort()

	record, _ := d.readByteArray()
	recordType, _ := d.readByte()
	_, _ = d.readBoolean()
	o.event["clusterID"] = clusterID

	deserializedRecord := deserialize(record)

	o.event["recordClass"] = deserializedRecord.oClass
	o.event["recordType"] = recordType
	o.resource = strconv.Itoa(int(clusterID))
	return true, true
}

func opRequestRecordReadParse(d *decoder, o *orientdbMessage) (bool, bool) {
	clusterID, _ := d.readShort()

	clusterPosition, _ := d.readLong()
	fetchPlan, _ := d.readString()
	ignoreCache, _ := d.readBoolean()
	_, _ = d.readBoolean()
	o.event["clusterID"] = clusterID
	o.event["clusterPosition"] = clusterPosition
	o.event["fetchPlan"] = fetchPlan
	o.event["ignoreCache"] = ignoreCache
	o.resource = strconv.Itoa(int(clusterID)) + "-" + strconv.Itoa(int(clusterPosition))
	return true, true
}

func opRequestRecordUpdateParse(d *decoder, o *orientdbMessage) (bool, bool) {
	clusterID, _ := d.readShort()

	clusterPosition, _ := d.readLong()
	updateContent, _ := d.readBoolean()

	record, _ := d.readByteArray()
	recordVersion, _ := d.readInt32()
	recordType, _ := d.readByte()
	_, _ = d.readBoolean()
	o.event["clusterID"] = clusterID
	o.event["clusterPosition"] = clusterPosition
	o.event["updateContent"] = updateContent

	deserializedRecord := deserialize(record)

	o.event["recordClass"] = deserializedRecord.oClass
	o.event["recordVersion"] = recordVersion
	o.event["recordType"] = fmt.Sprintf("%c", recordType)
	o.resource = strconv.Itoa(int(clusterID)) + "-" + strconv.Itoa(int(clusterPosition))
	return true, true
}

func opRequestRecordDeleteParse(d *decoder, o *orientdbMessage) (bool, bool) {
	clusterID, _ := d.readShort()

	clusterPosition, _ := d.readLong()
	recordVersion, _ := d.readInt32()
	_, _ = d.readBoolean()
	o.event["clusterID"] = clusterID
	o.event["clusterPosition"] = clusterPosition
	o.event["recordVersion"] = recordVersion
	o.resource = strconv.Itoa(int(clusterID)) + "-" + strconv.Itoa(int(clusterPosition))
	return true, true
}

func opRequestCommandParse(d *decoder, o *orientdbMessage) (bool, bool) {
	modByte, _ := d.readByte()

	payload, _ := d.readByteArray()

	commandLength, _ := unpackInt(payload[0:4])
	command, _ := unpackString(payload[4 : 4+commandLength])
	payload = payload[4+commandLength:]
	if command == "com.orientechnologies.orient.core.command.script.OCommandScript" {
		commandTypeLength, _ := unpackInt(payload[0:4])
		commandType, _ := unpackString(payload[4 : 4+commandTypeLength])
		payload = payload[4+commandTypeLength:]
		o.event["commandType"] = commandType
	}

	queryLength, _ := unpackInt(payload[0:4])
	query, _ := unpackString(payload[4 : 4+queryLength])
	payload = payload[4+queryLength:]
	o.event["query"] = query

	if command == "com.orientechnologies.orient.core.sql.query.OSQLSynchQuery" ||
		command == "com.orientechnologies.orient.core.sql.query.OSQLAsynchQuery" ||
		command == "com.orientechnologies.orient.graph.gremlin.OCommandGremlin" {
		limit, _ := unpackInt(payload[0:4])
		payload = payload[4:]
		o.event["limit"] = strconv.Itoa(limit)

		fetchPlanLength, _ := unpackInt(payload[0:4])
		fetchPlan, _ := unpackString(payload[4 : 4+fetchPlanLength])
		o.event["fetchPlan"] = fetchPlan
	}

	o.event["modByte"] = fmt.Sprintf("%c", modByte)
	return true, true
}

type decoder struct {
	in []byte
	i  int
}

func newDecoder(in []byte) *decoder {
	return &decoder{in: in, i: 0}
}

func (d *decoder) readString() (string, error) {
	length, err := d.readInt32()
	if err != nil {
		return "", err
	}

	data, err := d.readBytes(length)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (d *decoder) readByte() (byte, error) {
	b, err := d.readBytes(1)

	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (d *decoder) readBoolean() (bool, error) {
	b, err := d.readBytes(1)

	if err != nil {
		return false, err
	}

	if b[0] == 1 {
		return true, nil
	}
	return false, nil
}

func (d *decoder) readShort() (int16, error) {
	b, err := d.readBytes(2)

	if err != nil {
		return 0, err
	}
	return int16((uint16(b[1]) << 0) |
		(uint16(b[0]) << 8)), nil
}

func (d *decoder) readLong() (int64, error) {
	b, err := d.readBytes(8)

	if err != nil {
		return 0, err
	}
	return int64((uint64(b[7]) << 0) |
		(uint64(b[6]) << 8) |
		(uint64(b[5]) << 16) |
		(uint64(b[4]) << 24) |
		(uint64(b[3]) << 32) |
		(uint64(b[2]) << 40) |
		(uint64(b[1]) << 48) |
		(uint64(b[0]) << 56)), nil
}

func (d *decoder) readInt32() (int32, error) {
	b, err := d.readBytes(4)

	if err != nil {
		return 0, err
	}

	return int32((uint32(b[3]) << 0) |
		(uint32(b[2]) << 8) |
		(uint32(b[1]) << 16) |
		(uint32(b[0]) << 24)), nil
}

func (d *decoder) readByteArray() ([]byte, error) {
	length, err := d.readInt32()
	if err != nil {
		return []byte{}, err
	}

	data, err := d.readBytes(length)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (d *decoder) readBytes(length int32) ([]byte, error) {
	start := d.i
	d.i += int(length)
	if d.i > len(d.in) {
		return *new([]byte), errors.New("No byte to read")
	}
	return d.in[start : start+int(length)], nil
}
