package ibmmqi

/*
  Copyright (c) IBM Corporation 2016,2018

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
#include <stdlib.h>
#include <cmqc.h>
#include <cmqcfc.h>
*/
import "C"

import (
	"bytes"
	"encoding/binary"
)

type MQDLH struct {
	Reason         int32
	DestQName      string
	DestQMgrName   string
	Encoding       int32
	CodedCharSetId int32
	Format         string
	PutApplType    int32
	PutApplName    string
	PutDate        string
	PutTime        string
	strucLength    int // Not exported
}

func NewMQDLH(md *MQMD) *MQDLH {
	dlh := new(MQDLH)
	dlh.Reason = MQRC_NONE
	dlh.CodedCharSetId = MQCCSI_UNDEFINED
	dlh.PutApplType = 0
	dlh.PutApplName = ""
	dlh.PutTime = ""
	dlh.PutDate = ""
	dlh.Format = ""
	dlh.DestQName = ""
	dlh.DestQMgrName = ""

	dlh.strucLength = int(MQDLH_CURRENT_LENGTH)

	if md != nil {
		dlh.Encoding = md.Encoding
		if md.CodedCharSetId == MQCCSI_DEFAULT {
			dlh.CodedCharSetId = MQCCSI_INHERIT
		} else {
			dlh.CodedCharSetId = md.CodedCharSetId
		}
		dlh.Format = md.Format

		md.Format = MQFMT_DEAD_LETTER_HEADER
		md.MsgType = MQMT_REPORT
		md.CodedCharSetId = MQCCSI_Q_MGR
	}

	if (C.MQENC_NATIVE % 2) == 0 {
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	return dlh
}

func (dlh *MQDLH) Bytes() []byte {
	buf := make([]byte, dlh.strucLength)
	offset := 0

	copy(buf[offset:], "DLH ")
	offset += 4
	endian.PutUint32(buf[offset:], uint32(MQDLH_CURRENT_VERSION))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(dlh.Reason))
	offset += 4
	copy(buf[offset:], dlh.DestQName)
	offset += int(MQ_OBJECT_NAME_LENGTH)
	copy(buf[offset:], dlh.DestQMgrName)
	offset += int(MQ_Q_MGR_NAME_LENGTH)
	endian.PutUint32(buf[offset:], uint32(dlh.Encoding))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(dlh.CodedCharSetId))
	offset += 4
	copy(buf[offset:], dlh.Format)
	offset += int(MQ_FORMAT_LENGTH)
	endian.PutUint32(buf[offset:], uint32(dlh.PutApplType))
	offset += 4
	copy(buf[offset:], dlh.PutApplName)
	offset += int(MQ_PUT_APPL_NAME_LENGTH)
	copy(buf[offset:], dlh.PutDate)
	offset += int(MQ_PUT_DATE_LENGTH)
	copy(buf[offset:], dlh.PutTime)
	offset += int(MQ_PUT_TIME_LENGTH)

	return buf
}

/*
We have a byte array for the message contents. The start of that buffer
is the MQDLH structure. We read the bytes from that fixed header to match
the C structure definition for each field. The DLH does not have multiple
versions defined so we don't need to check that as we go through.
*/
func getHeaderDLH(md *MQMD, buf []byte) (*MQDLH, int, error) {

	var version int32

	dlh := NewMQDLH(nil)

	r := bytes.NewBuffer(buf)
	_ = readStringFromFixedBuffer(r, 4) // StrucId
	binary.Read(r, endian, &version)
	binary.Read(r, endian, &dlh.Reason)
	dlh.DestQName = readStringFromFixedBuffer(r, MQ_OBJECT_NAME_LENGTH)
	dlh.DestQMgrName = readStringFromFixedBuffer(r, MQ_Q_MGR_NAME_LENGTH)

	binary.Read(r, endian, &dlh.Encoding)
	binary.Read(r, endian, &dlh.CodedCharSetId)

	dlh.Format = readStringFromFixedBuffer(r, MQ_FORMAT_LENGTH)

	binary.Read(r, endian, &dlh.PutApplType)

	dlh.PutApplName = readStringFromFixedBuffer(r, MQ_PUT_APPL_NAME_LENGTH)
	dlh.PutDate = readStringFromFixedBuffer(r, MQ_PUT_DATE_LENGTH)
	dlh.PutTime = readStringFromFixedBuffer(r, MQ_PUT_TIME_LENGTH)

	return dlh, dlh.strucLength, nil
}
