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

import (
	"bytes"
	"unsafe"
)

/*
MQSD is a structure containing the MQ Subscription Descriptor (MQSD)
*/
type MQSD struct {
	Version int32
	Options int32

	ObjectName          string
	AlternateUserId     string
	AlternateSecurityId []byte
	SubExpiry           int32
	ObjectString        string
	SubName             string
	SubUserData         string
	SubCorrelId         []byte

	PubPriority        int32
	PubAccountingToken []byte

	PubApplIdentityData string

	SelectionString string
	SubLevel        int32
	ResObjectString string
}

/*
NewMQSD fills in default values for the MQSD structure
*/
func NewMQSD() *MQSD {

	sd := new(MQSD)

	sd.Version = int32(C.MQSD_VERSION_1)
	sd.Options = 0

	sd.ObjectName = ""
	sd.AlternateUserId = ""
	sd.AlternateSecurityId = bytes.Repeat([]byte{0}, C.MQ_SECURITY_ID_LENGTH)
	sd.SubExpiry = int32(C.MQEI_UNLIMITED)
	sd.ObjectString = ""
	sd.SubName = ""
	sd.SubUserData = ""
	sd.SubCorrelId = bytes.Repeat([]byte{0}, C.MQ_CORREL_ID_LENGTH)

	sd.PubPriority = int32(C.MQPRI_PRIORITY_AS_PUBLISHED)
	sd.PubAccountingToken = bytes.Repeat([]byte{0}, C.MQ_ACCOUNTING_TOKEN_LENGTH)

	sd.PubApplIdentityData = ""

	sd.SelectionString = ""
	sd.SubLevel = 1
	sd.ResObjectString = ""

	return sd
}

/*
It is expected that copyXXtoC and copyXXfromC will be called as
matching pairs. That means that we can handle the MQCHARV type
by allocating storage in the toC function and freeing it in fromC.
If the input string for an MQCHARV type is empty, then we allocate
a fixed length buffer for any potential output.

In the fromC function, that buffer is freed. Conveniently, we can
free it always, because if we didn't explicitly call malloc(), it was
allocated by C.CString and still needs to be freed.
*/
func copySDtoC(mqsd *C.MQSD, gosd *MQSD) {
	var i int
	const vsbufsize = 10240

	setMQIString((*C.char)(&mqsd.StrucId[0]), "SD  ", 4)
	mqsd.Version = C.MQLONG(gosd.Version)
	mqsd.Options = C.MQLONG(gosd.Options) | C.MQSO_FAIL_IF_QUIESCING

	setMQIString((*C.char)(&mqsd.ObjectName[0]), gosd.ObjectName, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqsd.AlternateUserId[0]), gosd.AlternateUserId, C.MQ_USER_ID_LENGTH)
	for i = 0; i < C.MQ_SECURITY_ID_LENGTH; i++ {
		mqsd.AlternateSecurityId[i] = C.MQBYTE(gosd.AlternateSecurityId[i])
	}
	mqsd.SubExpiry = C.MQLONG(gosd.SubExpiry)

	mqsd.ObjectString.VSLength = (C.MQLONG)(len(gosd.ObjectString))
	mqsd.ObjectString.VSCCSID = C.MQCCSI_APPL
	if mqsd.ObjectString.VSLength == 0 {
		mqsd.ObjectString.VSPtr = C.MQPTR(C.malloc(vsbufsize))
		mqsd.ObjectString.VSBufSize = vsbufsize
	} else {
		mqsd.ObjectString.VSPtr = (C.MQPTR)(C.CString(gosd.ObjectString))
	}

	mqsd.SubName.VSLength = (C.MQLONG)(len(gosd.SubName))
	mqsd.SubName.VSCCSID = C.MQCCSI_APPL
	if mqsd.SubName.VSLength == 0 {
		mqsd.SubName.VSPtr = C.MQPTR(C.malloc(vsbufsize))
		mqsd.SubName.VSBufSize = vsbufsize
	} else {
		mqsd.SubName.VSPtr = (C.MQPTR)(C.CString(gosd.SubName))
	}

	mqsd.SubUserData.VSLength = (C.MQLONG)(len(gosd.SubUserData))
	mqsd.SubUserData.VSCCSID = C.MQCCSI_APPL
	if mqsd.SubUserData.VSLength == 0 {
		mqsd.SubUserData.VSPtr = C.MQPTR(C.malloc(vsbufsize))
		mqsd.SubUserData.VSBufSize = vsbufsize
	} else {
		mqsd.SubUserData.VSPtr = (C.MQPTR)(C.CString(gosd.SubUserData))
	}

	for i = 0; i < C.MQ_CORREL_ID_LENGTH; i++ {
		mqsd.SubCorrelId[i] = C.MQBYTE(gosd.SubCorrelId[i])
	}

	mqsd.PubPriority = C.MQLONG(gosd.PubPriority)
	for i = 0; i < C.MQ_ACCOUNTING_TOKEN_LENGTH; i++ {
		mqsd.PubAccountingToken[i] = C.MQBYTE(gosd.PubAccountingToken[i])
	}

	setMQIString((*C.char)(&mqsd.PubApplIdentityData[0]), gosd.PubApplIdentityData, C.MQ_APPL_IDENTITY_DATA_LENGTH)

	mqsd.SelectionString.VSLength = (C.MQLONG)(len(gosd.SelectionString))
	mqsd.SelectionString.VSCCSID = C.MQCCSI_APPL
	if mqsd.SelectionString.VSLength == 0 {
		mqsd.SelectionString.VSPtr = C.MQPTR(C.malloc(vsbufsize))
		mqsd.SelectionString.VSBufSize = vsbufsize
	} else {
		mqsd.SelectionString.VSPtr = (C.MQPTR)(C.CString(gosd.SelectionString))
	}

	mqsd.SubLevel = C.MQLONG(gosd.SubLevel)

	mqsd.ResObjectString.VSLength = (C.MQLONG)(len(gosd.ResObjectString))
	mqsd.ResObjectString.VSCCSID = C.MQCCSI_APPL
	if mqsd.ResObjectString.VSLength == 0 {
		mqsd.ResObjectString.VSPtr = C.MQPTR(C.malloc(vsbufsize))
		mqsd.ResObjectString.VSBufSize = vsbufsize
	} else {
		mqsd.ResObjectString.VSPtr = (C.MQPTR)(C.CString(gosd.ResObjectString))
	}
	return
}

