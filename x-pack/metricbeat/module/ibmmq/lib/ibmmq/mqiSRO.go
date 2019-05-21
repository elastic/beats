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
This module contains the Subscription Request Options structure
*/

type MQSRO struct {
	Options int32
	NumPubs int32
}

func NewMQSRO() *MQSRO {
	sro := new(MQSRO)
	sro.Options = int32(C.MQSRO_NONE)
	sro.NumPubs = 0
	return sro
}

func copySROtoC(mqsro *C.MQSRO, gosro *MQSRO) {
	setMQIString((*C.char)(&mqsro.StrucId[0]), "SRO ", 4)
	mqsro.Version = 1
	mqsro.Options = C.MQLONG(gosro.Options) | C.MQSRO_FAIL_IF_QUIESCING
	mqsro.NumPubs = C.MQLONG(gosro.NumPubs)
}

func copySROfromC(mqsro *C.MQSRO, gosro *MQSRO) {
	gosro.Options = int32(mqsro.Options)
	gosro.NumPubs = int32(mqsro.NumPubs)
	return
}
