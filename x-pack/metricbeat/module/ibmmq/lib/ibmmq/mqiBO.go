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
This module contains the Begin Options structure
*/

type MQBO struct {
	Options int32
}

func NewMQBO() *MQBO {
	bo := new(MQBO)
	bo.Options = int32(C.MQBO_NONE)
	return bo
}

func copyBOtoC(mqbo *C.MQBO, gobo *MQBO) {
	setMQIString((*C.char)(&mqbo.StrucId[0]), "BO  ", 4)
	mqbo.Version = 1
	mqbo.Options = C.MQLONG(gobo.Options)
}

func copyBOfromC(mqbo *C.MQBO, gobo *MQBO) {
	gobo.Options = int32(mqbo.Options)
	return
}
