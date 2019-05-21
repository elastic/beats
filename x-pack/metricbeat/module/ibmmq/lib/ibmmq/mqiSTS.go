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
import (
	"unsafe"
)

/*
MQSTS is a structure containing the MQ Status Information Record
*/
type MQSTS struct {
	CompCode           int32
	Reason             int32
	PutSuccessCount    int32
	PutWarningCount    int32
	PutFailureCount    int32
	ObjectType         int32
	ObjectName         string
	ObjectQMgrName     string
	ResolvedObjectName string
	ResolvedQMgrName   string
	ObjectString       string
	SubName            string
	OpenOptions        int32
	SubOptions         int32
}

/*
NewMQSTS fills in default values for the MQSTS structure
*/
func NewMQSTS() *MQSTS {

	sts := new(MQSTS)
	sts.CompCode = int32(C.MQCC_OK)
	sts.Reason = int32(C.MQRC_NONE)
	sts.PutSuccessCount = 0
	sts.PutWarningCount = 0
	sts.PutFailureCount = 0
	sts.ObjectType = int32(C.MQOT_Q)
	sts.ObjectName = ""
	sts.ObjectQMgrName = ""
	sts.ResolvedObjectName = ""
	sts.ResolvedQMgrName = ""
	sts.ObjectString = ""
	sts.SubName = ""
	sts.OpenOptions = 0
	sts.SubOptions = 0

	return sts
}

func copySTStoC(mqsts *C.MQSTS, gosts *MQSTS) {
	const vsbufsize = C.MQ_TOPIC_STR_LENGTH

	setMQIString((*C.char)(&mqsts.StrucId[0]), "STS ", 4)
	mqsts.Version = C.MQSTS_VERSION_2

	mqsts.PutSuccessCount = C.MQLONG(gosts.PutSuccessCount)
	mqsts.PutWarningCount = C.MQLONG(gosts.PutWarningCount)
	mqsts.PutFailureCount = C.MQLONG(gosts.PutFailureCount)
	mqsts.ObjectType = C.MQLONG(gosts.ObjectType)

	setMQIString((*C.char)(&mqsts.ObjectName[0]), gosts.ObjectName, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqsts.ObjectQMgrName[0]), gosts.ObjectQMgrName, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqsts.ResolvedObjectName[0]), gosts.ResolvedObjectName, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqsts.ResolvedQMgrName[0]), gosts.ResolvedQMgrName, C.MQ_OBJECT_NAME_LENGTH)

	if gosts.ObjectString != "" {

		mqsts.ObjectString.VSLength = (C.MQLONG)(len(gosts.ObjectString))
		mqsts.ObjectString.VSCCSID = C.MQCCSI_APPL
		if mqsts.ObjectString.VSLength == 0 {
			mqsts.ObjectString.VSPtr = C.MQPTR(C.malloc(vsbufsize))
			mqsts.ObjectString.VSBufSize = vsbufsize
		} else {
			mqsts.ObjectString.VSPtr = (C.MQPTR)(C.CString(gosts.ObjectString))
		}
	}
	if gosts.SubName != "" {

		mqsts.SubName.VSLength = (C.MQLONG)(len(gosts.SubName))
		mqsts.SubName.VSCCSID = C.MQCCSI_APPL
		if mqsts.SubName.VSLength == 0 {
			mqsts.SubName.VSPtr = C.MQPTR(C.malloc(vsbufsize))
			mqsts.SubName.VSBufSize = vsbufsize
		} else {
			mqsts.SubName.VSPtr = (C.MQPTR)(C.CString(gosts.SubName))
		}
	}

	mqsts.OpenOptions = C.MQLONG(gosts.OpenOptions)
	mqsts.SubOptions = C.MQLONG(gosts.SubOptions)

	return
}

func copySTSfromC(mqsts *C.MQSTS, gosts *MQSTS) {

	gosts.CompCode = int32(mqsts.CompCode)
	gosts.Reason = int32(mqsts.Reason)
	gosts.PutSuccessCount = int32(mqsts.PutSuccessCount)
	gosts.PutWarningCount = int32(mqsts.PutWarningCount)
	gosts.PutFailureCount = int32(mqsts.PutFailureCount)
	gosts.ObjectType = int32(mqsts.ObjectType)

	gosts.ObjectName = trimStringN((*C.char)(&mqsts.ObjectName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gosts.ObjectQMgrName = trimStringN((*C.char)(&mqsts.ObjectQMgrName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gosts.ResolvedObjectName = trimStringN((*C.char)(&mqsts.ResolvedObjectName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gosts.ResolvedQMgrName = trimStringN((*C.char)(&mqsts.ResolvedQMgrName[0]), C.MQ_OBJECT_NAME_LENGTH)

	if mqsts.Version >= C.MQSTS_VERSION_2 {
		gosts.ObjectString = trimStringN((*C.char)(mqsts.ObjectString.VSPtr), (C.int)(mqsts.ObjectString.VSLength))
		C.free(unsafe.Pointer(mqsts.ObjectString.VSPtr))

		gosts.SubName = trimStringN((*C.char)(mqsts.SubName.VSPtr), (C.int)(mqsts.SubName.VSLength))
		C.free(unsafe.Pointer(mqsts.SubName.VSPtr))

		gosts.OpenOptions = int32(mqsts.OpenOptions)
		gosts.SubOptions = int32(mqsts.SubOptions)
	}

	return
}
