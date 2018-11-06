/*
Package ibmmq provides a wrapper to a subset of the IBM MQ
procedural interface (the MQI).

In this initial implementation not all the MQI verbs are
included, but it does have the core operations required to
put and get messages and work with topics.

The verbs are given mixed case names without MQ - Open instead
of MQOPEN etc.

If an MQI call returns MQCC_FAILED or MQCC_WARNING, a custom error
type is returned containing the MQCC/MQRC values as
a formatted string. Use mqreturn:= err(*ibmmq.MQReturn) to access
the particular MQRC or MQCC values.

The build directives for Windows assume the header and library files have
been copied to a temporary location, because the default paths are not
acceptable to Go (it does not like spaces or special characters like ~).
Note: This problem appears to have been fixed in Go 1.9, and once that
is fully available, the directives will be changed to a more reasonable
path in this file. For example
    cgo windows CFLAGS -I"c:/Program Files/IBM/MQ/tools/c/include" -m64

The build directives for Linux assume the default MQ installation path
in /opt/mqm. These would need to be changed in this file if you use a
non-default path.
*/
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
  See the License for the specific

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
#cgo !windows CFLAGS: -I/opt/mqm/inc -D_REENTRANT
#cgo windows CFLAGS:  -I"C:/Program Files/IBM/MQ/Tools/c/include"
#cgo !windows LDFLAGS: -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath=/opt/mqm/lib64 -Wl,-rpath=/usr/lib64
#cgo windows LDFLAGS: -L "C:/Program Files/IBM/MQ/bin64" -lmqm

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqxc.h>
*/
import "C"

import (
	"encoding/binary"
	"unsafe"
)

/*
   This file contains the C wrappers, calling out to structure-specific
   functions where necessary.

   Define some basic types to hold the
   references to MQ objects - hconn, hobj - and
   a simple way to pass the combination of MQCC/MQRC
   returned from MQI verbs

   The object name is copied into the structures only
   for convenience. It's not really needed, but
   it can sometimes be nice to print which queue an hObj
   refers to during debug.
*/

/*
MQQueueManager contains the connection to the queue manager
*/
type MQQueueManager struct {
	hConn C.MQHCONN
	Name  string
}

/*
MQObject contains a reference to an open object and the associated
queue manager
*/
type MQObject struct {
	hObj C.MQHOBJ
	qMgr *MQQueueManager
	Name string
}

/*
MQReturn holds the MQRC and MQCC values returned from an MQI verb. It
implements the Error() function so is returned as the specific error
from the verbs. See the sample programs for how to access the
MQRC/MQCC values in this returned error.
*/
type MQReturn struct {
	MQCC int32
	MQRC int32
	verb string
}

func (e *MQReturn) Error() string {
	return mqstrerror(e.verb, C.MQLONG(e.MQCC), C.MQLONG(e.MQRC))
}

/*
 * Copy a Go string into a fixed-size C char array such as MQCHAR12
 * Once the string has been copied, it can be immediately freed
 * Empty strings have first char set to 0 in MQI structures
 */
func setMQIString(a *C.char, v string, l int) {
	if len(v) > 0 {
		p := C.CString(v)
		C.strncpy(a, p, (C.size_t)(l))
		C.free(unsafe.Pointer(p))
	} else {
		*a = 0
	}
}

/*
Conn is the function to connect to a queue manager
*/
func Conn(goQMgrName string) (MQQueueManager, error) {
	return Connx(goQMgrName, nil)
}

/*
Connx is the extended function to connect to a queue manager.
*/
func Connx(goQMgrName string, gocno *MQCNO) (MQQueueManager, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqcno C.MQCNO

	if (C.MQENC_NATIVE % 2) == 0 { // May be needed for conversion later
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	qMgr := MQQueueManager{}
	mqQMgrName := unsafe.Pointer(C.CString(goQMgrName))
	defer C.free(mqQMgrName)

	// Set up a default CNO if not provided.
	if gocno == nil {
		// Because Go programs are always threaded, and we cannot
		// tell on which thread we might get dispatched, allow handles always to
		// be shareable.
		gocno = NewMQCNO()
		gocno.Options = MQCNO_HANDLE_SHARE_NO_BLOCK
	} else {
		if (gocno.Options & (MQCNO_HANDLE_SHARE_NO_BLOCK |
			MQCNO_HANDLE_SHARE_BLOCK)) == 0 {
			gocno.Options |= MQCNO_HANDLE_SHARE_NO_BLOCK
		}
	}
	copyCNOtoC(&mqcno, gocno)

	C.MQCONNX((*C.MQCHAR)(mqQMgrName), &mqcno, &qMgr.hConn, &mqcc, &mqrc)

	if gocno != nil {
		copyCNOfromC(&mqcno, gocno)
	}

	mqreturn := &MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCONNX",
	}

	if mqcc != C.MQCC_OK {
		return qMgr, mqreturn
	}

	qMgr.Name = goQMgrName

	return qMgr, nil
}

/*
Disc is the function to disconnect from the queue manager
*/
func (x *MQQueueManager) Disc() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	C.MQDISC(&x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQDISC",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

/*
Open an object such as a queue or topic
*/
func (x *MQQueueManager) Open(good *MQOD, goOpenOptions int32) (MQObject, error) {

	return x.OpenOnRemoteQmgr(good, goOpenOptions, x)

}

func (x *MQQueueManager) OpenOnRemoteQmgr(good *MQOD, goOpenOptions int32, targetQmgr *MQQueueManager) (MQObject, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqod C.MQOD
	var mqOpenOptions C.MQLONG

	object := MQObject{
		Name: good.ObjectName,
		qMgr: targetQmgr,
	}

	copyODtoC(&mqod, good)
	mqOpenOptions = C.MQLONG(goOpenOptions)

	C.MQOPEN(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqod)),
		mqOpenOptions,
		&object.hObj,
		&mqcc,
		&mqrc)

	copyODfromC(&mqod, good)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQOPEN",
	}

	if mqcc != C.MQCC_OK {
		return object, &mqreturn
	}

	// ObjectName may have changed because it's a model queue
	object.Name = good.ObjectName

	return object, nil

}

