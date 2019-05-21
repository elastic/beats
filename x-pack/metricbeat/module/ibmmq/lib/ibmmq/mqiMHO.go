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
MQCMHO is a structure containing the MQ Create Message Handle Options
*/
type MQCMHO struct {
	Options int32
}

/*
MQDMHO is a structure containing the MQ Delete Message Handle Options
*/
type MQDMHO struct {
	Options int32
}

/*
NewMQCMHO fills in default values for the MQCMHO structure
*/
func NewMQCMHO() *MQCMHO {

	cmho := new(MQCMHO)
	cmho.Options = int32(C.MQCMHO_DEFAULT_VALIDATION)

	return cmho
}

func copyCMHOtoC(mqcmho *C.MQCMHO, gocmho *MQCMHO) {
	setMQIString((*C.char)(&mqcmho.StrucId[0]), "CMHO", 4)
	mqcmho.Version = C.MQCMHO_VERSION_1
	mqcmho.Options = C.MQLONG(gocmho.Options)
	return
}

func copyCMHOfromC(mqcmho *C.MQCMHO, gocmho *MQCMHO) {
	gocmho.Options = int32(mqcmho.Options)
	return
}

/*
NewMQDMHO fills in default values for the MQDMHO structure
*/
func NewMQDMHO() *MQDMHO {
	dmho := new(MQDMHO)
	dmho.Options = int32(C.MQDMHO_NONE)
	return dmho
}

func copyDMHOtoC(mqdmho *C.MQDMHO, godmho *MQDMHO) {
	setMQIString((*C.char)(&mqdmho.StrucId[0]), "DMHO", 4)
	mqdmho.Version = C.MQDMHO_VERSION_1
	mqdmho.Options = C.MQLONG(godmho.Options)
	return
}

func copyDMHOfromC(mqdmho *C.MQDMHO, godmho *MQDMHO) {
	godmho.Options = int32(mqdmho.Options)
	return
}
