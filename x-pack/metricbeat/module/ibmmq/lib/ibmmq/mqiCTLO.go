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
MQCTLO is a structure containing the MQ Control Options
*/
type MQCTLO struct {
	ConnectionArea interface{}
	Options        int32
}

/*
NewMQCTLO creates a MQCTLO structure.
*/
func NewMQCTLO() *MQCTLO {
	ctlo := new(MQCTLO)
	ctlo.Options = C.MQCTLO_NONE
	ctlo.ConnectionArea = nil
	return ctlo
}

func copyCTLOtoC(mqctlo *C.MQCTLO, goctlo *MQCTLO) {
	setMQIString((*C.char)(&mqctlo.StrucId[0]), "CTLO", 4)
	mqctlo.Version = C.MQCTLO_VERSION_1
	mqctlo.Options = (C.MQLONG)(goctlo.Options) | C.MQCTLO_FAIL_IF_QUIESCING
	// Always pass NULL to the C function as the real array is saved/restored in the Go layer
	mqctlo.ConnectionArea = (C.MQPTR)(C.NULL)
	return
}

func copyCTLOfromC(mqctlo *C.MQCTLO, goctlo *MQCTLO) {
	// There are no output fields for this structure
	return
}