func copySDfromC(mqsd *C.MQSD, gosd *MQSD) {
	var i int
	gosd.Version = int32(mqsd.Version)
	gosd.Options = int32(mqsd.Options)

	gosd.ObjectName = trimStringN((*C.char)(&mqsd.ObjectName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gosd.AlternateUserId = trimStringN((*C.char)(&mqsd.AlternateUserId[0]), C.MQ_USER_ID_LENGTH)
	for i := 0; i < C.MQ_SECURITY_ID_LENGTH; i++ {
		gosd.AlternateSecurityId[i] = (byte)(mqsd.AlternateSecurityId[i])
	}
	gosd.SubExpiry = int32(mqsd.SubExpiry)

	gosd.ObjectString = trimStringN((*C.char)(mqsd.ObjectString.VSPtr), (C.int)(mqsd.ObjectString.VSLength))
	C.free(unsafe.Pointer(mqsd.ObjectString.VSPtr))
	gosd.SubName = trimStringN((*C.char)(mqsd.SubName.VSPtr), (C.int)(mqsd.SubName.VSLength))
	C.free(unsafe.Pointer(mqsd.SubName.VSPtr))
	gosd.SubUserData = trimStringN((*C.char)(mqsd.SubUserData.VSPtr), (C.int)(mqsd.SubUserData.VSLength))
	C.free(unsafe.Pointer(mqsd.SubUserData.VSPtr))

	for i = 0; i < C.MQ_CORREL_ID_LENGTH; i++ {
		gosd.SubCorrelId[i] = (byte)(mqsd.SubCorrelId[i])
	}

	gosd.PubPriority = int32(mqsd.PubPriority)
	for i = 0; i < C.MQ_ACCOUNTING_TOKEN_LENGTH; i++ {
		gosd.PubAccountingToken[i] = (byte)(mqsd.PubAccountingToken[i])
	}

	gosd.PubApplIdentityData = trimStringN((*C.char)(&mqsd.PubApplIdentityData[0]), C.MQ_APPL_IDENTITY_DATA_LENGTH)

	gosd.SelectionString = trimStringN((*C.char)(mqsd.SelectionString.VSPtr), (C.int)(mqsd.SelectionString.VSLength))
	C.free(unsafe.Pointer(mqsd.SelectionString.VSPtr))

	gosd.SubLevel = int32(mqsd.SubLevel)

	gosd.ResObjectString = trimStringN((*C.char)(mqsd.ResObjectString.VSPtr), (C.int)(mqsd.ResObjectString.VSLength))
	C.free(unsafe.Pointer(mqsd.ResObjectString.VSPtr))
	return
}
