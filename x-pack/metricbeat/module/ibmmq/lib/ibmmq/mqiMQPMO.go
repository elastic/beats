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
#include <string.h>
#include <cmqc.h>

*/
import "C"

/*
MQPMO is a structure containing the MQ Put MessageOptions (MQPMO)
*/
type MQPMO struct {
	Version           int32
	Options           int32
	Timeout           int32
	Context           C.MQHOBJ
	KnownDestCount    int32
	UnknownDestCount  int32
	InvalidDestCount  int32
	ResolvedQName     string
	ResolvedQMgrName  string
	RecsPresent       int32
	PutMsgRecFields   int32
	PutMsgRecOffset   int32
	ResponseRecOffset int32
	PutMsgRecPtr      C.MQPTR
	ResponseRecPtr    C.MQPTR

	OriginalMsgHandle C.MQHMSG
	NewMsgHandle      C.MQHMSG
	Action            int32
	PubLevel          int32
}

/*
NewMQPMO fills in default values for the MQPMO structure
*/
func NewMQPMO() *MQPMO {

	pmo := new(MQPMO)

	pmo.Version = int32(C.MQPMO_VERSION_1)
	pmo.Options = int32(C.MQPMO_NONE)
	pmo.Timeout = -1
	pmo.Context = 0
	pmo.KnownDestCount = 0
	pmo.UnknownDestCount = 0
	pmo.InvalidDestCount = 0
	pmo.ResolvedQName = ""
	pmo.ResolvedQMgrName = ""
	pmo.RecsPresent = 0
	pmo.PutMsgRecFields = 0
	pmo.PutMsgRecOffset = 0
	pmo.ResponseRecOffset = 0
	pmo.PutMsgRecPtr = nil
	pmo.ResponseRecPtr = nil

	pmo.OriginalMsgHandle = C.MQHM_NONE
	pmo.NewMsgHandle = C.MQHM_NONE
	pmo.Action = int32(C.MQACTP_NEW)
	pmo.PubLevel = 9

	return pmo
}

func copyPMOtoC(mqpmo *C.MQPMO, gopmo *MQPMO) {

	setMQIString((*C.char)(&mqpmo.StrucId[0]), "PMO ", 4)
	mqpmo.Version = C.MQLONG(gopmo.Version)

	mqpmo.Options = C.MQLONG(gopmo.Options)
	mqpmo.Timeout = C.MQLONG(gopmo.Timeout)
	mqpmo.Context = gopmo.Context
	mqpmo.KnownDestCount = C.MQLONG(gopmo.KnownDestCount)
	mqpmo.UnknownDestCount = C.MQLONG(gopmo.UnknownDestCount)
	mqpmo.InvalidDestCount = C.MQLONG(gopmo.InvalidDestCount)

	setMQIString((*C.char)(&mqpmo.ResolvedQName[0]), gopmo.ResolvedQName, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqpmo.ResolvedQMgrName[0]), gopmo.ResolvedQMgrName, C.MQ_OBJECT_NAME_LENGTH)

	mqpmo.RecsPresent = C.MQLONG(gopmo.RecsPresent)
	mqpmo.PutMsgRecFields = C.MQLONG(gopmo.PutMsgRecFields)
	mqpmo.PutMsgRecOffset = C.MQLONG(gopmo.PutMsgRecOffset)
	mqpmo.ResponseRecOffset = C.MQLONG(gopmo.ResponseRecOffset)
	mqpmo.PutMsgRecPtr = gopmo.PutMsgRecPtr
	mqpmo.ResponseRecPtr = gopmo.ResponseRecPtr

	mqpmo.OriginalMsgHandle = gopmo.OriginalMsgHandle
	mqpmo.NewMsgHandle = gopmo.NewMsgHandle
	mqpmo.Action = C.MQLONG(gopmo.Action)
	mqpmo.PubLevel = C.MQLONG(gopmo.PubLevel)

	return
}

func copyPMOfromC(mqpmo *C.MQPMO, gopmo *MQPMO) {

	gopmo.Version = int32(mqpmo.Version)

	gopmo.Options = int32(mqpmo.Options)
	gopmo.Timeout = int32(mqpmo.Timeout)
	gopmo.Context = mqpmo.Context
	gopmo.KnownDestCount = int32(mqpmo.KnownDestCount)
	gopmo.UnknownDestCount = int32(mqpmo.UnknownDestCount)
	gopmo.InvalidDestCount = int32(mqpmo.InvalidDestCount)

	gopmo.ResolvedQName = C.GoStringN((*C.char)(&mqpmo.ResolvedQName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gopmo.ResolvedQMgrName = C.GoStringN((*C.char)(&mqpmo.ResolvedQMgrName[0]), C.MQ_OBJECT_NAME_LENGTH)

	gopmo.RecsPresent = int32(mqpmo.RecsPresent)
	gopmo.PutMsgRecFields = int32(mqpmo.PutMsgRecFields)
	gopmo.PutMsgRecOffset = int32(mqpmo.PutMsgRecOffset)
	gopmo.ResponseRecOffset = int32(mqpmo.ResponseRecOffset)
	gopmo.PutMsgRecPtr = mqpmo.PutMsgRecPtr
	gopmo.ResponseRecPtr = mqpmo.ResponseRecPtr

	gopmo.OriginalMsgHandle = mqpmo.OriginalMsgHandle
	gopmo.NewMsgHandle = mqpmo.NewMsgHandle
	gopmo.Action = int32(mqpmo.Action)
	gopmo.PubLevel = int32(mqpmo.PubLevel)
	return
}