/*
Close the object
*/
func (object *MQObject) Close(goCloseOptions int32) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqCloseOptions C.MQLONG

	mqCloseOptions = C.MQLONG(goCloseOptions)

	C.MQCLOSE(object.qMgr.hConn, &object.hObj, mqCloseOptions, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCLOSE",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Sub is the function to subscribe to a topic
*/
func (x *MQQueueManager) Sub(gosd *MQSD, qObject *MQObject) (MQObject, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqsd C.MQSD

	subObject := MQObject{
		Name: gosd.ObjectName,
		qMgr: x,
	}

	copySDtoC(&mqsd, gosd)

	C.MQSUB(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqsd)),
		&qObject.hObj,
		&subObject.hObj,
		&mqcc,
		&mqrc)

	copySDfromC(&mqsd, gosd)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSUB",
	}

	if mqcc != C.MQCC_OK {
		return subObject, &mqreturn
	}

	qObject.qMgr = x // Force the correct hConn for managed objects

	return subObject, nil

}

/*
Cmit is the function to commit an in-flight transaction
*/
func (x *MQQueueManager) Cmit() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	C.MQCMIT(x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCMIT",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Back is the function to backout an in-flight transaction
*/
func (x *MQQueueManager) Back() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	C.MQBACK(x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQBACK",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Put a message to a queue or publish to a topic
*/
func (object MQObject) Put(gomd *MQMD,
	gopmo *MQPMO, buffer []byte) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqpmo C.MQPMO
	var ptr C.PMQVOID

	bufflen := len(buffer)

	copyMDtoC(&mqmd, gomd)
	copyPMOtoC(&mqpmo, gopmo)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQPUT(object.qMgr.hConn, object.hObj, (C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqpmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&mqcc, &mqrc)

	copyMDfromC(&mqmd, gomd)
	copyPMOfromC(&mqpmo, gopmo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQPUT",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

/*
Put1 puts a single messsage to a queue or topic. Typically used for one-shot
replies where it can be cheaper than multiple Open/Put/Close
sequences
*/
func (x *MQQueueManager) Put1(good *MQOD, gomd *MQMD,
	gopmo *MQPMO, buffer []byte) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqpmo C.MQPMO
	var mqod C.MQOD
	var ptr C.PMQVOID

	copyODtoC(&mqod, good)
	copyMDtoC(&mqmd, gomd)
	copyPMOtoC(&mqpmo, gopmo)

	bufflen := len(buffer)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQPUT1(x.hConn, (C.PMQVOID)(unsafe.Pointer(&mqod)),
		(C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqpmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&mqcc, &mqrc)

	copyODfromC(&mqod, good)
	copyMDfromC(&mqmd, gomd)
	copyPMOfromC(&mqpmo, gopmo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQPUT1",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Get a message from a queue
The length of the retrieved message is returned.
*/
func (object MQObject) Get(gomd *MQMD,
	gogmo *MQGMO, buffer []byte) (int, error) {

	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqgmo C.MQGMO
	var datalen C.MQLONG
	var ptr C.PMQVOID

	bufflen := len(buffer)

	copyMDtoC(&mqmd, gomd)
	copyGMOtoC(&mqgmo, gogmo)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQGET(object.qMgr.hConn, object.hObj, (C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqgmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&datalen,
		&mqcc, &mqrc)

	godatalen := int(datalen)
	copyMDfromC(&mqmd, gomd)
	copyGMOfromC(&mqgmo, gogmo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQGET",
	}

	if mqcc != C.MQCC_OK {
		return 0, &mqreturn
	}

	return godatalen, nil

}

/*
Inq is the function to inquire on an attribute of an object

Slices are returned containing the integer attributes, and all the
strings concatenated into a single buffer - the caller needs to know
how long each field in that buffer will be.

The caller passes in how many integer selectors are expected to be
returned, as well as the maximum length of the char buffer to be returned
*/
func (object MQObject) Inq(goSelectors []int32, intAttrCount int, charAttrLen int) ([]int32,
	[]byte, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqCharAttrs C.PMQCHAR
	var goCharAttrs []byte
	var goIntAttrs []int32
	var ptr C.PMQLONG

	if intAttrCount > 0 {
		goIntAttrs = make([]int32, intAttrCount)
		ptr = (C.PMQLONG)(unsafe.Pointer(&goIntAttrs[0]))
	} else {
		ptr = nil
	}
	if charAttrLen > 0 {
		mqCharAttrs = (C.PMQCHAR)(C.malloc(C.size_t(charAttrLen)))
		defer C.free(unsafe.Pointer(mqCharAttrs))
	} else {
		mqCharAttrs = nil
	}

	// Pass in the selectors directly
	C.MQINQ(object.qMgr.hConn, object.hObj,
		C.MQLONG(len(goSelectors)),
		C.PMQLONG(unsafe.Pointer(&goSelectors[0])),
		C.MQLONG(intAttrCount),
		ptr,
		C.MQLONG(charAttrLen),
		mqCharAttrs,
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQINQ",
	}

	if mqcc != C.MQCC_OK {
		return nil, nil, &mqreturn
	}

	if charAttrLen > 0 {
		goCharAttrs = C.GoBytes(unsafe.Pointer(mqCharAttrs), C.int(charAttrLen))
	}

	return goIntAttrs, goCharAttrs, nil
}
