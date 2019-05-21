package ibmmqi

/*
  Copyright (c) IBM Corporation 2018

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

/*
MQCBC is a structure containing the MQ Callback Context
The CompCode and Reason in the C structure are not included here. They
are set in an independent MQReturn structure passed to the callback. Similarly
for the hObj
*/
type MQCBC struct {
	CallType       int32
	CallbackArea   interface{} // These fields are saved/restored in parent function
	ConnectionArea interface{}
	State          int32
	DataLength     int32
	BufferLength   int32
	Flags          int32
	ReconnectDelay int32
}

/*
NewMQCBC creates a MQCBC structure. There are no default values
as the structure is created within MQ.
*/
func NewMQCBC() *MQCBC {
	cbc := new(MQCBC)
	return cbc
}

/*
Since we do not create the structure, there's no conversion for it into
a C format
*/
func copyCBCtoC(mqcbc *C.MQCBC, gocbc *MQCBC) {
	return
}

/*
But we do need a conversion process from C
*/
func copyCBCfromC(mqcbc *C.MQCBC, gocbc *MQCBC) {
	gocbc.CallType = int32(mqcbc.CallType)
	gocbc.State = int32(mqcbc.State)
	gocbc.DataLength = int32(mqcbc.DataLength)
	gocbc.BufferLength = int32(mqcbc.BufferLength)
	gocbc.Flags = int32(mqcbc.Flags)
	gocbc.ReconnectDelay = int32(mqcbc.ReconnectDelay)
	// ConnectionArea and CallbackArea are restored outside this function

	return
}
