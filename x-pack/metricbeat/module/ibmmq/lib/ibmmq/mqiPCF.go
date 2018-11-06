package ibmmqi

/*
  Copyright (c) IBM Corporation 2016

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific

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
	"fmt"
)

/*
MQCFH is a structure containing the MQ PCF Header fields
*/
type MQCFH struct {
	Type           int32
	StrucLength    int32
	Version        int32
	Command        int32
	MsgSeqNumber   int32
	Control        int32
	CompCode       int32
	Reason         int32
	ParameterCount int32
}

var endian binary.ByteOrder

/*
PCFParameter is a structure containing the data associated with
various types of PCF element. Use the Type field to decide which
of the data fields is relevant.
*/
type PCFParameter struct {
	Type           int32
	Parameter      int32
	Int64Value     []int64 // Always store as 64; cast to 32 when needed
	String         []string
	CodedCharSetId int32
	ParameterCount int32
	GroupList      []*PCFParameter
	strucLength    int32 // Do not need to expose these
	stringLength   int32 // lengths
}

/*
NewMQCFH returns a PCF Command Header structure with correct initialisation
*/
func NewMQCFH() *MQCFH {
	cfh := new(MQCFH)
	cfh.Type = C.MQCFT_COMMAND
	cfh.StrucLength = C.MQCFH_STRUC_LENGTH
	cfh.Version = C.MQCFH_VERSION_1
	cfh.Command = C.MQCMD_NONE
	cfh.MsgSeqNumber = 1
	cfh.Control = C.MQCFC_LAST
	cfh.CompCode = C.MQCC_OK
	cfh.Reason = C.MQRC_NONE
	cfh.ParameterCount = 0

	if (C.MQENC_NATIVE % 2) == 0 {
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	return cfh
}

/*
Bytes serialises an MQCFH structure as if it were the corresponding C structure
*/
func (cfh *MQCFH) Bytes() []byte {

	buf := make([]byte, cfh.StrucLength)
	offset := 0

	endian.PutUint32(buf[offset:], uint32(cfh.Type))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.StrucLength))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Version))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Command))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.MsgSeqNumber))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Control))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.CompCode))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Reason))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.ParameterCount))
	offset += 4

	return buf
}

/*
Bytes serialises a PCFParameter into the C structure
corresponding to its type
*/
func (p *PCFParameter) Bytes() []byte {
	var buf []byte

	switch p.Type {
	case C.MQCFT_INTEGER:
		buf = make([]byte, C.MQCFIN_STRUC_LENGTH)
		offset := 0

		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Int64Value[0]))
		offset += 4

	case C.MQCFT_STRING:
		buf = make([]byte, C.MQCFST_STRUC_LENGTH_FIXED+roundTo4(int32(len(p.String[0]))))
		offset := 0
		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(C.MQCCSI_DEFAULT))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(p.String[0])))
		offset += 4
		copy(buf[offset:], []byte(p.String[0]))
	}
	return buf
}

/*
ReadPCFHeader extracts the MQCFH from an MQ message
*/
func ReadPCFHeader(buf []byte) (*MQCFH, int) {
	cfh := new(MQCFH)
	fullLen := len(buf)
	p := bytes.NewBuffer(buf)

	binary.Read(p, endian, &cfh.Type)
	binary.Read(p, endian, &cfh.StrucLength)
	binary.Read(p, endian, &cfh.Version)
	binary.Read(p, endian, &cfh.Command)
	binary.Read(p, endian, &cfh.MsgSeqNumber)
	binary.Read(p, endian, &cfh.Control)
	binary.Read(p, endian, &cfh.CompCode)
	binary.Read(p, endian, &cfh.Reason)
	binary.Read(p, endian, &cfh.ParameterCount)

	bytesRead := fullLen - p.Len()
	return cfh, bytesRead
}

/*
ReadPCFParameter extracts the next PCF parameter element from an
MQ message.
*/
func ReadPCFParameter(buf []byte) (*PCFParameter, int) {
	var i32 int32
	var i64 int64
	var mqlong int32
	var count int32

	pcfParm := new(PCFParameter)
	fullLen := len(buf)
	p := bytes.NewBuffer(buf)

	binary.Read(p, endian, &pcfParm.Type)
	binary.Read(p, endian, &pcfParm.strucLength)

	switch pcfParm.Type {
	// There are more PCF element types but the samples only
	// needed a subset
	case C.MQCFT_INTEGER:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &i32)
		pcfParm.Int64Value = append(pcfParm.Int64Value, int64(i32))

	case C.MQCFT_INTEGER_LIST:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &count)
		for i := 0; i < int(count); i++ {
			binary.Read(p, endian, &i32)
			pcfParm.Int64Value = append(pcfParm.Int64Value, int64(i32))
		}

	case C.MQCFT_INTEGER64:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &mqlong) // Used for alignment
		binary.Read(p, endian, &i64)
		pcfParm.Int64Value = append(pcfParm.Int64Value, i64)

	case C.MQCFT_INTEGER64_LIST:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &count)
		for i := 0; i < int(count); i++ {
			binary.Read(p, endian, &i64)
			pcfParm.Int64Value = append(pcfParm.Int64Value, i64)
		}

	case C.MQCFT_STRING:
		offset := int32(C.MQCFST_STRUC_LENGTH_FIXED)
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &pcfParm.CodedCharSetId)
		binary.Read(p, endian, &pcfParm.stringLength)
		s := string(buf[offset : pcfParm.stringLength+offset])
		pcfParm.String = append(pcfParm.String, s)
		p.Next(int(pcfParm.strucLength - offset))

	case C.MQCFT_STRING_LIST:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &pcfParm.CodedCharSetId)
		binary.Read(p, endian, &count)
		binary.Read(p, endian, &pcfParm.stringLength)
		for i := 0; i < int(count); i++ {
			offset := C.MQCFSL_STRUC_LENGTH_FIXED + i*int(pcfParm.stringLength)
			s := string(buf[offset : int(pcfParm.stringLength)+offset])
			pcfParm.String = append(pcfParm.String, s)
		}
		p.Next(int(pcfParm.strucLength - C.MQCFSL_STRUC_LENGTH_FIXED))

	case C.MQCFT_GROUP:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &pcfParm.ParameterCount)
	default:
		fmt.Println("mqiPCF.go: Unknown PCF type ", pcfParm.Type)
		// Skip the remains of this structure, assuming it really is
		// PCF and we just don't know how to process the element type
		p.Next(int(pcfParm.strucLength - 8))
	}

	bytesRead := fullLen - p.Len()
	return pcfParm, bytesRead
}

func roundTo4(u int32) int32 {
	return ((u) + ((4 - ((u) % 4)) % 4))
}
