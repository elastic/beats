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
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>

*/
import "C"
import "bytes"

/*
MQGMO is a structure containing the MQ Get Message Options (MQGMO)
*/
type MQGMO struct {
	Version        int32
	Options        int32
	WaitInterval   int32
	Signal1        int32
	Signal2        int32
	ResolvedQName  string
	MatchOptions   int32
	GroupStatus    rune
	SegmentStatus  rune
	Segmentation   rune
	Reserved1      rune
	MsgToken       []byte
	ReturnedLength int32
	Reserved2      int32
	MsgHandle      MQMessageHandle
}

/*
NewMQGMO fills in default values for the MQGMO structure
*/
func NewMQGMO() *MQGMO {

	gmo := new(MQGMO)
	gmo.Version = int32(C.MQGMO_VERSION_1)
	gmo.Options = int32(C.MQGMO_NO_WAIT + C.MQGMO_PROPERTIES_AS_Q_DEF)
	gmo.WaitInterval = int32(C.MQWI_UNLIMITED)
	gmo.Signal1 = 0
	gmo.Signal2 = 0
	gmo.ResolvedQName = ""
	gmo.MatchOptions = int32(C.MQMO_MATCH_MSG_ID + C.MQMO_MATCH_CORREL_ID)
	gmo.GroupStatus = rune(C.MQGS_NOT_IN_GROUP)
	gmo.SegmentStatus = rune(C.MQSS_NOT_A_SEGMENT)
	gmo.Segmentation = rune(C.MQSEG_INHIBITED)
	gmo.Reserved1 = ' '
	gmo.MsgToken = bytes.Repeat([]byte{0}, C.MQ_MSG_TOKEN_LENGTH)
	gmo.ReturnedLength = int32(C.MQRL_UNDEFINED)
	gmo.Reserved2 = 0
	gmo.MsgHandle.hMsg = C.MQHM_NONE

	return gmo
}

func copyGMOtoC(mqgmo *C.MQGMO, gogmo *MQGMO) {
	var i int

	setMQIString((*C.char)(&mqgmo.StrucId[0]), "GMO ", 4)
	mqgmo.Version = C.MQLONG(gogmo.Version)
	mqgmo.Options = C.MQLONG(gogmo.Options) | C.MQGMO_FAIL_IF_QUIESCING
	mqgmo.WaitInterval = C.MQLONG(gogmo.WaitInterval)
	mqgmo.Signal1 = C.MQLONG(gogmo.Signal1)
	mqgmo.Signal2 = C.MQLONG(gogmo.Signal2)
	setMQIString((*C.char)(&mqgmo.ResolvedQName[0]), gogmo.ResolvedQName, C.MQ_OBJECT_NAME_LENGTH)
	mqgmo.MatchOptions = C.MQLONG(gogmo.MatchOptions)
	mqgmo.GroupStatus = C.MQCHAR(gogmo.GroupStatus)
	mqgmo.SegmentStatus = C.MQCHAR(gogmo.SegmentStatus)
	mqgmo.Segmentation = C.MQCHAR(gogmo.Segmentation)
	mqgmo.Reserved1 = C.MQCHAR(gogmo.Reserved1)
	for i = 0; i < C.MQ_MSG_TOKEN_LENGTH; i++ {
		mqgmo.MsgToken[i] = C.MQBYTE(gogmo.MsgToken[i])
	}
	mqgmo.ReturnedLength = C.MQLONG(gogmo.ReturnedLength)
	mqgmo.Reserved2 = C.MQLONG(gogmo.Reserved2)
	if gogmo.MsgHandle.hMsg != C.MQHM_NONE {
		if mqgmo.Version < C.MQGMO_VERSION_4 {
			mqgmo.Version = C.MQGMO_VERSION_4
		}
		mqgmo.MsgHandle = gogmo.MsgHandle.hMsg
	}
	return
}

func copyGMOfromC(mqgmo *C.MQGMO, gogmo *MQGMO) {
	var i int

	gogmo.Version = int32(mqgmo.Version)
	gogmo.Options = int32(mqgmo.Options)
	gogmo.WaitInterval = int32(mqgmo.WaitInterval)
	gogmo.Signal1 = int32(mqgmo.Signal1)
	gogmo.Signal2 = int32(mqgmo.Signal2)
	gogmo.ResolvedQName = trimStringN((*C.char)(&mqgmo.ResolvedQName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gogmo.MatchOptions = int32(mqgmo.MatchOptions)
	gogmo.GroupStatus = rune(mqgmo.GroupStatus)
	gogmo.SegmentStatus = rune(mqgmo.SegmentStatus)
	gogmo.Segmentation = rune(mqgmo.Segmentation)
	gogmo.Reserved1 = rune(mqgmo.Reserved1)
	for i = 0; i < C.MQ_MSG_TOKEN_LENGTH; i++ {
		gogmo.MsgToken[i] = (byte)(mqgmo.MsgToken[i])
	}
	gogmo.ReturnedLength = int32(mqgmo.ReturnedLength)
	gogmo.Reserved2 = int32(mqgmo.Reserved2)
	if mqgmo.Version >= C.MQGMO_VERSION_4 {
		gogmo.MsgHandle.hMsg = mqgmo.MsgHandle
	}
	return
}
