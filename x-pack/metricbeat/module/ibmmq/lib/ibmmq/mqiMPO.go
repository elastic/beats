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
import "unsafe"

/*
This module contains the Message Property structures
*/

type MQIMPO struct {
	Options      int32
	ReturnedName string
	TypeString   string
}

type MQSMPO struct {
	Options int32
}

type MQDMPO struct {
	Options int32
}

type MQPD struct {
	Options     int32
	Support     int32
	Context     int32
	CopyOptions int32
}

func NewMQIMPO() *MQIMPO {
	impo := new(MQIMPO)
	impo.Options = int32(C.MQIMPO_NONE)
	impo.ReturnedName = ""
	impo.TypeString = ""

	return impo
}

func NewMQDMPO() *MQDMPO {
	dmpo := new(MQDMPO)
	dmpo.Options = int32(C.MQDMPO_DEL_FIRST)
	return dmpo
}

func NewMQSMPO() *MQSMPO {
	smpo := new(MQSMPO)
	smpo.Options = int32(C.MQSMPO_SET_FIRST)
	return smpo
}

func NewMQPD() *MQPD {
	pd := new(MQPD)
	pd.Options = int32(C.MQPD_NONE)
	pd.Support = int32(C.MQPD_SUPPORT_OPTIONAL)
	pd.Context = int32(C.MQPD_NO_CONTEXT)
	pd.CopyOptions = int32(C.MQCOPY_DEFAULT)
	return pd
}

func copyIMPOtoC(mqimpo *C.MQIMPO, goimpo *MQIMPO) {
	const vsbufsize = 10240
	setMQIString((*C.char)(&mqimpo.StrucId[0]), "IMPO", 4)
	mqimpo.Version = 1
	mqimpo.Options = C.MQLONG(goimpo.Options)
	mqimpo.RequestedEncoding = C.MQLONG(C.MQENC_NATIVE)
	mqimpo.RequestedCCSID = C.MQLONG(C.MQCCSI_APPL)
	mqimpo.ReturnedEncoding = C.MQLONG(C.MQENC_NATIVE)
	mqimpo.ReturnedCCSID = 0
	mqimpo.Reserved1 = 0

	mqimpo.ReturnedName.VSLength = 0
	mqimpo.ReturnedName.VSCCSID = C.MQCCSI_APPL
	mqimpo.ReturnedName.VSPtr = C.MQPTR(C.malloc(vsbufsize))
	mqimpo.ReturnedName.VSBufSize = vsbufsize

	setMQIString((*C.char)(&mqimpo.TypeString[0]), "", 8)
	return
}

func copyDMPOtoC(mqdmpo *C.MQDMPO, godmpo *MQDMPO) {
	setMQIString((*C.char)(&mqdmpo.StrucId[0]), "DMPO", 4)
	mqdmpo.Version = 1
	mqdmpo.Options = C.MQLONG(godmpo.Options)
}
func copySMPOtoC(mqsmpo *C.MQSMPO, gosmpo *MQSMPO) {
	setMQIString((*C.char)(&mqsmpo.StrucId[0]), "SMPO", 4)
	mqsmpo.Version = 1
	mqsmpo.Options = C.MQLONG(gosmpo.Options)
	mqsmpo.ValueEncoding = C.MQLONG(C.MQENC_NATIVE)
	mqsmpo.ValueCCSID = C.MQLONG(C.MQCCSI_APPL)
}
func copyPDtoC(mqpd *C.MQPD, gopd *MQPD) {
	setMQIString((*C.char)(&mqpd.StrucId[0]), "PD  ", 4)
	mqpd.Version = 1
	mqpd.Options = C.MQLONG(gopd.Options)
	mqpd.Support = C.MQLONG(gopd.Support)
	mqpd.Context = C.MQLONG(gopd.Context)
	mqpd.CopyOptions = C.MQLONG(gopd.CopyOptions)
}

func copyIMPOfromC(mqimpo *C.MQIMPO, goimpo *MQIMPO) {
	goimpo.Options = int32(mqimpo.Options)
	goimpo.TypeString = trimStringN((*C.char)(&mqimpo.TypeString[0]), 8)
	goimpo.ReturnedName = trimStringN((*C.char)(mqimpo.ReturnedName.VSPtr), (C.int)(mqimpo.ReturnedName.VSLength))
	C.free(unsafe.Pointer(mqimpo.ReturnedName.VSPtr))
	return
}

func copySMPOfromC(mqsmpo *C.MQSMPO, gosmpo *MQSMPO) {
	return
}

func copyDMPOfromC(mqdmpo *C.MQDMPO, godmpo *MQDMPO) {
	return
}

func copyPDfromC(mqpd *C.MQPD, gopd *MQPD) {
	return
}
