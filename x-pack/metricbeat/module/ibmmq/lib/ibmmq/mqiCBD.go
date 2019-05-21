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
MQCBD is a structure containing the MQ Callback Descriptor
*/
type MQCBD struct {
	CallbackType     int32
	Options          int32
	CallbackArea     interface{}
	CallbackFunction MQCB_FUNCTION
	CallbackName     string
	MaxMsgLength     int32
}

/*
NewMQCBD fills in default values for the MQCBD structure
*/
func NewMQCBD() *MQCBD {
	cbd := new(MQCBD)
	cbd.CallbackType = C.MQCBT_MESSAGE_CONSUMER
	cbd.Options = C.MQCBDO_NONE
	cbd.CallbackArea = nil
	cbd.CallbackFunction = nil
	cbd.CallbackName = ""
	cbd.MaxMsgLength = C.MQCBD_FULL_MSG_LENGTH

	return cbd
}

func copyCBDtoC(mqcbd *C.MQCBD, gocbd *MQCBD) {

	setMQIString((*C.char)(&mqcbd.StrucId[0]), "CBD ", 4)
	mqcbd.Version = C.MQCBD_VERSION_1

	mqcbd.CallbackType = C.MQLONG(gocbd.CallbackType)
	mqcbd.Options = C.MQLONG(gocbd.Options) | C.MQCBDO_FAIL_IF_QUIESCING
	// CallbackArea is always set to NULL here. The user's values are saved/restored elsewhere
	mqcbd.CallbackArea = (C.MQPTR)(C.NULL)

	setMQIString((*C.char)(&mqcbd.CallbackName[0]), gocbd.CallbackName, 128) // There's no MQI constant for the length

	mqcbd.MaxMsgLength = C.MQLONG(gocbd.MaxMsgLength)

	return
}

func copyCBDfromC(mqcbd *C.MQCBD, gocbd *MQCBD) {
	// There are no modified output parameters
	return
}
